package abort_test

import (
	"context"
	"testing"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	alertmodel "github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	ticketmodel "github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository/memory"
	"github.com/secmon-lab/warren/pkg/service/command/abort"
	"github.com/secmon-lab/warren/pkg/service/command/core"
)

// mockThreadService is a mock implementation of SlackThreadService
type mockThreadService struct {
	messages []string
	thread   slack.Thread
}

func (m *mockThreadService) ChannelID() string { return m.thread.ChannelID }
func (m *mockThreadService) ThreadID() string  { return m.thread.ThreadID }
func (m *mockThreadService) Entity() *slack.Thread {
	return &m.thread
}

func (m *mockThreadService) PostAlert(ctx context.Context, alert *alertmodel.Alert) error {
	return nil
}

func (m *mockThreadService) PostComment(ctx context.Context, message string) error {
	m.messages = append(m.messages, message)
	return nil
}

func (m *mockThreadService) PostCommentWithMessageID(ctx context.Context, message string) (string, error) {
	m.messages = append(m.messages, message)
	return "M" + string(rune(len(m.messages))), nil
}

func (m *mockThreadService) PostContextBlock(ctx context.Context, text string) error {
	return nil
}

func (m *mockThreadService) PostTicket(ctx context.Context, ticket *ticketmodel.Ticket, alerts alertmodel.Alerts) (string, error) {
	return "", nil
}

func (m *mockThreadService) PostLinkToTicket(ctx context.Context, ticketURL, ticketTitle string) error {
	return nil
}

func (m *mockThreadService) PostFinding(ctx context.Context, finding *ticketmodel.Finding) error {
	return nil
}

func (m *mockThreadService) UpdateAlert(ctx context.Context, alert alertmodel.Alert) error {
	return nil
}

func (m *mockThreadService) UpdateAlertList(ctx context.Context, list *alertmodel.List, status string) error {
	return nil
}

func (m *mockThreadService) PostAlerts(ctx context.Context, alerts alertmodel.Alerts) error {
	return nil
}

func (m *mockThreadService) PostAlertList(ctx context.Context, list *alertmodel.List) (string, error) {
	return "", nil
}

func (m *mockThreadService) PostAlertLists(ctx context.Context, clusters []*alertmodel.List) error {
	return nil
}

func (m *mockThreadService) PostTicketList(ctx context.Context, tickets []*ticketmodel.Ticket) error {
	return nil
}

func (m *mockThreadService) Reply(ctx context.Context, message string) {
	_ = m.PostComment(ctx, message)
}

func (m *mockThreadService) NewStateFunc(ctx context.Context, message string) func(ctx context.Context, msg string) {
	_ = m.PostComment(ctx, message)
	return func(_ context.Context, _ string) {}
}

func (m *mockThreadService) NewUpdatableMessage(ctx context.Context, initialMessage string) func(ctx context.Context, newMsg string) {
	_ = m.PostComment(ctx, initialMessage)
	return func(_ context.Context, _ string) {}
}

func (m *mockThreadService) NewTraceMessage(ctx context.Context, initialMessage string) func(ctx context.Context, traceMsg string) {
	_ = m.PostComment(ctx, initialMessage)
	return func(_ context.Context, _ string) {}
}

func (m *mockThreadService) AttachFile(ctx context.Context, title, fileName string, data []byte) error {
	return nil
}

var _ interfaces.SlackThreadService = &mockThreadService{}

func TestAbortCommand(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()

	// Create ticket with slack thread
	thread := &slack.Thread{
		ChannelID: "C123",
		ThreadID:  "M456",
	}

	tk := ticketmodel.New(ctx, []types.AlertID{}, thread)
	tk.Title = "Test Ticket"

	if err := repo.PutTicket(ctx, tk); err != nil {
		t.Fatalf("Failed to put ticket: %v", err)
	}

	// Create running session
	sess := session.NewSession(ctx, tk.ID, "", "", "")
	if err := repo.PutSession(ctx, sess); err != nil {
		t.Fatalf("Failed to put session: %v", err)
	}

	// Create mock thread service
	threadSvc := &mockThreadService{thread: *thread}

	// Create clients
	clients := core.NewClients(repo, nil, threadSvc)

	// Create Slack message using the test helper - message in the thread
	slackMsg := slack.NewTestMessageInThread(*thread, "U123", "abort")

	t.Run("abort running session", func(t *testing.T) {
		err := abort.Execute(ctx, clients, &slackMsg, "")
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Verify session status was updated
		retrievedSess, err := repo.GetSession(ctx, sess.ID)
		if err != nil {
			t.Fatalf("Failed to get session: %v", err)
		}

		if retrievedSess.Status != types.SessionStatusAborted {
			t.Errorf("Session status = %v, want %v", retrievedSess.Status, types.SessionStatusAborted)
		}

		// Verify confirmation message was posted
		if len(threadSvc.messages) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(threadSvc.messages))
		}

		expectedMsg := "🛑 Session aborted. The agent will stop at the next checkpoint."
		if threadSvc.messages[0] != expectedMsg {
			t.Errorf("Message = %q, want %q", threadSvc.messages[0], expectedMsg)
		}
	})

	t.Run("no running session", func(t *testing.T) {
		// Clear messages
		threadSvc.messages = nil

		// Complete the session
		sess.UpdateStatus(ctx, types.SessionStatusCompleted)
		if err := repo.PutSession(ctx, sess); err != nil {
			t.Fatalf("Failed to update session: %v", err)
		}

		err := abort.Execute(ctx, clients, &slackMsg, "")
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Verify info message was posted
		if len(threadSvc.messages) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(threadSvc.messages))
		}

		expectedMsg := "ℹ️ No running session found for this ticket."
		if threadSvc.messages[0] != expectedMsg {
			t.Errorf("Message = %q, want %q", threadSvc.messages[0], expectedMsg)
		}
	})

	t.Run("no ticket found", func(t *testing.T) {
		// Clear messages
		threadSvc.messages = nil

		// Create message for non-existent thread
		nonExistentThread := slack.Thread{ChannelID: "C999", ThreadID: "nonexistent", TeamID: "T123"}
		nonExistentMsg := slack.NewTestMessageInThread(nonExistentThread, "U123", "abort")

		err := abort.Execute(ctx, clients, &nonExistentMsg, "")
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Verify error message was posted
		if len(threadSvc.messages) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(threadSvc.messages))
		}

		expectedMsg := "😣 No ticket found for this thread."
		if threadSvc.messages[0] != expectedMsg {
			t.Errorf("Message = %q, want %q", threadSvc.messages[0], expectedMsg)
		}
	})
}
