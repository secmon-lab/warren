package websocket

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	websocket_model "github.com/secmon-lab/warren/pkg/domain/model/websocket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

// Note: webSocketResponder has been removed in favor of using msg.With context-based approach

// Handler handles WebSocket connections for chat functionality
type Handler struct {
	hub            *Hub
	repository     interfaces.Repository
	useCases       *usecase.UseCases
	upgrader       websocket.Upgrader
	frontendURL    string
	allowedOrigins []string // Additional allowed origins for development
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, repository interfaces.Repository, useCases *usecase.UseCases) *Handler {
	h := &Handler{
		hub:        hub,
		repository: repository,
		useCases:   useCases,
	}
	h.setupUpgrader()
	return h
}

// WithFrontendURL sets the frontend URL for origin checking
func (h *Handler) WithFrontendURL(url string) *Handler {
	h.frontendURL = url
	h.setupUpgrader()
	return h
}

// WithAllowedOrigins sets additional allowed origins (useful for development)
func (h *Handler) WithAllowedOrigins(origins []string) *Handler {
	h.allowedOrigins = origins
	h.setupUpgrader()
	return h
}

// setupUpgrader configures the WebSocket upgrader with appropriate origin checking
func (h *Handler) setupUpgrader() {
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}
}

// checkOrigin validates the request origin for CSWH protection
func (h *Handler) checkOrigin(r *http.Request) bool {
	logger := logging.From(r.Context())

	// If no frontend URL is configured, deny all connections for security
	if h.frontendURL == "" && len(h.allowedOrigins) == 0 {
		logger.Warn("Origin check failed: no frontend URL or allowed origins configured")
		return false
	}

	// Get the request origin
	origin := r.Header.Get("Origin")

	if origin == "" {
		// No origin header means same-origin request, which should be allowed
		logger.Debug("Origin check passed: no origin header (same-origin request)")
		return true
	}

	// Build list of allowed origins
	allowedOrigins := []string{}

	// Add frontend URL as allowed origin
	if h.frontendURL != "" {
		frontendOrigin := strings.TrimSuffix(h.frontendURL, "/")
		allowedOrigins = append(allowedOrigins, frontendOrigin)
	}

	// Add additional allowed origins
	for _, allowed := range h.allowedOrigins {
		allowedOrigins = append(allowedOrigins, strings.TrimSuffix(allowed, "/"))
	}

	logger.Debug("WebSocket origin check",
		"origin", origin,
		"allowed_origins", allowedOrigins,
		"frontend_url", h.frontendURL)

	// Check if the origin matches any allowed origin
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			logger.Debug("Origin check passed: origin matches allowed origin",
				"origin", origin,
				"matched", allowed)
			return true
		}
	}

	logger.Warn("Origin check failed: origin not in allowed list",
		"origin", origin,
		"allowed_origins", allowedOrigins)
	return false
}

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10
)

// HandleTicketChat handles WebSocket connections for ticket chat
func (h *Handler) HandleTicketChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.From(ctx)

	// Extract ticket ID from URL
	ticketIDStr := chi.URLParam(r, "ticketID")
	if ticketIDStr == "" {
		logger.Warn("missing ticket ID in WebSocket request")
		http.Error(w, "Missing ticket ID", http.StatusBadRequest)
		return
	}

	ticketID := types.TicketID(ticketIDStr)

	// Extract tab ID from query parameter (for distinguishing multiple connections from same user)
	tabID := r.URL.Query().Get("tab")
	if tabID == "" {
		// Generate a fallback tab ID if not provided (for backward compatibility)
		tabID = "default"
	}

	// Get user ID from context (set by authentication middleware)
	userID := user.FromContext(ctx)
	if userID == "" {
		logger.Warn("missing user ID in WebSocket request",
			"headers", r.Header)
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	logger.Debug("WebSocket authentication successful",
		"user_id", userID,
		"ticket_id", ticketID,
		"tab_id", tabID)

	// Verify ticket exists and user has access
	ticket, err := h.repository.GetTicket(ctx, ticketID)
	if err != nil {
		logger.Error("failed to get ticket", "error", err, "ticket_id", ticketID)
		http.Error(w, "Failed to verify ticket", http.StatusInternalServerError)
		return
	}
	if ticket == nil {
		logger.Warn("ticket not found", "ticket_id", ticketID)
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	logger.Debug("Ticket validation successful, attempting WebSocket upgrade",
		"ticket_id", ticketID,
		"user_id", userID,
		"tab_id", tabID)

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("failed to upgrade connection",
			"error", err,
			"error_type", fmt.Sprintf("%T", err),
			"error_string", err.Error(),
			"ticket_id", ticketID,
			"user_id", userID,
			"headers", r.Header,
			"response_written", w.Header().Get("Content-Type") != "")
		// Don't call http.Error here as upgrader may have already written headers
		return
	}

	logger.Info("WebSocket connection established",
		"ticket_id", ticketID,
		"user_id", userID,
		"tab_id", tabID)

	// Create client and register with hub (with tab ID for unique identification)
	client := h.hub.NewClientWithTabID(conn, ticketID, userID, tabID)
	h.hub.Register(client)

	logger.Debug("Starting WebSocket client goroutines",
		"ticket_id", ticketID,
		"user_id", userID,
		"tab_id", tabID,
		"client_id", client.clientID)

	// Start client goroutines
	go h.writePump(client)
	go h.readPump(client)

	logger.Info("WebSocket client setup completed",
		"ticket_id", ticketID,
		"user_id", userID)
}

// readPump pumps messages from the websocket connection to the hub
func (h *Handler) readPump(client *Client) {
	logger := logging.From(client.ctx)

	logger.Debug("ReadPump started",
		"ticket_id", client.ticketID,
		"user_id", client.userID)

	defer func() {
		logger.Debug("ReadPump ending, unregistering client",
			"ticket_id", client.ticketID,
			"user_id", client.userID,
			"client_id", client.clientID)
		h.hub.Unregister(client)
		if err := client.conn.Close(); err != nil {
			logger.Debug("failed to close connection in readPump", "error", err)
		}
	}()

	client.conn.SetReadLimit(maxMessageSize)
	if err := client.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		logger.Error("failed to set read deadline", "error", err)
		return
	}
	client.conn.SetPongHandler(func(string) error {
		if err := client.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			logger.Error("failed to set read deadline in pong handler", "error", err)
		}
		return nil
	})

	for {
		select {
		case <-client.ctx.Done():
			logger.Info("Client context cancelled, stopping read pump",
				"ticket_id", client.ticketID,
				"user_id", client.userID)
			return
		default:
		}

		// Read message from WebSocket
		_, messageBytes, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("unexpected WebSocket close",
					"error", err,
					"ticket_id", client.ticketID,
					"user_id", client.userID,
					"client_id", client.clientID)
			} else {
				logger.Debug("WebSocket read error",
					"error", err,
					"ticket_id", client.ticketID,
					"user_id", client.userID,
					"client_id", client.clientID)
			}
			break
		}

		// Parse message
		var chatMessage websocket_model.ChatMessage
		if err := chatMessage.FromBytes(messageBytes); err != nil {
			logger.Warn("invalid message format", "error", err)
			h.sendErrorToClient(client, "Invalid message format")
			continue
		}

		// Validate message type
		if !chatMessage.IsValidMessageType() {
			logger.Warn("invalid message type", "type", chatMessage.Type)
			h.sendErrorToClient(client, "Invalid message type")
			continue
		}

		// Handle different message types
		switch chatMessage.Type {
		case "ping":
			// Respond with pong (connection health check)
			h.sendPongToClient(client)

		case "message":
			// Process chat message
			if err := h.handleChatMessage(client, &chatMessage); err != nil {
				logger.Error("failed to handle chat message", "error", err)
				h.sendErrorToClient(client, "Failed to process message")
			}

		default:
			logger.Warn("unhandled message type", "type", chatMessage.Type)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection
func (h *Handler) writePump(client *Client) {
	logger := logging.From(client.ctx)

	logger.Debug("WritePump started",
		"ticket_id", client.ticketID,
		"user_id", client.userID)

	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		if err := client.conn.Close(); err != nil {
			logger.Debug("failed to close connection in writePump", "error", err)
		}
	}()

	for {
		select {
		case <-client.ctx.Done():
			logger.Info("Client context cancelled, stopping write pump",
				"ticket_id", client.ticketID,
				"user_id", client.userID)
			return

		case message, ok := <-client.send:
			if err := client.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logger.Error("failed to set write deadline", "error", err)
				return
			}
			if !ok {
				// The hub closed the channel
				if err := client.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					logger.Debug("failed to write close message", "error", err)
				}
				return
			}

			// Send the main message
			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Error("failed to write message", "error", err)
				return
			}

			// Send queued messages as separate WebSocket messages
			n := len(client.send)
			for i := 0; i < n; i++ {
				queuedMessage := <-client.send
				if err := client.conn.WriteMessage(websocket.TextMessage, queuedMessage); err != nil {
					logger.Error("failed to write queued message", "error", err)
					return
				}
			}

		case <-ticker.C:
			if err := client.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logger.Error("failed to set write deadline for ping", "error", err)
				return
			}
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleChatMessage processes a chat message from a client
func (h *Handler) handleChatMessage(client *Client, message *websocket_model.ChatMessage) error {
	ctx := client.ctx
	logger := logging.From(ctx)

	logger.Info("Processing chat message",
		"ticket_id", client.ticketID,
		"user_id", client.userID,
		"content", message.Content)

	// Get the ticket to process the chat message
	ticket, err := h.repository.GetTicket(ctx, client.ticketID)
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket")
	}
	if ticket == nil {
		return goerr.New("ticket not found")
	}

	// Create user info for the message
	wsUser := &websocket_model.User{
		ID:   client.userID,
		Name: client.userID, // TODO: Get actual user name from user service
	}

	// Broadcast the user's message to all clients
	if err := h.hub.SendMessageToTicket(client.ticketID, message.Content, wsUser); err != nil {
		return goerr.Wrap(err, "failed to broadcast message")
	}

	// Process the message with Chat UseCase for AI response using msg context
	if h.useCases != nil {
		go func() {
			// Use a new context for the async operation
			asyncCtx := user.WithUserID(context.Background(), client.userID)

			logger.Debug("Starting chat processing",
				"ticket_id", client.ticketID,
				"user_id", client.userID,
				"client_id", client.clientID,
				"message", message.Content)

			// Create Warren user for agent messages
			warrenUser := &websocket_model.User{
				ID:   "warren",
				Name: "Warren",
			}

			// Setup notification functions for WebSocket
			notifyFunc := func(ctx context.Context, message string) {
				if err := h.hub.SendMessageToClient(client.clientID, message, warrenUser); err != nil {
					// Check if the error is due to client disconnection
					if err.Error() == "client not found" {
						logger.Debug("client disconnected, skipping message", "client_id", client.clientID)
					} else {
						logger.Error("failed to send message to client", "error", err, "client_id", client.clientID)
					}
				}
			}

			traceFunc := func(ctx context.Context, message string) {
				if err := h.hub.SendTraceToClient(client.clientID, message, warrenUser); err != nil {
					// Check if the error is due to client disconnection
					if err.Error() == "client not found" {
						logger.Debug("client disconnected, skipping trace", "client_id", client.clientID)
					} else {
						logger.Error("failed to send trace to client", "error", err, "client_id", client.clientID)
					}
				}
			}

			// Setup context with WebSocket-specific message handlers
			asyncCtx = msg.With(asyncCtx, notifyFunc, traceFunc)

			if err := h.useCases.Chat(asyncCtx, ticket, message.Content); err != nil {
				logger.Error("failed to process chat message",
					"error", err,
					"ticket_id", client.ticketID,
					"user_id", client.userID,
					"client_id", client.clientID,
					"message", message.Content)
				h.sendErrorToClient(client, "Failed to process message")
			} else {
				logger.Debug("Chat processing completed successfully",
					"ticket_id", client.ticketID,
					"user_id", client.userID,
					"client_id", client.clientID)
			}
		}()
	} else {
		// If no UseCases configured (e.g., in tests), just log
		logger.Debug("No UseCases configured, skipping AI processing")
	}

	return nil
}

// sendErrorToClient sends an error message to a specific client
func (h *Handler) sendErrorToClient(client *Client, message string) {
	response := websocket_model.NewErrorResponse(message)
	data, err := response.ToBytes()
	if err != nil {
		return
	}

	select {
	case client.send <- data:
	default:
		// Client's send channel is full, ignore
	}
}

// sendPongToClient sends a pong response to a specific client
func (h *Handler) sendPongToClient(client *Client) {
	response := websocket_model.NewPongResponse()
	data, err := response.ToBytes()
	if err != nil {
		return
	}

	select {
	case client.send <- data:
	default:
		// Client's send channel is full, ignore
	}
}
