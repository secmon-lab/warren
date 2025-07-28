package websocket_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/websocket"
)

func TestChatMessage_FromBytes(t *testing.T) {
	testCases := []struct {
		name     string
		jsonData string
		expected websocket.ChatMessage
		wantErr  bool
	}{
		{
			name:     "valid message",
			jsonData: `{"type":"message","content":"Hello","timestamp":1234567890}`,
			expected: websocket.ChatMessage{
				Type:      "message",
				Content:   "Hello",
				Timestamp: 1234567890,
			},
			wantErr: false,
		},
		{
			name:     "valid ping",
			jsonData: `{"type":"ping","content":"","timestamp":1234567890}`,
			expected: websocket.ChatMessage{
				Type:      "ping",
				Content:   "",
				Timestamp: 1234567890,
			},
			wantErr: false,
		},
		{
			name:     "invalid json",
			jsonData: `{"type":"message","content":}`,
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var msg websocket.ChatMessage
			err := msg.FromBytes([]byte(tc.jsonData))

			if tc.wantErr {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
				gt.Value(t, msg.Type).Equal(tc.expected.Type)
				gt.Value(t, msg.Content).Equal(tc.expected.Content)
				gt.Value(t, msg.Timestamp).Equal(tc.expected.Timestamp)
			}
		})
	}
}

func TestChatResponse_ToBytes(t *testing.T) {
	user := &websocket.User{
		ID:   "user123",
		Name: "Test User",
	}

	resp := &websocket.ChatResponse{
		Type:      "message",
		Content:   "Hello World",
		User:      user,
		Timestamp: 1234567890,
		MessageID: "msg123",
	}

	bytes, err := resp.ToBytes()
	gt.NoError(t, err)

	// Parse back to verify structure
	var parsed map[string]interface{}
	err = json.Unmarshal(bytes, &parsed)
	gt.NoError(t, err)

	gt.Value(t, parsed["type"]).Equal("message")
	gt.Value(t, parsed["content"]).Equal("Hello World")
	gt.Value(t, parsed["timestamp"]).Equal(float64(1234567890))
	gt.Value(t, parsed["message_id"]).Equal("msg123")

	userMap := parsed["user"].(map[string]interface{})
	gt.Value(t, userMap["id"]).Equal("user123")
	gt.Value(t, userMap["name"]).Equal("Test User")
}

func TestNewChatResponse(t *testing.T) {
	resp := websocket.NewChatResponse("status", "Connected")

	gt.Value(t, resp.Type).Equal("status")
	gt.Value(t, resp.Content).Equal("Connected")

	// Timestamp should be recent (within last 10 seconds)
	now := time.Now().Unix()
	gt.Value(t, resp.Timestamp > 0).Equal(true)
	gt.Value(t, resp.Timestamp >= now-10).Equal(true)
	gt.Value(t, resp.Timestamp <= now).Equal(true)
}

func TestNewMessageResponse(t *testing.T) {
	user := &websocket.User{
		ID:   "user456",
		Name: "Jane Doe",
	}

	resp := websocket.NewMessageResponse("Test message", user)

	gt.Value(t, resp.Type).Equal("message")
	gt.Value(t, resp.Content).Equal("Test message")
	gt.Value(t, resp.User).Equal(user)
	gt.Value(t, resp.Timestamp > 0).Equal(true)
}

func TestUserFromSlackUser(t *testing.T) {
	t.Run("valid slack user", func(t *testing.T) {
		slackUser := &slack.User{
			ID:   "slack123",
			Name: "Slack User",
		}

		user := websocket.UserFromSlackUser(slackUser)

		gt.Value(t, user).NotNil()
		gt.Value(t, user.ID).Equal("slack123")
		gt.Value(t, user.Name).Equal("Slack User")
	})

	t.Run("nil slack user", func(t *testing.T) {
		user := websocket.UserFromSlackUser(nil)
		gt.Value(t, user).Nil()
	})
}

func TestChatMessage_IsValidMessageType(t *testing.T) {
	testCases := []struct {
		msgType string
		valid   bool
	}{
		{"message", true},
		{"ping", true},
		{"invalid", false},
		{"", false},
	}

	for _, tc := range testCases {
		t.Run(tc.msgType, func(t *testing.T) {
			msg := websocket.ChatMessage{Type: tc.msgType}
			gt.Value(t, msg.IsValidMessageType()).Equal(tc.valid)
		})
	}
}

func TestChatResponse_IsValidResponseType(t *testing.T) {
	testCases := []struct {
		responseType string
		valid        bool
	}{
		{"message", true},
		{"history", true},
		{"status", true},
		{"error", true},
		{"invalid", false},
		{"", false},
	}

	for _, tc := range testCases {
		t.Run(tc.responseType, func(t *testing.T) {
			resp := websocket.ChatResponse{Type: tc.responseType}
			gt.Value(t, resp.IsValidResponseType()).Equal(tc.valid)
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("NewStatusResponse", func(t *testing.T) {
		resp := websocket.NewStatusResponse("System ready")
		gt.Value(t, resp.Type).Equal("status")
		gt.Value(t, resp.Content).Equal("System ready")
	})

	t.Run("NewErrorResponse", func(t *testing.T) {
		resp := websocket.NewErrorResponse("Authentication failed")
		gt.Value(t, resp.Type).Equal("error")
		gt.Value(t, resp.Content).Equal("Authentication failed")
	})
}
