package interfaces

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/session"
)

// SessionNotifier is the abstraction that chat business logic uses to emit
// Messages inside a Session's currently-running Turn. Each implementation
// (Slack / Web / CLI / Noop) is bound to a specific Session + Turn at
// construction time and is responsible for:
//
//  1. Persisting the Message via the Repository (the single-entry point for
//     session.Message writes).
//  2. Delivering the payload to its channel (Slack API post, WebSocket Hub
//     broadcast, stdout write).
//
// The chat-session-redesign spec requires that strategy code (aster /
// bluebell / chat usecase) route all message writes through SessionNotifier.
// Direct Repository.PutSessionMessage calls from strategy code are
// prohibited; Phase 5 removes them.
//
// This type is separate from the alert pipeline Notifier (same package,
// different concern) and from the deprecated ChatNotifier in chat.go.
type SessionNotifier interface {
	// Notify records a Message with Type=response and delivers it on the
	// owning channel.
	Notify(ctx context.Context, content string) error
	// Trace records a Message with Type=trace.
	Trace(ctx context.Context, content string) error
	// Warn records a Message with Type=warning.
	Warn(ctx context.Context, content string) error
	// Plan records a Message with Type=plan.
	Plan(ctx context.Context, content string) error
	// NotifyUser records a Message with Type=user. author must not be nil.
	// Note that for Slack non-mention messages the TurnID of the created
	// Message is nil; the SessionNotifier implementation decides this from
	// its construction-time binding rather than from an argument.
	NotifyUser(ctx context.Context, content string, author *session.Author) error
}
