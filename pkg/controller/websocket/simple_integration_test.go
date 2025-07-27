package websocket_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/m-mizutani/gt"
	websocket_ctrl "github.com/secmon-lab/warren/pkg/controller/websocket"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	websocket_model "github.com/secmon-lab/warren/pkg/domain/model/websocket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/chat"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

func TestSimpleWebSocketIntegration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup
	repo := repository.NewMemory()
	hub := websocket_ctrl.NewHub(ctx)
	go hub.Run()
	defer hub.Close()

	// Create test ticket
	testTicket := ticket.New(ctx, []types.AlertID{}, nil)
	testTicket.ID = types.TicketID("test-ticket")
	testTicket.Metadata.Title = "Test Ticket"
	err := repo.PutTicket(ctx, testTicket)
	gt.NoError(t, err)

	// Create handler
	handler := websocket_ctrl.NewHandler(hub, repo, nil)

	// Setup router with auth
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := user.WithUserID(r.Context(), "test-user")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Get("/ws/chat/ticket/{ticketID}", handler.HandleTicketChat)

	server := httptest.NewServer(r)
	defer server.Close()

	// Test 1: Basic WebSocket connection
	t.Run("BasicConnection", func(t *testing.T) {
		wsURL := strings.Replace(server.URL, "http", "ws", 1) + "/ws/chat/ticket/test-ticket"
		ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		gt.NoError(t, err)
		gt.Value(t, resp.StatusCode).Equal(http.StatusSwitchingProtocols)
		defer ws.Close()

		// Should receive initial status message
		var status websocket_model.ChatResponse
		err = ws.ReadJSON(&status)
		gt.NoError(t, err)
		gt.Value(t, status.Type).Equal("status")
		gt.Value(t, status.Content).Equal("Connected to chat")
	})

	// Test 2: Broadcast via ChatNotifier
	t.Run("BroadcastMessage", func(t *testing.T) {
		wsURL := strings.Replace(server.URL, "http", "ws", 1) + "/ws/chat/ticket/test-ticket"
		ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		gt.NoError(t, err)
		gt.Value(t, resp.StatusCode).Equal(http.StatusSwitchingProtocols)
		defer ws.Close()

		// Skip status message
		var status websocket_model.ChatResponse
		if err := ws.ReadJSON(&status); err != nil {
			t.Logf("Warning: Failed to read initial status message: %v", err)
		}

		// Send broadcast
		wsNotifier := chat.NewWebSocketNotifier(hub)
		err = wsNotifier.NotifyMessage(ctx, testTicket.ID, "Broadcast test")
		gt.NoError(t, err)

		// Receive broadcast
		var msg websocket_model.ChatResponse
		if err := ws.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Logf("Warning: Failed to set read deadline: %v", err)
		}
		err = ws.ReadJSON(&msg)
		gt.NoError(t, err)
		gt.Value(t, msg.Type).Equal("message")
		gt.Value(t, msg.Content).Equal("Broadcast test")
	})

	// Test 3: Ping/Pong
	t.Run("PingPong", func(t *testing.T) {
		wsURL := strings.Replace(server.URL, "http", "ws", 1) + "/ws/chat/ticket/test-ticket"
		ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		gt.NoError(t, err)
		gt.Value(t, resp.StatusCode).Equal(http.StatusSwitchingProtocols)
		defer ws.Close()

		// Skip status message
		var status websocket_model.ChatResponse
		if err := ws.ReadJSON(&status); err != nil {
			t.Logf("Warning: Failed to read initial status message: %v", err)
		}

		// Send ping
		ping := websocket_model.ChatMessage{
			Type:      "ping",
			Timestamp: time.Now().Unix(),
		}
		err = ws.WriteJSON(ping)
		gt.NoError(t, err)

		// Receive pong
		var pong websocket_model.ChatResponse
		if err := ws.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Logf("Warning: Failed to set read deadline: %v", err)
		}
		err = ws.ReadJSON(&pong)
		gt.NoError(t, err)
		gt.Value(t, pong.Type).Equal("pong")
	})
}

func TestWebSocketSecurity(t *testing.T) {
	// Test: No authentication
	t.Run("NoAuth", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := repository.NewMemory()
		hub := websocket_ctrl.NewHub(ctx)
		go hub.Run()
		defer hub.Close()

		handler := websocket_ctrl.NewHandler(hub, repo, nil)

		r := chi.NewRouter()
		r.Get("/ws/chat/ticket/{ticketID}", handler.HandleTicketChat)
		server := httptest.NewServer(r)
		defer server.Close()

		wsURL := strings.Replace(server.URL, "http", "ws", 1) + "/ws/chat/ticket/test"
		_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		gt.Error(t, err)
		gt.Value(t, resp.StatusCode).Equal(http.StatusUnauthorized)
	})

	// Test: Invalid ticket
	t.Run("InvalidTicket", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := repository.NewMemory()
		hub := websocket_ctrl.NewHub(ctx)
		go hub.Run()
		defer hub.Close()

		handler := websocket_ctrl.NewHandler(hub, repo, nil)

		r := chi.NewRouter()
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := user.WithUserID(r.Context(), "test-user")
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})
		r.Get("/ws/chat/ticket/{ticketID}", handler.HandleTicketChat)
		server := httptest.NewServer(r)
		defer server.Close()

		wsURL := strings.Replace(server.URL, "http", "ws", 1) + "/ws/chat/ticket/non-existent"
		_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		gt.Error(t, err)
		gt.Value(t, resp.StatusCode).Equal(http.StatusInternalServerError)
	})
}
