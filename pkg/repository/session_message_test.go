package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestSessionMessageOperations(t *testing.T) {
	ctx := context.Background()

	testFn := func(t *testing.T, repo interfaces.Repository) {

		// Use random IDs to avoid conflicts in parallel tests
		sessionID := types.SessionID(time.Now().Format("20060102150405.000000"))

		// Create test session first
		sess := &session.Session{
			ID:        sessionID,
			TicketID:  types.TicketID("ticket-" + sessionID.String()),
			RequestID: "req-123",
			Status:    types.SessionStatusRunning,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		gt.NoError(t, repo.PutSession(ctx, sess))

		t.Run("PutSessionMessage and GetSessionMessages", func(t *testing.T) {
			// Create and save messages with small delays to ensure ordering
			msg1 := session.NewMessage(ctx, sessionID, session.MessageTypeTrace, "💭 Thinking...")
			gt.NoError(t, repo.PutSessionMessage(ctx, msg1))

			msg2 := session.NewMessage(ctx, sessionID, session.MessageTypeTrace, "🤖 Executing tool")
			gt.NoError(t, repo.PutSessionMessage(ctx, msg2))

			msg3 := session.NewMessage(ctx, sessionID, session.MessageTypePlan, "🎯 Goal\n☑️ Task1")
			gt.NoError(t, repo.PutSessionMessage(ctx, msg3))

			// Retrieve messages
			messages, err := repo.GetSessionMessages(ctx, sessionID)
			gt.NoError(t, err)
			gt.A(t, messages).Length(3)

			// Verify messages are ordered by created_at
			gt.V(t, messages[0].CreatedAt.Before(messages[1].CreatedAt) || messages[0].CreatedAt.Equal(messages[1].CreatedAt)).Equal(true)
			gt.V(t, messages[1].CreatedAt.Before(messages[2].CreatedAt) || messages[1].CreatedAt.Equal(messages[2].CreatedAt)).Equal(true)

			// Verify message contents
			gt.Equal(t, messages[0].Content, "💭 Thinking...")
			gt.Equal(t, messages[0].Type, session.MessageTypeTrace)
			gt.Equal(t, messages[1].Content, "🤖 Executing tool")
			gt.Equal(t, messages[1].Type, session.MessageTypeTrace)
			gt.Equal(t, messages[2].Content, "🎯 Goal\n☑️ Task1")
			gt.Equal(t, messages[2].Type, session.MessageTypePlan)

			// Verify all fields are properly set
			for _, msg := range messages {
				gt.V(t, msg.ID).NotEqual(types.MessageID(""))
				gt.V(t, msg.SessionID).Equal(sessionID)
				gt.V(t, msg.CreatedAt.IsZero()).Equal(false)
				gt.V(t, msg.UpdatedAt.IsZero()).Equal(false)
			}
		})

		t.Run("GetSessionMessages for non-existent session", func(t *testing.T) {
			nonExistentID := types.SessionID("non-existent-" + time.Now().Format("20060102150405.000000"))
			messages, err := repo.GetSessionMessages(ctx, nonExistentID)
			gt.NoError(t, err)
			gt.A(t, messages).Length(0)
		})

		t.Run("Messages with different types", func(t *testing.T) {
			sessionID2 := types.SessionID(time.Now().Format("20060102150405.000001"))

			// Create another session
			sess2 := &session.Session{
				ID:        sessionID2,
				TicketID:  types.TicketID("ticket-" + sessionID2.String()),
				RequestID: "req-456",
				Status:    types.SessionStatusRunning,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			gt.NoError(t, repo.PutSession(ctx, sess2))

			// Add messages of all types
			trace := session.NewMessage(ctx, sessionID2, session.MessageTypeTrace, "trace msg")
			gt.NoError(t, repo.PutSessionMessage(ctx, trace))

			plan := session.NewMessage(ctx, sessionID2, session.MessageTypePlan, "plan msg")
			gt.NoError(t, repo.PutSessionMessage(ctx, plan))

			response := session.NewMessage(ctx, sessionID2, session.MessageTypeResponse, "response msg")
			gt.NoError(t, repo.PutSessionMessage(ctx, response))

			// Retrieve and verify
			messages, err := repo.GetSessionMessages(ctx, sessionID2)
			gt.NoError(t, err)
			gt.A(t, messages).Length(3)

			gt.Equal(t, messages[0].Type, session.MessageTypeTrace)
			gt.Equal(t, messages[1].Type, session.MessageTypePlan)
			gt.Equal(t, messages[2].Type, session.MessageTypeResponse)
		})

		t.Run("Multiple messages preserve order", func(t *testing.T) {
			sessionID3 := types.SessionID(time.Now().Format("20060102150405.000002"))

			// Create another session
			sess3 := &session.Session{
				ID:        sessionID3,
				TicketID:  types.TicketID("ticket-" + sessionID3.String()),
				RequestID: "req-789",
				Status:    types.SessionStatusRunning,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			gt.NoError(t, repo.PutSession(ctx, sess3))

			// Add many messages and verify ordering by created_at
			for i := 1; i <= 10; i++ {
				msg := session.NewMessage(ctx, sessionID3, session.MessageTypeTrace, "msg-"+string(rune('0'+i)))
				gt.NoError(t, repo.PutSessionMessage(ctx, msg))
			}

			// Retrieve and verify order
			messages, err := repo.GetSessionMessages(ctx, sessionID3)
			gt.NoError(t, err)
			gt.A(t, messages).Length(10)

			for i := 0; i < 9; i++ {
				gt.V(t, messages[i].CreatedAt.Before(messages[i+1].CreatedAt) || messages[i].CreatedAt.Equal(messages[i+1].CreatedAt)).Equal(true)
			}
		})
	}

	t.Run("memory repository", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})

	t.Run("firestore repository", func(t *testing.T) {
		repo := newFirestoreClient(t)
		testFn(t, repo)
	})
}
