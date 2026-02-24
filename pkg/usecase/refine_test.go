package usecase_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/refine"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase"
)

func newMockLLMWithResponse(responseJSON string) *mock.LLMClientMock {
	return &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{responseJSON},
					}, nil
				},
			}, nil
		},
	}
}

func TestRefine_NoLLMClient(t *testing.T) {
	uc := usecase.New()
	err := uc.Refine(context.Background())
	gt.Error(t, err)
}

func TestRefine_NoOpenTickets(t *testing.T) {
	repo := repository.NewMemory()
	llmMock := newMockLLMWithResponse(`{"message": "", "reason": "no tickets"}`)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithLLMClient(llmMock),
	)

	err := uc.Refine(context.Background())
	gt.NoError(t, err)
}

func TestRefine_ReviewOpenTicket_NoAction(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	// Create an open ticket
	ticketData := ticket.Ticket{
		ID:        types.NewTicketID(),
		Status:    types.TicketStatusOpen,
		CreatedAt: time.Now().Add(-24 * time.Hour),
		Metadata: ticket.Metadata{
			Title:       "Test Ticket",
			Description: "Test description",
		},
	}
	gt.NoError(t, repo.PutTicket(ctx, ticketData))

	// LLM says "do not act"
	llmMock := newMockLLMWithResponse(`{"message": "", "reason": "recent activity detected"}`)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithLLMClient(llmMock),
	)

	err := uc.Refine(ctx)
	gt.NoError(t, err)

	// Verify LLM was called (for ticket review)
	gt.V(t, len(llmMock.NewSessionCalls())).Equal(1)
}

func TestRefine_ReviewOpenTicket_WithAction(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	// Create an open ticket without Slack thread (CLI mode)
	ticketData := ticket.Ticket{
		ID:        types.NewTicketID(),
		Status:    types.TicketStatusOpen,
		CreatedAt: time.Now().Add(-72 * time.Hour),
		Metadata: ticket.Metadata{
			Title:       "Stagnant Ticket",
			Description: "No progress for days",
		},
	}
	gt.NoError(t, repo.PutTicket(ctx, ticketData))

	// LLM says "act"
	llmMock := newMockLLMWithResponse(`{"message": "Please check on this ticket", "reason": "no activity for 3 days"}`)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithLLMClient(llmMock),
	)

	err := uc.Refine(ctx)
	gt.NoError(t, err)

	// Verify LLM was called
	gt.V(t, len(llmMock.NewSessionCalls())).Equal(1)
}

func TestRefine_ReviewOpenTicket_WithActionButEmptyMessage(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	// Create an open ticket without Slack thread (CLI mode)
	ticketData := ticket.Ticket{
		ID:        types.NewTicketID(),
		Status:    types.TicketStatusOpen,
		CreatedAt: time.Now().Add(-72 * time.Hour),
		Metadata: ticket.Metadata{
			Title:       "Stagnant Ticket",
			Description: "No progress for days",
		},
	}
	gt.NoError(t, repo.PutTicket(ctx, ticketData))

	// LLM returns empty message
	llmMock := newMockLLMWithResponse(`{"message": "", "reason": ""}`)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithLLMClient(llmMock),
	)

	// Should not error with empty message
	err := uc.Refine(ctx)
	gt.NoError(t, err)
}

func TestRefine_ConsolidateUnboundAlerts_NotEnoughAlerts(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	// Only 1 unbound alert - not enough for consolidation
	a := alert.Alert{
		ID:        types.NewAlertID(),
		Schema:    "test-schema",
		Status:    alert.AlertStatusActive,
		CreatedAt: time.Now(),
		Data:      map[string]any{"key": "value"},
		Metadata: alert.Metadata{
			Title:       "Single Alert",
			Description: "Only one alert",
		},
	}
	gt.NoError(t, repo.PutAlert(ctx, a))

	llmMock := newMockLLMWithResponse(`{"message": "", "reason": ""}`)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithLLMClient(llmMock),
	)

	err := uc.Refine(ctx)
	gt.NoError(t, err)
}

func TestRefine_ConsolidateUnboundAlerts_WithGroups(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	// Create 3 unbound alerts
	alertIDs := make([]types.AlertID, 3)
	for i := range 3 {
		a := alert.Alert{
			ID:        types.NewAlertID(),
			Schema:    "test-schema",
			Status:    alert.AlertStatusActive,
			CreatedAt: time.Now(),
			Data:      map[string]any{"source_ip": "192.168.1.1"},
			Metadata: alert.Metadata{
				Title:       "Brute Force Alert",
				Description: "Brute force detected",
			},
			SlackThread: &slack.Thread{
				ChannelID: "C123",
				ThreadID:  "1234567890.123456",
			},
		}
		gt.NoError(t, repo.PutAlert(ctx, a))
		alertIDs[i] = a.ID
	}

	// Multi-step LLM responses: ticket review (0 tickets), then alert summaries (3x), then consolidation
	callCount := 0
	llmMock := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					callCount++
					var response string
					if callCount <= 3 {
						// Alert summary responses
						response = `{"alert_id": "` + alertIDs[callCount-1].String() + `", "title": "Brute Force", "identities": ["192.168.1.1"], "parameters": ["ssh"], "context": "network", "root_cause": "brute force"}`
					} else {
						// Consolidation response
						consolidation := map[string]any{
							"groups": []map[string]any{
								{
									"reason":           "same source IP brute force",
									"primary_alert_id": alertIDs[0].String(),
									"alert_ids":        []string{alertIDs[0].String(), alertIDs[1].String(), alertIDs[2].String()},
								},
							},
						}
						data, _ := json.Marshal(consolidation)
						response = string(data)
					}
					return &gollem.Response{
						Texts: []string{response},
					}, nil
				},
			}, nil
		},
	}

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithLLMClient(llmMock),
	)

	err := uc.Refine(ctx)
	gt.NoError(t, err)

	// Verify LLM was called: 3 summaries + 1 consolidation = 4 sessions
	gt.V(t, len(llmMock.NewSessionCalls())).Equal(4)
}

func TestRefine_ConsolidateUnboundAlerts_NoGroups(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	// Create 2 unbound alerts
	for range 2 {
		a := alert.Alert{
			ID:        types.NewAlertID(),
			Schema:    "test-schema",
			Status:    alert.AlertStatusActive,
			CreatedAt: time.Now(),
			Data:      map[string]any{"key": "value"},
			Metadata: alert.Metadata{
				Title:       "Random Alert",
				Description: "Random description",
			},
		}
		gt.NoError(t, repo.PutAlert(ctx, a))
	}

	// Multi-step responses: 2 summaries + empty consolidation
	callCount := 0
	llmMock := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					callCount++
					var response string
					if callCount <= 2 {
						response = `{"alert_id": "test", "title": "Alert", "identities": [], "parameters": [], "context": "test", "root_cause": "unknown"}`
					} else {
						response = `{"groups": []}`
					}
					return &gollem.Response{
						Texts: []string{response},
					}, nil
				},
			}, nil
		},
	}

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithLLMClient(llmMock),
	)

	err := uc.Refine(ctx)
	gt.NoError(t, err)
}

func TestHandleCreateTicketFromRefine(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	// Create alerts
	alertID1 := types.NewAlertID()
	alertID2 := types.NewAlertID()

	for _, id := range []types.AlertID{alertID1, alertID2} {
		a := alert.Alert{
			ID:        id,
			Schema:    "test-schema",
			Status:    alert.AlertStatusActive,
			CreatedAt: time.Now(),
			Data:      map[string]any{"key": "value"},
			Metadata: alert.Metadata{
				Title:       "Test Alert " + id.String(),
				Description: "Test description",
			},
		}
		gt.NoError(t, repo.PutAlert(ctx, a))
	}

	// Create a refine group
	group := &refine.Group{
		ID:             types.NewRefineGroupID(),
		PrimaryAlertID: alertID1,
		AlertIDs:       []types.AlertID{alertID1, alertID2},
		Reason:         "test consolidation",
		CreatedAt:      time.Now(),
		Status:         refine.GroupStatusPending,
	}
	gt.NoError(t, repo.PutRefineGroup(ctx, group))

	// LLM mock for ticket metadata generation
	llmMock := newMockLLMWithResponse(`{"title": "Consolidated Ticket", "description": "Merged alerts"}`)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithLLMClient(llmMock),
	)

	// Simulate button press
	slackUser := slack.User{ID: "U123", Name: "test-user"}
	slackThread := slack.Thread{ChannelID: "C123", ThreadID: "1234567890.123456"}

	err := uc.HandleSlackInteractionBlockActions(ctx, slackUser, slackThread, slack.ActionIDCreateTicketFromRefine, string(group.ID), "trigger-123")
	// This will fail because slackService is nil, but the important thing is
	// it recognizes the action ID and attempts to process it
	gt.Error(t, err) // "slack service not configured"

	// Verify the group status was NOT updated (because slackService was nil and it returned early)
	got, err := repo.GetRefineGroup(ctx, group.ID)
	gt.NoError(t, err)
	gt.V(t, got.Status).Equal(refine.GroupStatusPending)
}
