package usecase_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase"
)

func TestGenerateInitialTicketComment(t *testing.T) {
	ctx := context.Background()
	ctx = lang.With(ctx, lang.Japanese) // Test with Japanese language

	// Create test ticket
	testTicket := ticket.Ticket{
		ID:       types.NewTicketID(),
		AlertIDs: []types.AlertID{types.NewAlertID()},
		SlackThread: &slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  "test-thread",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    types.TicketStatusOpen,
	}
	testTicket.Title = "Suspicious Login Activity"
	testTicket.Description = "Multiple failed login attempts detected from unusual locations"
	testTicket.Summary = "Potential brute force attack detected on user accounts"
	testTicket.Assignee = &slack.User{
		ID:   "U123456",
		Name: "John Doe",
	}

	// Create test alerts
	testAlerts := alert.Alerts{
		&alert.Alert{
			ID:     types.NewAlertID(),
			Schema: "login_monitor",
			Data:   map[string]interface{}{"source_ip": "192.168.1.100", "user": "admin"},
		},
		&alert.Alert{
			ID:     types.NewAlertID(),
			Schema: "auth_failure",
			Data:   map[string]interface{}{"attempts": 5, "user": "admin"},
		},
	}
	testAlerts[0].Title = "Failed Login Alert"
	testAlerts[1].Title = "Authentication Failure"

	// Mock LLM client
	llmMock := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{"お疲れさまです！このチケットについて一緒に調査してみませんか？🔍 何か気になる点があれば気軽に共有してください。"},
					}, nil
				},
			}, nil
		},
	}

	// Create repository
	repo := repository.NewMemory()

	// Create UseCases instance
	uc := usecase.New(
		usecase.WithLLMClient(llmMock),
		usecase.WithRepository(repo),
	)

	// Test the generateInitialTicketComment function
	comment, err := uc.GenerateInitialTicketCommentForTest(ctx, &testTicket, testAlerts)

	// Verify results
	gt.NoError(t, err)
	gt.Value(t, comment).NotEqual("")

	// Check that the comment contains expected Japanese content
	gt.Value(t, strings.Contains(comment, "調査")).Equal(true)
	gt.Value(t, strings.Contains(comment, "🔍")).Equal(true)

	// Verify LLM was called
	gt.Array(t, llmMock.NewSessionCalls()).Length(1)
}

func TestGenerateInitialTicketComment_English(t *testing.T) {
	ctx := context.Background()
	ctx = lang.With(ctx, lang.English) // Test with English language

	// Create test ticket
	testTicket := ticket.Ticket{
		ID:       types.NewTicketID(),
		AlertIDs: []types.AlertID{types.NewAlertID()},
		SlackThread: &slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  "test-thread",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    types.TicketStatusOpen,
	}
	testTicket.Title = "Network Anomaly Detected"
	testTicket.Description = "Unusual network traffic patterns observed"
	testTicket.Summary = "Potential data exfiltration attempt detected"

	// Create test alerts
	testAlerts := alert.Alerts{
		&alert.Alert{
			ID:     types.NewAlertID(),
			Schema: "network_monitor",
			Data:   map[string]interface{}{"bytes_transferred": 10000000},
		},
	}
	testAlerts[0].Title = "Network Traffic Alert"

	// Mock LLM client
	llmMock := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{"Thanks for creating this ticket! Let's investigate this network anomaly together. 🔍 Feel free to share any initial observations."},
					}, nil
				},
			}, nil
		},
	}

	// Create repository
	repo := repository.NewMemory()

	// Create UseCases instance
	uc := usecase.New(
		usecase.WithLLMClient(llmMock),
		usecase.WithRepository(repo),
	)

	// Test the generateInitialTicketComment function
	comment, err := uc.GenerateInitialTicketCommentForTest(ctx, &testTicket, testAlerts)

	// Verify results
	gt.NoError(t, err)
	gt.Value(t, comment).NotEqual("")

	// Check that the comment contains expected English content
	gt.Value(t, strings.Contains(comment, "investigate")).Equal(true)
	gt.Value(t, strings.Contains(comment, "🔍")).Equal(true)

	// Verify LLM was called
	gt.Array(t, llmMock.NewSessionCalls()).Length(1)
}

func TestGenerateInitialTicketComment_LLMError(t *testing.T) {
	ctx := context.Background()

	// Create test ticket
	testTicket := ticket.Ticket{
		ID:       types.NewTicketID(),
		AlertIDs: []types.AlertID{types.NewAlertID()},
		Status:   types.TicketStatusOpen,
	}
	testTicket.Title = "Test Ticket"
	testTicket.Description = "Test Description"
	testTicket.Summary = "Test Summary"

	// Create test alerts
	testAlerts := alert.Alerts{
		&alert.Alert{
			ID:     types.NewAlertID(),
			Schema: "test_schema",
		},
	}

	// Mock LLM client that returns error
	llmMock := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return nil, goerr.New("LLM generation failed")
				},
			}, nil
		},
	}

	// Create repository
	repo := repository.NewMemory()

	// Create UseCases instance
	uc := usecase.New(
		usecase.WithLLMClient(llmMock),
		usecase.WithRepository(repo),
	)

	// Test the generateInitialTicketComment function
	comment, err := uc.GenerateInitialTicketCommentForTest(ctx, &testTicket, testAlerts)

	// Verify error handling
	gt.Error(t, err)
	gt.Value(t, comment).Equal("")
	gt.Value(t, strings.Contains(err.Error(), "failed to generate comment")).Equal(true)
}

func TestGenerateInitialTicketComment_EmptyResponse(t *testing.T) {
	ctx := context.Background()

	// Create test ticket
	testTicket := ticket.Ticket{
		ID:       types.NewTicketID(),
		AlertIDs: []types.AlertID{types.NewAlertID()},
		Status:   types.TicketStatusOpen,
	}
	testTicket.Title = "Test Ticket"
	testTicket.Description = "Test Description"
	testTicket.Summary = "Test Summary"

	// Create test alerts
	testAlerts := alert.Alerts{
		&alert.Alert{
			ID:     types.NewAlertID(),
			Schema: "test_schema",
		},
	}

	// Mock LLM client that returns empty response
	llmMock := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{}, // Empty response
					}, nil
				},
			}, nil
		},
	}

	// Create repository
	repo := repository.NewMemory()

	// Create UseCases instance
	uc := usecase.New(
		usecase.WithLLMClient(llmMock),
		usecase.WithRepository(repo),
	)

	// Test the generateInitialTicketComment function
	comment, err := uc.GenerateInitialTicketCommentForTest(ctx, &testTicket, testAlerts)

	// Verify error handling for empty response
	gt.Error(t, err)
	gt.Value(t, comment).Equal("")
	gt.Value(t, strings.Contains(err.Error(), "no comment generated")).Equal(true)
}
