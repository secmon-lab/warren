package interfaces

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/types"
)

// WebEventPublisher is the minimal contract the WebNotifier (service layer)
// uses to reach the WebSocket Hub (controller layer) without importing the
// controller package directly. The concrete implementation lives in
// pkg/controller/websocket/hub_publisher.go and delegates to Hub's
// per-ticket broadcast.
//
// Only a single method is exposed so that the WebNotifier stays trivially
// mockable and independent of WebSocket framing details.
type WebEventPublisher interface {
	// PublishToTicket sends a pre-serialized JSON payload to every
	// WebSocket client currently subscribed to ticketID on this instance.
	// Cross-instance delivery is out of scope (chat-session-redesign §G:
	// a WebSocket connection's input and output are always on the same
	// instance, mirroring Slack webhook handling).
	PublishToTicket(ctx context.Context, ticketID types.TicketID, payload []byte) error
}
