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
	"github.com/secmon-lab/warren/pkg/repository"
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

func runSessionSourceBackfill(ctx context.Context, projectID, databaseID string, dryRun bool) error {
	logger := logging.From(ctx)

	repo, cleanup, err := openFirestoreRepository(ctx, projectID, databaseID)
	if err != nil {
		return err
	}
	defer cleanup()

	listAll := func(ctx context.Context) ([]*sessModel.Session, error) {
		return listAllSessions(ctx, projectID, databaseID)
	}

	job := migration.NewSessionSourceBackfillJob(repo, listAll)
	result, err := job.Run(ctx, migration.Options{DryRun: dryRun})
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

func runCommentToMessage(ctx context.Context, projectID, databaseID string, dryRun bool) error {
	logger := logging.From(ctx)
	repo, cleanup, err := openFirestoreRepository(ctx, projectID, databaseID)
	if err != nil {
		return err
	}
	defer cleanup()

	job := migration.NewCommentToMessageJob(repo, nil, nil, nil)
	result, err := job.Run(ctx, migration.Options{DryRun: dryRun})
	if err != nil {
		// The current implementation is intentionally a stub; surface
		// the "not yet implemented" error so operators know to use a
		// different path rather than assume success.
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

func runTurnSynthesis(ctx context.Context, projectID, databaseID string, dryRun bool) error {
	repo, cleanup, err := openFirestoreRepository(ctx, projectID, databaseID)
	if err != nil {
		return err
	}
	defer cleanup()

	job := migration.NewTurnSynthesisJob(repo)
	_, err = job.Run(ctx, migration.Options{DryRun: dryRun})
	return err
}

func runHistoryScope(ctx context.Context, projectID, databaseID string, dryRun bool) error {
	repo, cleanup, err := openFirestoreRepository(ctx, projectID, databaseID)
	if err != nil {
		return err
	}
	defer cleanup()

	// The history-scope job also needs a storage service; that wiring
	// is deferred until the job body is implemented (it will take the
	// service as a constructor argument).
	job := migration.NewHistoryScopeJob(repo, nil)
	_, err = job.Run(ctx, migration.Options{DryRun: dryRun})
	return err
}

func runCleanupLegacy(ctx context.Context, projectID, databaseID string, dryRun bool) error {
	repo, cleanup, err := openFirestoreRepository(ctx, projectID, databaseID)
	if err != nil {
		return err
	}
	defer cleanup()

	job := migration.NewCleanupLegacyJob(repo)
	_, err = job.Run(ctx, migration.Options{DryRun: dryRun})
	return err
}
