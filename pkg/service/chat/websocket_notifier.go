package chat

import (
	"context"
	"encoding/json"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/controller/websocket"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	websocket_model "github.com/secmon-lab/warren/pkg/domain/model/websocket"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// WebSocketNotifier sends notifications via WebSocket connections
type WebSocketNotifier struct {
	hub *websocket.Hub
}

// NewWebSocketNotifier creates a new WebSocketNotifier
func NewWebSocketNotifier(hub *websocket.Hub) interfaces.ChatNotifier {
	return &WebSocketNotifier{
		hub: hub,
	}
}

// NotifyMessage sends a message via WebSocket to connected clients for the ticket
func (w *WebSocketNotifier) NotifyMessage(ctx context.Context, ticketID types.TicketID, message string) error {
	response := &websocket_model.ChatResponse{
		Type:      "message",
		Content:   message,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(response)
	if err != nil {
		return goerr.Wrap(err, "failed to marshal WebSocket message")
	}

	w.hub.BroadcastToTicket(ticketID, data)
	return nil
}

// NotifyTrace sends a trace message via WebSocket to connected clients for the ticket
func (w *WebSocketNotifier) NotifyTrace(ctx context.Context, ticketID types.TicketID, message string) error {
	response := &websocket_model.ChatResponse{
		Type:      "trace",
		Content:   message,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(response)
	if err != nil {
		return goerr.Wrap(err, "failed to marshal WebSocket trace message")
	}

	w.hub.BroadcastToTicket(ticketID, data)
	return nil
}
