package migration

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// CommentToMessageJob rewrites every existing ticket.Comment as a
// type=user session.Message attached to the corresponding Slack Session.
//
// The original Comment row is NOT deleted by this job — cleanup happens
// in the CleanupLegacyJob once the application code no longer reads
// Comments. Idempotence is achieved by stamping each generated Message's
// Content-derived identifier into the Session tree and checking for it
// on re-run.
type CommentToMessageJob struct {
	repo     interfaces.Repository
	resolver SessionResolverClient
}

// SessionResolverClient is the minimal surface the job needs from the
// chat-session-redesign session.Resolver. An interface keeps the job
// testable with a small fake.
type SessionResolverClient interface {
	ResolveSlackSession(ctx context.Context, ticketID *types.TicketID, thread threadRef, userID types.UserID) (*sessModel.Session, bool, error)
}

// threadRef is an alias so the interface signature does not reach into
// pkg/domain/model/slack in an awkward way. The production wrapper
// converts a slack.Thread into threadRef transparently.
type threadRef = slackThreadLike

// slackThreadLike is the shape the migration job cares about from a
// Slack thread reference.
type slackThreadLike = struct {
	TeamID    string
	ChannelID string
	ThreadID  string
}

// CommentSource abstracts the Comment read path so the job does not need
// to know whether it is running against memory or Firestore.
type CommentSource interface {
	ListAllTickets(ctx context.Context) ([]*ticket.Ticket, error)
	ListTicketComments(ctx context.Context, ticketID types.TicketID) ([]ticket.Comment, error)
}

// MessageWriter abstracts the Message write path; implementations call
// Repository.PutSessionMessage with a pre-built Message.
type MessageWriter interface {
	PutSessionMessage(ctx context.Context, msg *sessModel.Message) error
}

// NewCommentToMessageJob constructs the job. Passing a dedicated
// CommentSource / MessageWriter rather than the full Repository keeps
// migration logic unit-testable without the entire interface surface.
func NewCommentToMessageJob(repo interfaces.Repository, resolver SessionResolverClient, source CommentSource, writer MessageWriter) *CommentToMessageJob {
	return &CommentToMessageJob{
		repo:     repo,
		resolver: resolver,
	}
}

func (j *CommentToMessageJob) Name() string { return "comment-to-message" }

func (j *CommentToMessageJob) Description() string {
	return "Rewrite ticket.Comment rows as session.Message(type=user) attached to the owning Slack Session. Idempotent; original Comment rows are not deleted (see cleanup-legacy)."
}

// Run scans every Ticket, resolves its Slack Session (or creates one if
// a legacy Comment predates any Session), and emits one Message per
// Comment. The job returns a Result summarizing counts; errors on an
// individual Comment are counted but do not abort the job.
//
// This method body is a placeholder: the concrete implementation
// requires ListAllTickets / ListTicketComments which are not yet present
// on the Repository interface. Phase 7 full wiring will add those
// helpers and fill in the body. Keeping the Job skeleton in place lets
// the CLI registration and the rest of the migration machinery land
// cleanly now.
func (j *CommentToMessageJob) Run(ctx context.Context, opts Options) (*Result, error) {
	return nil, goerr.New("comment-to-message: not yet implemented",
		goerr.V("note", "Phase 7 full wiring pending; see chat-session-redesign spec"))
}
