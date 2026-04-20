package migration

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	slackModel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// CommentToMessageJob rewrites every existing ticket.Comment as a
// type=user session.Message attached to the corresponding Slack Session.
// The legacy Comment row is NOT deleted — operators retain the original
// data indefinitely and decide separately when it is safe to discard.
//
// Idempotence: each generated Message carries a SessionID derived
// deterministically from (ticket_id, slack_thread), so re-running the
// job inside the same (session, content, author, created_at) tuple
// yields the same logical Message. To keep the job side-effect-free on
// exact re-runs, the body skips writing a Message when a Message with
// the same content already exists on the target Session under that
// author and timestamp.
type CommentToMessageJob struct {
	source       CommentSource
	resolver     CommentResolverClient
	writer       MessageWriter
	readMessages SessionMessageReader
}

// CommentSource abstracts the pre-redesign Comment subcollection so the
// migration job can operate against either a raw Firestore client
// (production, see pkg/cli/migrate_chat_session.go) or an in-memory
// fake (tests) without pulling Comment CRUD back onto the main
// Repository interface. The interface is read-only on purpose: this PR
// migrates legacy Comment rows into Session Messages but never deletes
// the originals — operators can decide separately when the pre-redesign
// data is safe to discard.
type CommentSource interface {
	ListTicketsWithComments(ctx context.Context) ([]*ticket.Ticket, error)
	GetTicketComments(ctx context.Context, ticketID types.TicketID) ([]ticket.Comment, error)
}

// CommentResolverClient is the minimal surface the job needs from the
// chat-session-redesign session.Resolver.
//
// LookupSlackSession is used in dry-run so the scan does not create
// brand-new Session documents just to produce a preview count — this
// keeps `--dry-run` true to its name.
type CommentResolverClient interface {
	ResolveSlackSession(ctx context.Context, ticketID *types.TicketID, thread slackModel.Thread, userID types.UserID) (*sessModel.Session, bool, error)
	LookupSlackSession(ctx context.Context, ticketID *types.TicketID, thread slackModel.Thread) (*sessModel.Session, bool, error)
}

// MessageWriter abstracts the Message write path.
type MessageWriter interface {
	PutSessionMessage(ctx context.Context, msg *sessModel.Message) error
}

// SessionMessageReader lets the job check for existing Messages to
// enforce idempotence.
type SessionMessageReader interface {
	GetSessionMessages(ctx context.Context, sessionID types.SessionID) ([]*sessModel.Message, error)
}

// NewCommentToMessageJob constructs the job. All dependencies are
// required at the interface level (callers pass the same Repository
// value behind each interface in production).
func NewCommentToMessageJob(source CommentSource, resolver CommentResolverClient, writer MessageWriter, reader SessionMessageReader) *CommentToMessageJob {
	return &CommentToMessageJob{
		source:       source,
		resolver:     resolver,
		writer:       writer,
		readMessages: reader,
	}
}

func (j *CommentToMessageJob) Name() string { return "comment-to-message" }

func (j *CommentToMessageJob) Description() string {
	return "Rewrite ticket.Comment rows as session.Message(type=user) attached to the owning Slack Session. Idempotent and non-destructive: original Comment rows are retained indefinitely."
}

// Run scans every Ticket with Slack comments, resolves its Slack
// Session, and emits one Message per Comment. Errors on an individual
// Comment are counted but do not abort the whole job.
func (j *CommentToMessageJob) Run(ctx context.Context, opts Options) (*Result, error) {
	if j.source == nil || j.resolver == nil || j.writer == nil || j.readMessages == nil {
		return nil, goerr.New("comment-to-message: dependencies not wired")
	}
	tickets, err := j.source.ListTicketsWithComments(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list tickets with comments")
	}

	result := &Result{JobName: j.Name()}

	for _, t := range tickets {
		if t == nil {
			continue
		}
		comments, err := j.source.GetTicketComments(ctx, t.ID)
		if err != nil {
			result.Errors++
			continue
		}
		if len(comments) == 0 {
			continue
		}
		if t.SlackThread == nil {
			// No Slack thread context; the Session cannot be
			// resolved deterministically, so these Comments have
			// to be migrated manually. Count and move on.
			result.Scanned += len(comments)
			result.Skipped += len(comments)
			continue
		}

		tid := t.ID

		// Dry-run: never create a Session document, only look up the
		// existing one (if any) to compute a deterministic preview.
		var sess *sessModel.Session
		var existing []*sessModel.Message
		if opts.DryRun {
			found, ok, err := j.resolver.LookupSlackSession(ctx, &tid, *t.SlackThread)
			if err != nil {
				result.Errors += len(comments)
				continue
			}
			if ok {
				sess = found
				existing, err = j.readMessages.GetSessionMessages(ctx, sess.ID)
				if err != nil {
					result.Errors += len(comments)
					continue
				}
			}
			// When the Session does not yet exist, every comment is a
			// fresh write; `existing` stays nil.
		} else {
			resolved, _, err := j.resolver.ResolveSlackSession(ctx, &tid, *t.SlackThread, "")
			if err != nil {
				result.Errors += len(comments)
				continue
			}
			sess = resolved
			existing, err = j.readMessages.GetSessionMessages(ctx, sess.ID)
			if err != nil {
				result.Errors += len(comments)
				continue
			}
		}

		existingKey := make(map[string]bool, len(existing))
		for _, m := range existing {
			if m.Type != sessModel.MessageTypeUser {
				continue
			}
			authorID := ""
			if m.Author != nil {
				if m.Author.SlackUserID != nil {
					authorID = *m.Author.SlackUserID
				} else {
					authorID = string(m.Author.UserID)
				}
			}
			existingKey[commentKey(authorID, m.Content, m.CreatedAt.Format("2006-01-02T15:04:05Z"))] = true
		}

		for _, c := range comments {
			result.Scanned++
			authorID := ""
			if c.User != nil {
				authorID = c.User.ID
			}
			key := commentKey(authorID, c.Comment, c.CreatedAt.Format("2006-01-02T15:04:05Z"))
			if existingKey[key] {
				result.Skipped++
				continue
			}
			if opts.DryRun {
				result.Migrated++
				continue
			}

			tidCopy := t.ID
			msg := &sessModel.Message{
				ID:        types.NewMessageID(),
				SessionID: sess.ID,
				TicketID:  &tidCopy,
				Type:      sessModel.MessageTypeUser,
				Content:   c.Comment,
				CreatedAt: c.CreatedAt,
				UpdatedAt: c.CreatedAt,
			}
			if c.User != nil {
				slackID := c.User.ID
				display := c.User.Name
				if display == "" {
					display = c.User.ID
				}
				msg.Author = &sessModel.Author{
					UserID:      types.UserID(c.User.ID),
					DisplayName: display,
					SlackUserID: &slackID,
				}
			}
			if err := j.writer.PutSessionMessage(ctx, msg); err != nil {
				result.Errors++
				continue
			}
			result.Migrated++
		}
	}
	return result, nil
}

// commentKey produces a stable identifier for a Comment based on the
// fields that survive the Comment->Message translation. Includes the
// author's Slack user ID so same-second same-content posts from
// different users are preserved as distinct rows.
func commentKey(authorID, content, createdAt string) string {
	return createdAt + "|" + authorID + "|" + content
}
