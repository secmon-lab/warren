package cli

// chat-session-redesign Phase 7 CLI wrappers. These functions bridge the
// migrate CLI command to the business-logic Jobs in
// pkg/usecase/migration. Each wrapper:
//
//   1. Constructs a Firestore-backed Repository for the target database.
//   2. Builds the concrete Job with its dependencies.
//   3. Invokes Run with Options{DryRun: dryRun} and logs the Result.
//
// Repository construction uses the pkg/repository package's existing
// Firestore factory so authentication and error wrapping match the rest
// of the application.

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/session"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/usecase/migration"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"google.golang.org/api/iterator"
)

// openFirestoreRepository opens a Firestore-backed Repository for the
// given project/database. Callers must defer the returned cleanup.
func openFirestoreRepository(ctx context.Context, projectID, databaseID string) (interfaces.Repository, func(), error) {
	repo, err := repository.NewFirestore(ctx, projectID, databaseID)
	if err != nil {
		return nil, func() {}, goerr.Wrap(err, "failed to open firestore repository")
	}
	cleanup := func() {
		if closer, ok := any(repo).(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}
	return repo, cleanup, nil
}

// listAllSessions enumerates every Session document in the `sessions`
// collection for backfill-style jobs. The Firestore repository keeps
// its own `sessions` collection so the low-level iterator is colocated
// here to avoid leaking it through the Repository interface.
func listAllSessions(ctx context.Context, projectID, databaseID string) ([]*sessModel.Session, error) {
	db, err := firestore.NewClientWithDatabase(ctx, projectID, databaseID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to open firestore client")
	}
	defer safe.Close(ctx, db)

	iter := db.Collection("sessions").Documents(ctx)
	var out []*sessModel.Session
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate sessions")
		}
		var s sessModel.Session
		if err := doc.DataTo(&s); err != nil {
			return nil, goerr.Wrap(err, "failed to decode session",
				goerr.V("doc_id", doc.Ref.ID))
		}
		out = append(out, &s)
	}
	return out, nil
}

// firestoreCommentSource satisfies migration.LegacyCommentStore using
// the raw Firestore client directly, never going through the Repository
// interface. This keeps every legacy-Comment code path confined to the
// migration wrapper: the main application's Repository no longer knows
// `ticket.Comment` exists.
type firestoreCommentSource struct {
	projectID  string
	databaseID string
}

const (
	firestoreTicketsCollection  = "tickets"
	firestoreCommentsCollection = "comments"
)

func (f firestoreCommentSource) openClient(ctx context.Context) (*firestore.Client, error) {
	db, err := firestore.NewClientWithDatabase(ctx, f.projectID, f.databaseID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to open firestore client")
	}
	return db, nil
}

func (f firestoreCommentSource) ListTicketsWithComments(ctx context.Context) ([]*ticket.Ticket, error) {
	db, err := f.openClient(ctx)
	if err != nil {
		return nil, err
	}
	defer safe.Close(ctx, db)

	iter := db.Collection(firestoreTicketsCollection).Documents(ctx)
	var out []*ticket.Ticket
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate tickets")
		}
		var t ticket.Ticket
		if err := doc.DataTo(&t); err != nil {
			return nil, goerr.Wrap(err, "failed to decode ticket",
				goerr.V("doc_id", doc.Ref.ID))
		}
		out = append(out, &t)
	}
	return out, nil
}

func (f firestoreCommentSource) GetTicketComments(ctx context.Context, ticketID types.TicketID) ([]ticket.Comment, error) {
	db, err := f.openClient(ctx)
	if err != nil {
		return nil, err
	}
	defer safe.Close(ctx, db)

	iter := db.Collection(firestoreTicketsCollection).Doc(ticketID.String()).
		Collection(firestoreCommentsCollection).
		Documents(ctx)
	var out []ticket.Comment
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate comments",
				goerr.V("ticket_id", ticketID))
		}
		var c ticket.Comment
		if err := doc.DataTo(&c); err != nil {
			return nil, goerr.Wrap(err, "failed to decode comment",
				goerr.V("doc_id", doc.Ref.ID))
		}
		out = append(out, c)
	}
	return out, nil
}

func (f firestoreCommentSource) DeleteTicketComment(ctx context.Context, ticketID types.TicketID, commentID types.CommentID) error {
	db, err := f.openClient(ctx)
	if err != nil {
		return err
	}
	defer safe.Close(ctx, db)

	_, err = db.Collection(firestoreTicketsCollection).Doc(ticketID.String()).
		Collection(firestoreCommentsCollection).Doc(commentID.String()).
		Delete(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to delete comment",
			goerr.V("ticket_id", ticketID), goerr.V("comment_id", commentID))
	}
	return nil
}

func runSessionSourceBackfill(ctx context.Context, rt *migrateRuntime) error {
	logger := logging.From(ctx)

	repo, cleanup, err := openFirestoreRepository(ctx, rt.projectID, rt.databaseID)
	if err != nil {
		return err
	}
	defer cleanup()

	listAll := func(ctx context.Context) ([]*sessModel.Session, error) {
		return listAllSessions(ctx, rt.projectID, rt.databaseID)
	}

	job := migration.NewSessionSourceBackfillJob(repo, listAll)
	result, err := job.Run(ctx, migration.Options{DryRun: rt.dryRun})
	if err != nil {
		return err
	}
	logger.Info("session-source-backfill complete",
		"scanned", result.Scanned,
		"migrated", result.Migrated,
		"skipped", result.Skipped,
		"errors", result.Errors,
		"details", result.Details,
	)
	return nil
}

func runCommentToMessage(ctx context.Context, rt *migrateRuntime) error {
	logger := logging.From(ctx)
	repo, cleanup, err := openFirestoreRepository(ctx, rt.projectID, rt.databaseID)
	if err != nil {
		return err
	}
	defer cleanup()

	source := firestoreCommentSource{projectID: rt.projectID, databaseID: rt.databaseID}
	resolver := session.NewResolver(repo)
	job := migration.NewCommentToMessageJob(source, resolver, repo, repo)
	result, err := job.Run(ctx, migration.Options{DryRun: rt.dryRun})
	if err != nil {
		return err
	}
	logger.Info("comment-to-message complete",
		"scanned", result.Scanned,
		"migrated", result.Migrated,
		"skipped", result.Skipped,
		"errors", result.Errors,
	)
	return nil
}

func runTurnSynthesis(ctx context.Context, rt *migrateRuntime) error {
	logger := logging.From(ctx)
	repo, cleanup, err := openFirestoreRepository(ctx, rt.projectID, rt.databaseID)
	if err != nil {
		return err
	}
	defer cleanup()

	listAll := func(ctx context.Context) ([]*sessModel.Session, error) {
		return listAllSessions(ctx, rt.projectID, rt.databaseID)
	}

	job := migration.NewTurnSynthesisJob(repo, listAll)
	result, err := job.Run(ctx, migration.Options{DryRun: rt.dryRun})
	if err != nil {
		return err
	}
	logger.Info("turn-synthesis complete",
		"scanned", result.Scanned,
		"migrated", result.Migrated,
		"skipped", result.Skipped,
		"errors", result.Errors,
	)
	return nil
}

// buildStorageService constructs a storage.Service for Phase 7 jobs
// that copy or delete GCS-backed data. Returns an error when the
// storage bucket was not configured on the migrate command.
func buildStorageService(ctx context.Context, rt *migrateRuntime) (*storage.Service, error) {
	if rt.storage == nil || !rt.storage.IsConfigured() {
		return nil, goerr.New("storage bucket is required for this migration (set --storage-bucket)")
	}
	client, err := rt.storage.Configure(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to configure storage client")
	}
	return storage.New(client, storage.WithPrefix(rt.storage.Prefix())), nil
}

func runHistoryScope(ctx context.Context, rt *migrateRuntime) error {
	logger := logging.From(ctx)
	storageSvc, err := buildStorageService(ctx, rt)
	if err != nil {
		return err
	}

	listAll := func(ctx context.Context) ([]*sessModel.Session, error) {
		return listAllSessions(ctx, rt.projectID, rt.databaseID)
	}

	job := migration.NewHistoryScopeJob(storageSvc, listAll)
	result, err := job.Run(ctx, migration.Options{DryRun: rt.dryRun})
	if err != nil {
		return err
	}
	logger.Info("history-scope complete",
		"scanned", result.Scanned,
		"migrated", result.Migrated,
		"skipped", result.Skipped,
		"errors", result.Errors,
	)
	return nil
}

func runCleanupLegacy(ctx context.Context, rt *migrateRuntime) error {
	logger := logging.From(ctx)

	// Storage is optional for cleanup-legacy; when absent, only
	// Firestore ticket_comments are purged.
	var storageSvc *storage.Service
	if rt.storage != nil && rt.storage.IsConfigured() {
		svc, err := buildStorageService(ctx, rt)
		if err != nil {
			return err
		}
		storageSvc = svc
	}

	store := firestoreCommentSource{projectID: rt.projectID, databaseID: rt.databaseID}
	listSessions := func(ctx context.Context) ([]*sessModel.Session, error) {
		return listAllSessions(ctx, rt.projectID, rt.databaseID)
	}

	job := migration.NewCleanupLegacyJob(store, storageSvc, listSessions)
	result, err := job.Run(ctx, migration.Options{DryRun: rt.dryRun})
	if err != nil {
		return err
	}
	logger.Info("cleanup-legacy complete",
		"scanned", result.Scanned,
		"migrated", result.Migrated,
		"skipped", result.Skipped,
		"errors", result.Errors,
	)
	return nil
}
