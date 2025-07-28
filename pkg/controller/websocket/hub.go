package websocket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/m-mizutani/goerr/v2"
	websocket_model "github.com/secmon-lab/warren/pkg/domain/model/websocket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients grouped by ticket ID
	tickets map[types.TicketID]map[*Client]bool

	// Registered clients indexed by client ID for direct access
	clients map[string]*Client

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast message to clients of a specific ticket
	broadcast chan *BroadcastMessage

	// Mutex to protect concurrent access to tickets and clients maps
	mu sync.RWMutex

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// Client is a middleman between the websocket connection and the hub
type Client struct {
	hub *Hub

	// The websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// Ticket ID this client is connected to
	ticketID types.TicketID

	// User ID of the connected user
	userID string

	// Tab ID to distinguish multiple connections from same user
	tabID string

	// Unique client ID for this connection
	clientID string

	// Context for this client
	ctx    context.Context
	cancel context.CancelFunc

	// Mutex to protect send channel
	mu sync.Mutex
}

// BroadcastMessage represents a message to be broadcast to all clients of a ticket
type BroadcastMessage struct {
	TicketID types.TicketID
	Message  []byte
}

const (
	// Maximum message size allowed from peer (64KB)
	maxMessageSize = 64 * 1024

	// Maximum number of clients per ticket
	maxClientsPerTicket = 10

	// Buffer size for client send channel
	clientSendBufferSize = 256
)

// NewHub creates a new Hub
func NewHub(ctx context.Context) *Hub {
	ctx, cancel := context.WithCancel(ctx)
	return &Hub{
		tickets:    make(map[types.TicketID]map[*Client]bool),
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	logger := logging.From(h.ctx)
	logger.Info("WebSocket Hub started")

	defer func() {
		logger.Info("WebSocket Hub stopped")
		h.cancel()
	}()

	for {
		select {
		case <-h.ctx.Done():
			logger.Info("Hub context cancelled, shutting down")
			return

		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastToTicket(message)
		}
	}
}

// registerClient registers a new client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	logger := logging.From(h.ctx)

	// Check if ticket already has too many clients
	if clients, exists := h.tickets[client.ticketID]; exists {
		// Remove existing connections from the same user AND tab
		// This allows multiple tabs from the same user while preventing duplicate connections
		// from the same tab (e.g., during reconnection)
		for existingClient := range clients {
			if existingClient.userID == client.userID && existingClient.tabID == client.tabID {
				logger.Info("Replacing existing connection for user/tab",
					"ticket_id", client.ticketID,
					"user_id", client.userID,
					"tab_id", client.tabID,
					"old_client_id", existingClient.clientID,
					"new_client_id", client.clientID)
				delete(clients, existingClient)
				existingClient.mu.Lock()
				if existingClient.send != nil {
					close(existingClient.send)
					existingClient.send = nil
				}
				existingClient.mu.Unlock()
			}
		}

		if len(clients) >= maxClientsPerTicket {
			logger.Warn("Maximum clients reached for ticket",
				"ticket_id", client.ticketID,
				"max_clients", maxClientsPerTicket)
			client.mu.Lock()
			if client.send != nil {
				close(client.send)
				client.send = nil
			}
			client.mu.Unlock()
			return
		}
	} else {
		h.tickets[client.ticketID] = make(map[*Client]bool)
	}

	h.tickets[client.ticketID][client] = true
	h.clients[client.clientID] = client

	logger.Info("Client registered",
		"ticket_id", client.ticketID,
		"user_id", client.userID,
		"tab_id", client.tabID,
		"client_id", client.clientID,
		"total_clients", len(h.tickets[client.ticketID]))

	// Send welcome message
	welcomeResp := websocket_model.NewStatusResponse("Connected to chat")
	if data, err := welcomeResp.ToBytes(); err == nil {
		select {
		case client.send <- data:
		default:
			client.mu.Lock()
			if client.send != nil {
				close(client.send)
				client.send = nil
			}
			client.mu.Unlock()
			delete(h.tickets[client.ticketID], client)
		}
	}
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	logger := logging.From(h.ctx)

	if clients, exists := h.tickets[client.ticketID]; exists {
		if _, exists := clients[client]; exists {
			delete(clients, client)

			// Only close if channel is not nil (prevents double close)
			client.mu.Lock()
			if client.send != nil {
				close(client.send)
				client.send = nil
			}
			client.mu.Unlock()

			logger.Info("Client unregistered",
				"ticket_id", client.ticketID,
				"user_id", client.userID,
				"tab_id", client.tabID,
				"client_id", client.clientID,
				"remaining_clients", len(clients))

			// Remove ticket entry if no clients remain
			if len(clients) == 0 {
				delete(h.tickets, client.ticketID)
				logger.Info("Removed empty ticket", "ticket_id", client.ticketID)
			}
		}
	}

	// Remove from clients map
	delete(h.clients, client.clientID)

	// Cancel client context
	client.cancel()
}

// broadcastToTicket sends a message to all clients connected to a specific ticket
func (h *Hub) broadcastToTicket(message *BroadcastMessage) {
	h.mu.RLock()
	clients, exists := h.tickets[message.TicketID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	logger := logging.From(h.ctx)
	logger.Debug("Broadcasting message to ticket",
		"ticket_id", message.TicketID,
		"client_count", len(clients))

	// Send message to all clients of the ticket
	for client := range clients {
		select {
		case client.send <- message.Message:
		default:
			// Client's send channel is full, remove the client
			h.unregister <- client
		}
	}
}

// BroadcastToTicket sends a message to all clients of a specific ticket
func (h *Hub) BroadcastToTicket(ticketID types.TicketID, message []byte) {
	select {
	case h.broadcast <- &BroadcastMessage{
		TicketID: ticketID,
		Message:  message,
	}:
	case <-h.ctx.Done():
		// Hub is shutting down
	}
}

// GetClientCount returns the number of clients connected to a specific ticket
func (h *Hub) GetClientCount(ticketID types.TicketID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, exists := h.tickets[ticketID]; exists {
		return len(clients)
	}
	return 0
}

// GetTotalClientCount returns the total number of connected clients across all tickets
func (h *Hub) GetTotalClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, clients := range h.tickets {
		total += len(clients)
	}
	return total
}

// GetActiveTickets returns a list of ticket IDs that have active clients
func (h *Hub) GetActiveTickets() []types.TicketID {
	h.mu.RLock()
	defer h.mu.RUnlock()

	tickets := make([]types.TicketID, 0, len(h.tickets))
	for ticketID := range h.tickets {
		tickets = append(tickets, ticketID)
	}
	return tickets
}

// NewClient creates a new client (for backward compatibility)
func (h *Hub) NewClient(conn *websocket.Conn, ticketID types.TicketID, userID string) *Client {
	return h.NewClientWithTabID(conn, ticketID, userID, "default")
}

// NewClientWithTabID creates a new client with a specific tab ID
func (h *Hub) NewClientWithTabID(conn *websocket.Conn, ticketID types.TicketID, userID string, tabID string) *Client {
	ctx, cancel := context.WithCancel(h.ctx)

	// Generate unique client ID including tab ID
	clientID := generateClientIDWithTab(ticketID, userID, tabID)

	return &Client{
		hub:      h,
		conn:     conn,
		send:     make(chan []byte, clientSendBufferSize),
		ticketID: ticketID,
		userID:   userID,
		tabID:    tabID,
		clientID: clientID,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Register registers a client with the hub
func (h *Hub) Register(client *Client) {
	select {
	case h.register <- client:
	case <-h.ctx.Done():
		// Hub is shutting down
		close(client.send)
	}
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	select {
	case h.unregister <- client:
	case <-h.ctx.Done():
		// Hub is already shutting down
	}
}

// Close gracefully shuts down the hub
func (h *Hub) Close() error {
	h.cancel()

	// Close all client connections
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, clients := range h.tickets {
		for client := range clients {
			client.cancel()
			client.mu.Lock()
			if client.send != nil {
				close(client.send)
				client.send = nil
			}
			client.mu.Unlock()
		}
	}

	return nil
}

// SendToTicket is a convenience method to send a WebSocket message to a ticket
func (h *Hub) SendToTicket(ticketID types.TicketID, response *websocket_model.ChatResponse) error {
	data, err := response.ToBytes()
	if err != nil {
		return goerr.Wrap(err, "failed to marshal response")
	}

	h.BroadcastToTicket(ticketID, data)
	return nil
}

// SendMessageToTicket sends a chat message to all clients of a ticket
func (h *Hub) SendMessageToTicket(ticketID types.TicketID, content string, user *websocket_model.User) error {
	response := websocket_model.NewMessageResponse(content, user)
	return h.SendToTicket(ticketID, response)
}

// SendStatusToTicket sends a status message to all clients of a ticket
func (h *Hub) SendStatusToTicket(ticketID types.TicketID, content string) error {
	response := websocket_model.NewStatusResponse(content)
	return h.SendToTicket(ticketID, response)
}

// SendErrorToTicket sends an error message to all clients of a ticket
func (h *Hub) SendErrorToTicket(ticketID types.TicketID, content string) error {
	response := websocket_model.NewErrorResponse(content)
	return h.SendToTicket(ticketID, response)
}


// generateClientIDWithTab generates a unique client ID including tab ID
func generateClientIDWithTab(ticketID types.TicketID, userID string, tabID string) string {
	// Create a unique ID using timestamp and random bytes
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("client_%s_%s_%s_%d", ticketID, userID, tabID, timestamp)
	}
	randomHex := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("client_%s_%s_%s_%d_%s", ticketID, userID, tabID, timestamp, randomHex)
}

// SendToClient sends a message to a specific client by client ID
func (h *Hub) SendToClient(clientID string, response *websocket_model.ChatResponse) error {
	h.mu.RLock()
	client, exists := h.clients[clientID]
	h.mu.RUnlock()

	if !exists {
		return goerr.New("client not found", goerr.V("client_id", clientID))
	}

	data, err := response.ToBytes()
	if err != nil {
		return goerr.Wrap(err, "failed to marshal response")
	}

	select {
	case client.send <- data:
		return nil
	default:
		// Client's send channel is full, unregister the client
		h.Unregister(client)
		return goerr.New("client send channel full, client unregistered", goerr.V("client_id", clientID))
	}
}

// SendMessageToClient sends a chat message to a specific client
func (h *Hub) SendMessageToClient(clientID string, content string, user *websocket_model.User) error {
	response := websocket_model.NewMessageResponse(content, user)
	return h.SendToClient(clientID, response)
}

// SendTraceToClient sends a trace message to a specific client
func (h *Hub) SendTraceToClient(clientID string, content string, user *websocket_model.User) error {
	response := websocket_model.NewTraceResponse(content, user)
	return h.SendToClient(clientID, response)
}

// GetClientID returns the client ID for a client
func (c *Client) GetClientID() string {
	return c.clientID
}
