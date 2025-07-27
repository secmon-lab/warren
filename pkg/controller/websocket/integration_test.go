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

func TestWebSocketIntegration_CompleteFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup infrastructure
	repo := repository.NewMemory()
	hub := websocket_ctrl.NewHub(ctx)
	go hub.Run()
	defer hub.Close()

	// Create test ticket
	testTicket := ticket.New(ctx, []types.AlertID{}, nil)
	testTicket.ID = types.TicketID("test-ticket")
	testTicket.Metadata.Title = "Test Ticket"
	testTicket.Metadata.Description = "Test Description"
	err := repo.PutTicket(ctx, testTicket)
	gt.NoError(t, err)

	// Create chat notifiers (only WebSocket for this test)
	wsNotifier := chat.NewWebSocketNotifier(hub)

	// Note: Use cases would be used in a real integration test,
	// but for this test we're focusing on WebSocket functionality

	// Create WebSocket handler
	handler := websocket_ctrl.NewHandler(hub, repo, nil)

	// Setup router with auth middleware
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := user.WithUserID(r.Context(), "test-user")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Get("/ws/chat/ticket/{ticketID}", handler.HandleTicketChat)

	// Start test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Connect WebSocket client
	wsURL := strings.Replace(server.URL, "http", "ws", 1) + "/ws/chat/ticket/test-ticket"
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	gt.NoError(t, err)
	gt.Value(t, resp.StatusCode).Equal(http.StatusSwitchingProtocols)
	defer ws.Close()

	// Setup message receiver
	messages := make(chan *websocket_model.ChatResponse, 10)
	go func() {
		for {
			var msg websocket_model.ChatResponse
			err := ws.ReadJSON(&msg)
			if err != nil {
				return
			}
			// Skip initial status message
			if msg.Type != "status" {
				messages <- &msg
			}
		}
	}()

	// Test 1: Send a message via WebSocket
	chatMsg := websocket_model.ChatMessage{
		Type:      "message",
		Content:   "Hello from client",
		Timestamp: time.Now().Unix(),
	}
	err = ws.WriteJSON(chatMsg)
	gt.NoError(t, err)

	// Test 2: Send notification via ChatNotifier
	err = wsNotifier.NotifyMessage(ctx, testTicket.ID, "Hello from server")
	gt.NoError(t, err)

	// Small delay to ensure message is processed
	time.Sleep(50 * time.Millisecond)

	// Test 3: Send trace via ChatNotifier
	err = wsNotifier.NotifyTrace(ctx, testTicket.ID, "Debug trace message")
	gt.NoError(t, err)

	// Verify messages received
	timeout := time.After(2 * time.Second)
	receivedMessages := make(map[string]bool)

	for len(receivedMessages) < 2 {
		select {
		case msg := <-messages:
			if msg != nil {
				t.Logf("Received message: type=%s, content=%s", msg.Type, msg.Content)
				// Mark message as received based on content
				if msg.Type == "message" && msg.Content == "Hello from server" {
					receivedMessages["server_message"] = true
				} else if msg.Type == "trace" && msg.Content == "Debug trace message" {
					receivedMessages["trace_message"] = true
				} else {
					t.Logf("Unmatched message: type=%s, content=%s", msg.Type, msg.Content)
				}
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for messages, received %d/2 (got: %v)", len(receivedMessages), receivedMessages)
		}
	}

	// Verify both messages were received
	gt.Value(t, receivedMessages["server_message"]).Equal(true)
	gt.Value(t, receivedMessages["trace_message"]).Equal(true)
}

func TestWebSocketIntegration_MultipleClients(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup infrastructure
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

	// Setup router
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate different users
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				userID = "test-user"
			}
			ctx := user.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Get("/ws/chat/ticket/{ticketID}", handler.HandleTicketChat)

	server := httptest.NewServer(r)
	defer server.Close()

	// Connect two clients
	wsURL := strings.Replace(server.URL, "http", "ws", 1) + "/ws/chat/ticket/test-ticket"

	// Client 1
	headers1 := http.Header{"X-User-ID": []string{"user1"}}
	ws1, resp, err := websocket.DefaultDialer.Dial(wsURL, headers1)
	gt.NoError(t, err)
	gt.Value(t, resp.StatusCode).Equal(http.StatusSwitchingProtocols)
	defer ws1.Close()

	// Client 2
	headers2 := http.Header{"X-User-ID": []string{"user2"}}
	ws2, resp, err := websocket.DefaultDialer.Dial(wsURL, headers2)
	gt.NoError(t, err)
	gt.Value(t, resp.StatusCode).Equal(http.StatusSwitchingProtocols)
	defer ws2.Close()

	// Setup receivers
	messages1 := make(chan string, 10)
	messages2 := make(chan string, 10)

	go func() {
		for {
			var msg websocket_model.ChatResponse
			err := ws1.ReadJSON(&msg)
			if err != nil {
				return
			}
			// Skip status messages
			if msg.Type != "status" {
				messages1 <- msg.Content
			}
		}
	}()

	go func() {
		for {
			var msg websocket_model.ChatResponse
			err := ws2.ReadJSON(&msg)
			if err != nil {
				return
			}
			// Skip status messages
			if msg.Type != "status" {
				messages2 <- msg.Content
			}
		}
	}()

	// Wait for initial status messages to be processed
	time.Sleep(100 * time.Millisecond)

	// Broadcast message
	wsNotifier := chat.NewWebSocketNotifier(hub)
	err = wsNotifier.NotifyMessage(ctx, testTicket.ID, "Broadcast message")
	gt.NoError(t, err)

	// Both clients should receive the message
	timeout := time.After(2 * time.Second)

	select {
	case msg := <-messages1:
		gt.Value(t, msg).Equal("Broadcast message")
	case <-timeout:
		t.Fatal("Client 1 timeout waiting for broadcast")
	}

	select {
	case msg := <-messages2:
		gt.Value(t, msg).Equal("Broadcast message")
	case <-timeout:
		t.Fatal("Client 2 timeout waiting for broadcast")
	}
}

func TestWebSocketIntegration_SecurityTests(t *testing.T) {
	// Each subtest gets its own context to avoid channel issues

	// Test 1: No authentication
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

	// Test 2: Non-existent ticket
	t.Run("NonExistentTicket", func(t *testing.T) {
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

	// Test 3: Large message handling
	t.Run("LargeMessage", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := repository.NewMemory()
		hub := websocket_ctrl.NewHub(ctx)
		go hub.Run()
		defer hub.Close()

		handler := websocket_ctrl.NewHandler(hub, repo, nil)

		// Create test ticket
		testTicket := ticket.New(ctx, []types.AlertID{}, nil)
		testTicket.ID = types.TicketID("test-ticket")
		err := repo.PutTicket(ctx, testTicket)
		gt.NoError(t, err)

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

		wsURL := strings.Replace(server.URL, "http", "ws", 1) + "/ws/chat/ticket/test-ticket"
		ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		gt.NoError(t, err)
		gt.Value(t, resp.StatusCode).Equal(http.StatusSwitchingProtocols)
		defer ws.Close()

		// Send a message larger than 64KB (using 128KB to ensure it exceeds limit with JSON overhead)
		largeContent := strings.Repeat("x", 128*1024)
		largeMsg := websocket_model.ChatMessage{
			Type:      "message",
			Content:   largeContent,
			Timestamp: time.Now().Unix(),
		}

		err = ws.WriteJSON(largeMsg)
		// Write might succeed, but server will close connection when trying to read
		gt.NoError(t, err)

		// Wait for server to process and close connection
		time.Sleep(100 * time.Millisecond)

		// Try to send ping - should fail due to closed connection
		pingMsg := websocket_model.ChatMessage{
			Type:      "ping",
			Timestamp: time.Now().Unix(),
		}
		err = ws.WriteJSON(pingMsg)
		gt.Error(t, err) // Connection should be closed
	})
}

func TestWebSocketIntegration_ConnectionHandling(t *testing.T) {
	// Test ping/pong
	t.Run("PingPong", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := repository.NewMemory()
		hub := websocket_ctrl.NewHub(ctx)
		go hub.Run()
		defer hub.Close()

		// Create test ticket
		testTicket := ticket.New(ctx, []types.AlertID{}, nil)
		testTicket.ID = types.TicketID("test-ticket")
		err := repo.PutTicket(ctx, testTicket)
		gt.NoError(t, err)

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

		wsURL := strings.Replace(server.URL, "http", "ws", 1) + "/ws/chat/ticket/test-ticket"
		ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		gt.NoError(t, err)
		gt.Value(t, resp.StatusCode).Equal(http.StatusSwitchingProtocols)
		defer ws.Close()

		// Skip initial status message
		var status websocket_model.ChatResponse
		err = ws.ReadJSON(&status)
		gt.NoError(t, err)
		gt.Value(t, status.Type).Equal("status")

		// Send ping message
		pingMsg := websocket_model.ChatMessage{
			Type:      "ping",
			Timestamp: time.Now().Unix(),
		}
		err = ws.WriteJSON(pingMsg)
		gt.NoError(t, err)

		// Should receive pong
		var pong websocket_model.ChatResponse
		err = ws.ReadJSON(&pong)
		gt.NoError(t, err)
		gt.Value(t, pong.Type).Equal("pong")
	})

	// Test clean disconnect
	t.Run("CleanDisconnect", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := repository.NewMemory()
		hub := websocket_ctrl.NewHub(ctx)
		go hub.Run()
		defer hub.Close()

		// Create test ticket
		testTicket := ticket.New(ctx, []types.AlertID{}, nil)
		testTicket.ID = types.TicketID("test-ticket")
		err := repo.PutTicket(ctx, testTicket)
		gt.NoError(t, err)

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

		wsURL := strings.Replace(server.URL, "http", "ws", 1) + "/ws/chat/ticket/test-ticket"
		ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		gt.NoError(t, err)
		gt.Value(t, resp.StatusCode).Equal(http.StatusSwitchingProtocols)

		// Wait for connection to be registered
		time.Sleep(50 * time.Millisecond)

		// Get initial client count
		initialCount := hub.GetClientCount(testTicket.ID)
		gt.Value(t, initialCount).Equal(1)

		// Close connection
		err = ws.Close()
		gt.NoError(t, err)

		// Wait for cleanup
		time.Sleep(100 * time.Millisecond)

		// Verify client was removed
		finalCount := hub.GetClientCount(testTicket.ID)
		gt.Value(t, finalCount).Equal(0)
	})
}
