package chatnotifier

import (
	"context"
	"fmt"
	"io"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

// CLINotifier delivers messages to a writer (typically os.Stdout) and
// persists them as Messages in the current CLI Session.
type CLINotifier struct {
	repo     interfaces.Repository
	session  *session.Session
	turnID   *types.TurnID
	w        io.Writer
	ticketID *types.TicketID
}

// NewCLINotifier constructs a CLINotifier bound to sess + turnID. turnID may
// be nil for messages that are not tied to a Turn, though in CLI mode every
// invocation typically carries a Turn.
func NewCLINotifier(repo interfaces.Repository, sess *session.Session, turnID *types.TurnID, w io.Writer) *CLINotifier {
	return &CLINotifier{
		repo:     repo,
		session:  sess,
		turnID:   turnID,
		w:        w,
		ticketID: sess.TicketIDOrNil(),
	}
}

func (c *CLINotifier) persist(ctx context.Context, msgType session.MessageType, content string, author *session.Author) {
	msg := session.NewMessageV2(ctx, c.session.ID, c.ticketID, c.turnID, msgType, content, author)
	if err := c.repo.PutSessionMessage(ctx, msg); err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to persist CLI session message",
			goerr.V("session_id", c.session.ID),
			goerr.V("type", msgType),
		))
	}
}

func (c *CLINotifier) write(prefix, content string) error {
	if c.w == nil {
		return nil
	}
	_, err := fmt.Fprintf(c.w, "%s%s\n", prefix, content)
	if err != nil {
		return goerr.Wrap(err, "failed to write CLI notification")
	}
	return nil
}

func (c *CLINotifier) Notify(ctx context.Context, content string) error {
	c.persist(ctx, session.MessageTypeResponse, content, nil)
	return c.write("", content)
}

func (c *CLINotifier) Trace(ctx context.Context, content string) error {
	c.persist(ctx, session.MessageTypeTrace, content, nil)
	return c.write("  · ", content)
}

func (c *CLINotifier) Warn(ctx context.Context, content string) error {
	c.persist(ctx, session.MessageTypeWarning, content, nil)
	return c.write("⚠️  ", content)
}

func (c *CLINotifier) Plan(ctx context.Context, content string) error {
	c.persist(ctx, session.MessageTypePlan, content, nil)
	return c.write("📋 ", content)
}

func (c *CLINotifier) NotifyUser(ctx context.Context, content string, author *session.Author) error {
	if author == nil {
		return goerr.New("NotifyUser requires author")
	}
	c.persist(ctx, session.MessageTypeUser, content, author)
	// User input is already visible in the terminal; no echo back.
	return nil
}
