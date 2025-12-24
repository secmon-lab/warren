package session_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func TestNewMessage(t *testing.T) {
	ctx := context.Background()
	sessionID := types.NewSessionID()
	content := "test message content"

	msg := session.NewMessage(ctx, sessionID, session.MessageTypeTrace, content)

	gt.V(t, msg.ID).NotEqual(types.MessageID(""))
	gt.V(t, msg.SessionID).Equal(sessionID)
	gt.V(t, msg.Type).Equal(session.MessageTypeTrace)
	gt.V(t, msg.Content).Equal(content)
	gt.V(t, msg.CreatedAt.IsZero()).Equal(false)
	gt.V(t, msg.UpdatedAt.IsZero()).Equal(false)
	gt.V(t, msg.CreatedAt).Equal(msg.UpdatedAt)
}

func TestMessageTypes(t *testing.T) {
	testCases := []struct {
		name     string
		msgType  session.MessageType
		expected session.MessageType
	}{
		{
			name:     "trace type",
			msgType:  session.MessageTypeTrace,
			expected: "trace",
		},
		{
			name:     "plan type",
			msgType:  session.MessageTypePlan,
			expected: "plan",
		},
		{
			name:     "response type",
			msgType:  session.MessageTypeResponse,
			expected: "response",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gt.Equal(t, tc.msgType, tc.expected)
		})
	}
}
