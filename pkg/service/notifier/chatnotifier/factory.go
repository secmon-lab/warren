package chatnotifier

import (
	"context"
	"io"
	"os"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	slackModel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// SlackThreadFactory is the minimal surface the chatnotifier Factory needs
// from the Slack service: resolve a Thread reference into a
// SlackThreadService. The concrete implementation lives in
// pkg/service/slack.
type SlackThreadFactory interface {
	NewThread(thread slackModel.Thread) interfaces.SlackThreadService
}

// noopPublisher implements interfaces.WebEventPublisher as a silent sink.
// Used when the Factory is constructed without a real publisher so that
// WebNotifier can still persist Messages.
type noopPublisher struct{}

func (noopPublisher) PublishToTicket(context.Context, types.TicketID, []byte) error { return nil }

// Factory constructs SessionNotifier implementations per-request. It carries
// only shared service dependencies (repository, Slack service, web event
// publisher, CLI writer) and holds no per-Session state. A Factory is
// therefore safe to share across goroutines and across Sessions.
//
// Factories live for the lifetime of the process; the Session-specific
// binding happens inside FromSession which returns a fresh notifier each
// call.
type Factory struct {
	repo         interfaces.Repository
	slackService SlackThreadFactory
	webPublisher interfaces.WebEventPublisher
	cliWriter    io.Writer
}

// FactoryOption configures optional dependencies on a Factory.
type FactoryOption func(*Factory)

// WithSlackService provides the Slack service used to construct
// SlackChatNotifier instances. Without it, Slack Sessions fall back to
// NoopNotifier.
func WithSlackService(s SlackThreadFactory) FactoryOption {
	return func(f *Factory) { f.slackService = s }
}

// WithWebPublisher provides the WebSocket Hub publisher used to construct
// WebNotifier instances. Without it, Web Sessions fall back to
// NoopNotifier (Messages are still persisted via the caller, but no
// realtime push happens).
func WithWebPublisher(p interfaces.WebEventPublisher) FactoryOption {
	return func(f *Factory) { f.webPublisher = p }
}

// WithCLIWriter provides the writer used by CLINotifier (defaults to
// os.Stdout).
func WithCLIWriter(w io.Writer) FactoryOption {
	return func(f *Factory) { f.cliWriter = w }
}

// NewFactory constructs a Factory. repo is required.
func NewFactory(repo interfaces.Repository, opts ...FactoryOption) *Factory {
	f := &Factory{
		repo:      repo,
		cliWriter: os.Stdout,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// FromSession returns a SessionNotifier appropriate for sess.Source. turnID
// is the Turn that new Messages should be attributed to; it may be nil for
// messages that do not belong to a Turn (e.g. non-mention Slack thread
// messages).
//
// The returned notifier is owned by the caller and safe to pass through
// ctx via WithNotifier. It should not be reused across different Sessions.
func (f *Factory) FromSession(sess *session.Session, turnID *types.TurnID) interfaces.SessionNotifier {
	if sess == nil {
		return NoopNotifier{}
	}
	switch sess.Source {
	case session.SessionSourceSlack:
		if f.slackService == nil || sess.ChannelRef == nil || sess.ChannelRef.SlackThread == nil {
			return NoopNotifier{}
		}
		thread := f.slackService.NewThread(*sess.ChannelRef.SlackThread)
		return NewSlackChatNotifier(f.repo, thread, sess, turnID)

	case session.SessionSourceWeb:
		if f.webPublisher == nil {
			// Web publisher is not wired; fall back to persist-only via
			// WebNotifier with a noop publisher so Messages still land
			// in Firestore.
			return NewWebNotifier(f.repo, noopPublisher{}, sess, turnID)
		}
		return NewWebNotifier(f.repo, f.webPublisher, sess, turnID)

	case session.SessionSourceCLI:
		return NewCLINotifier(f.repo, sess, turnID, f.cliWriter)

	default:
		return NoopNotifier{}
	}
}
