package chatnotifier

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

type ctxKey struct{}

// WithNotifier attaches n to ctx so that downstream code can retrieve it via
// FromContext. The returned context carries a single SessionNotifier
// reference; there is no global registry keyed by Session ID.
//
// This helper is the *only* sanctioned channel for passing a SessionNotifier
// through a request. Per the chat-session-redesign multi-instance rule, we
// never place a package-scope map<SessionID, Notifier>; that would break the
// moment a second instance handles the response.
func WithNotifier(ctx context.Context, n interfaces.SessionNotifier) context.Context {
	if n == nil {
		n = NoopNotifier{}
	}
	return context.WithValue(ctx, ctxKey{}, n)
}

// FromContext retrieves the SessionNotifier stored on ctx by WithNotifier,
// falling back to NoopNotifier when none is set. Callers may safely call
// methods on the result without nil checks.
func FromContext(ctx context.Context) interfaces.SessionNotifier {
	if n, ok := ctx.Value(ctxKey{}).(interfaces.SessionNotifier); ok && n != nil {
		return n
	}
	return NoopNotifier{}
}
