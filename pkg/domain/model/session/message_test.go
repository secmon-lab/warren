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
			name:     "user type",
			msgType:  session.MessageTypeUser,
			expected: "user",
		},
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

func TestMessageType_Valid(t *testing.T) {
	gt.V(t, session.MessageTypeUser.Valid()).Equal(true)
	gt.V(t, session.MessageTypeTrace.Valid()).Equal(true)
	gt.V(t, session.MessageTypePlan.Valid()).Equal(true)
	gt.V(t, session.MessageTypeResponse.Valid()).Equal(true)
	gt.V(t, session.MessageTypeWarning.Valid()).Equal(true)
	gt.V(t, session.MessageType("bogus").Valid()).Equal(false)
	gt.V(t, session.MessageType("").Valid()).Equal(false)
}

func TestNewMessageV2_UserTypeCarriesAuthor(t *testing.T) {
	ctx := context.Background()
	tid := types.TicketID("tid_1")
	turn := types.TurnID("turn_1")
	author := &session.Author{
		UserID:      types.UserID("u1"),
		DisplayName: "Alice",
	}
	msg := session.NewMessageV2(ctx,
		types.SessionID("sid_1"), &tid, &turn,
		session.MessageTypeUser, "hi", author,
	)

	gt.V(t, msg.Type).Equal(session.MessageTypeUser)
	gt.V(t, msg.Author == nil).Equal(false)
	gt.V(t, msg.Author.DisplayName).Equal("Alice")
	gt.V(t, msg.TicketID != nil && *msg.TicketID == tid).Equal(true)
	gt.V(t, msg.TurnID != nil && *msg.TurnID == turn).Equal(true)
}

func TestNewMessageV2_AIType_NoAuthor_NilTurnOK(t *testing.T) {
	ctx := context.Background()
	msg := session.NewMessageV2(ctx,
		types.SessionID("sid_1"), nil, nil,
		session.MessageTypeTrace, "thinking...", nil,
	)

	gt.V(t, msg.Type).Equal(session.MessageTypeTrace)
	gt.V(t, msg.Author == nil).Equal(true)
	gt.V(t, msg.TicketID == nil).Equal(true)
	gt.V(t, msg.TurnID == nil).Equal(true)
}

func TestNewMessageV2_NonMentionUser_NilTurn(t *testing.T) {
	// Slack non-mention thread messages save Messages with TurnID=nil.
	ctx := context.Background()
	tid := types.TicketID("tid_1")
	author := &session.Author{UserID: types.UserID("u1"), DisplayName: "Bob"}

	msg := session.NewMessageV2(ctx,
		types.SessionID("sid_1"), &tid, nil,
		session.MessageTypeUser, "additional context", author,
	)

	gt.V(t, msg.Type).Equal(session.MessageTypeUser)
	gt.V(t, msg.Author == nil).Equal(false)
	gt.V(t, msg.TurnID == nil).Equal(true)
	gt.V(t, msg.TicketID != nil).Equal(true)
}
