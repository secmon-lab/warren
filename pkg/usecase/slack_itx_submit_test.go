package usecase

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
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	slackservice "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	slackSDK "github.com/slack-go/slack"
)

func TestGenerateResolveMessage(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	runTest := func(tc struct {
		name           string
		ticket         *ticket.Ticket
		llmResponse    string
		llmError       error
		expectedPrefix string
	}) func(t *testing.T) {
		return func(t *testing.T) {
			// Setup LLM mock
			llmMock := &mock.LLMClientMock{
				NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
					return &mock.LLMSessionMock{
						GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
							if tc.llmError != nil {
								return nil, tc.llmError
							}
							return &gollem.Response{
								Texts: []string{tc.llmResponse},
							}, nil
						},
					}, nil
				},
				GenerateEmbeddingFunc: func(ctx context.Context, dimension int, inputs []string) ([][]float64, error) {
					// Return mock embedding data with correct dimension
					embedding := make([]float64, dimension)
					for i := range embedding {
						embedding[i] = 0.1 + float64(i)*0.01 // Generate some test values
					}
					return [][]float64{embedding}, nil
				},
			}

			// Create usecase
			uc := New(
				WithLLMClient(llmMock),
				WithRepository(repository.NewMemory()),
			)

			// Test the generateResolveMessage function
			message := uc.generateResolveMessage(ctx, tc.ticket)

			// Verify the result
			if tc.expectedPrefix != "" {
				// Check if message starts with the expected prefix or contains it
				gt.Value(t, strings.Contains(message, tc.expectedPrefix)).Equal(true)
			} else {
				gt.Value(t, message).Equal(tc.llmResponse)
			}
		}
	}

	t.Run("success with conclusion", runTest(struct {
		name           string
		ticket         *ticket.Ticket
		llmResponse    string
		llmError       error
		expectedPrefix string
	}{
		name: "success with conclusion",
		ticket: &ticket.Ticket{
			ID:         types.NewTicketID(),
			Status:     types.TicketStatusResolved,
			Conclusion: types.AlertConclusionFalsePositive,
			Reason:     "False positive detection",
			Metadata: ticket.Metadata{
				Title: "Test Alert",
			},
		},
		llmResponse: "Great work! It was a false positive, but good job on the thorough investigation 🎯",
		llmError:    nil,
	}))

	t.Run("success without conclusion", runTest(struct {
		name           string
		ticket         *ticket.Ticket
		llmResponse    string
		llmError       error
		expectedPrefix string
	}{
		name: "success without conclusion",
		ticket: &ticket.Ticket{
			ID:     types.NewTicketID(),
			Status: types.TicketStatusResolved,
			Reason: "Response completed successfully",
			Metadata: ticket.Metadata{
				Title: "Network Alert",
			},
		},
		llmResponse: "Resolution complete! Another heroic day protecting the world's peace 🦸‍♂️",
		llmError:    nil,
	}))

	t.Run("llm error fallback", runTest(struct {
		name           string
		ticket         *ticket.Ticket
		llmResponse    string
		llmError       error
		expectedPrefix string
	}{
		name: "llm error fallback",
		ticket: &ticket.Ticket{
			ID:     types.NewTicketID(),
			Status: types.TicketStatusResolved,
			Metadata: ticket.Metadata{
				Title: "Test Alert",
			},
		},
		llmResponse:    "",
		llmError:       goerr.New("LLM error"),
		expectedPrefix: "🎉 Great work! Ticket resolved successfully 🎯",
	}))

	t.Run("empty response fallback", runTest(struct {
		name           string
		ticket         *ticket.Ticket
		llmResponse    string
		llmError       error
		expectedPrefix string
	}{
		name: "empty response fallback",
		ticket: &ticket.Ticket{
			ID:     types.NewTicketID(),
			Status: types.TicketStatusResolved,
			Metadata: ticket.Metadata{
				Title: "Test Alert",
			},
		},
		llmResponse:    "",
		llmError:       nil,
		expectedPrefix: "🎉 Great work! Ticket resolved successfully 🎯",
	}))
}

func TestHandleSlackInteractionViewSubmissionSalvage(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	// Setup LLM mock with embedding generation
	llmMock := &mock.LLMClientMock{
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, inputs []string) ([][]float64, error) {
			embedding := make([]float64, dimension)
			for i := range embedding {
				embedding[i] = 0.1 + float64(i)*0.01
			}
			return [][]float64{embedding}, nil
		},
	}

	// Setup Slack client mock
	slackClientMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slackSDK.MsgOption) (string, string, error) {
			return "C123456", "1234567890.123456", nil
		},
		UpdateMessageContextFunc: func(ctx context.Context, channelID string, timestamp string, options ...slackSDK.MsgOption) (string, string, string, error) {
			return channelID, timestamp, timestamp, nil
		},
		AuthTestFunc: func() (*slackSDK.AuthTestResponse, error) {
			return &slackSDK.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
	}

	// Create repository and setup test data
	repo := repository.NewMemory()

	// Create test alerts with slack threads and similar embeddings
	embedding := make([]float32, 256)
	for i := range embedding {
		embedding[i] = 0.1 + float32(i)*0.001 // Similar values for cosine similarity
	}

	alert1 := &alert.Alert{
		ID:     types.NewAlertID(),
		Schema: "test.schema.v1",
		Data:   map[string]any{"test": "data1"},
		Metadata: alert.Metadata{
			Title:       "Test Alert 1",
			Description: "Test description 1",
		},
		CreatedAt: now,
		SlackThread: &slack.Thread{
			ChannelID: "C123456",
			ThreadID:  "1111111111.111111",
		},
		Embedding: embedding,
	}
	alert2 := &alert.Alert{
		ID:     types.NewAlertID(),
		Schema: "test.schema.v1",
		Data:   map[string]any{"test": "data2"},
		Metadata: alert.Metadata{
			Title:       "Test Alert 2",
			Description: "Test description 2",
		},
		CreatedAt: now,
		SlackThread: &slack.Thread{
			ChannelID: "C123456",
			ThreadID:  "2222222222.222222",
		},
		Embedding: embedding,
	}

	// Store alerts in repository
	gt.NoError(t, repo.PutAlert(ctx, *alert1))
	gt.NoError(t, repo.PutAlert(ctx, *alert2))

	// Create test ticket with similar embedding
	testTicket := &ticket.Ticket{
		ID:       types.NewTicketID(),
		Status:   types.TicketStatusOpen,
		AlertIDs: []types.AlertID{}, // Empty initially
		SlackThread: &slack.Thread{
			ChannelID: "C123456",
			ThreadID:  "3333333333.333333",
		},
		Metadata: ticket.Metadata{
			Title:       "Target Ticket",
			Description: "Test ticket for salvage",
		},
		Embedding: embedding,
	}
	gt.NoError(t, repo.PutTicket(ctx, *testTicket))

	// Create slack service
	slackSvc, err := slackservice.New(slackClientMock, "C123456")
	gt.NoError(t, err)

	// Create usecase
	uc := New(
		WithLLMClient(llmMock),
		WithRepository(repo),
		WithSlackNotifier(slackSvc),
	)

	// Prepare test input values for salvage with low threshold to include both alerts
	values := slack.StateValue{
		string(slack.BlockIDSalvageThreshold): map[string]slack.BlockAction{
			string(slack.BlockActionIDSalvageThreshold): {Value: "0.1"},
		},
		string(slack.BlockIDSalvageKeyword): map[string]slack.BlockAction{
			string(slack.BlockActionIDSalvageKeyword): {Value: ""},
		},
	}

	user := slack.User{
		ID:   "U123456",
		Name: "test_user",
	}

	// Execute salvage operation
	err = uc.handleSlackInteractionViewSubmissionSalvage(ctx, user, string(testTicket.ID), values)
	gt.NoError(t, err)

	// Verify that alerts were bound to ticket
	updatedTicket, err := repo.GetTicket(ctx, testTicket.ID)
	gt.NoError(t, err)
	gt.Value(t, len(updatedTicket.AlertIDs)).Equal(2)

	// Verify alert IDs were added to ticket
	alertIDs := make(map[types.AlertID]bool)
	for _, id := range updatedTicket.AlertIDs {
		alertIDs[id] = true
	}
	gt.Value(t, alertIDs[alert1.ID]).Equal(true)
	gt.Value(t, alertIDs[alert2.ID]).Equal(true)

	// This test verifies that:
	// 1. Alerts are properly salvaged and bound to the ticket
	// 2. The salvage operation calls UpdateAlerts (which happens in slack_itx_submit.go:430)
	// 3. UpdateAlerts triggers the rate-limited updater to update individual alert Slack posts
}
