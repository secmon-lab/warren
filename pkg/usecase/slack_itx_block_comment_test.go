package usecase_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/usecase"

	slack_sdk "github.com/slack-go/slack"
)

func TestTicketCreation_NewThread_WithInitialComment(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	// Create multiple alert lists in the same thread to trigger new thread creation
	thread := slack.Thread{
		ChannelID: "test-channel",
		ThreadID:  "test-thread",
	}

	// Create first alert list
	alertList1 := alert.NewList(ctx, thread, nil, alert.Alerts{})
	gt.NoError(t, repo.PutAlertList(ctx, alertList1))

	// Create second alert list in the same thread
	alertList2 := alert.NewList(ctx, thread, nil, alert.Alerts{})
	gt.NoError(t, repo.PutAlertList(ctx, alertList2))

	// Create test alert
	testAlert := alert.New(ctx, types.AlertSchema("test.alert.v1"), map[string]any{
		"test": "data",
	}, alert.Metadata{
		Title:       "Test Alert",
		Description: "Test Description",
	})
	testAlert.SlackThread = &slack.Thread{
		ChannelID: thread.ChannelID,
		ThreadID:  thread.ThreadID,
	}
	gt.NoError(t, repo.PutAlert(ctx, testAlert))

	// Track slack messages
	var postMessageCount int
	var commentPosted bool

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			postMessageCount++
			// We expect at least 3 messages: ticket blocks, comment, and link
			if postMessageCount >= 2 {
				commentPosted = true
			}
			return "test-channel", "test-thread-" + string(rune(time.Now().UnixNano())), nil
		},
		UploadFileV2ContextFunc: func(ctx context.Context, params slack_sdk.UploadFileV2Parameters) (*slack_sdk.FileSummary, error) {
			return &slack_sdk.FileSummary{}, nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
		GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
			return &slack_sdk.TeamInfo{
				ID:     "test-team",
				Name:   "Test Team",
				Domain: "test",
			}, nil
		},
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	// Create LLM mock to generate comments
	llmMock := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					// Check if this is for ticket metadata or comment generation
					inputText := ""
					for _, inp := range input {
						if textInput, ok := inp.(gollem.Text); ok {
							inputText = string(textInput)
							break
						}
					}

					if strings.Contains(inputText, "Generate Initial Ticket Comment") {
						// This is for comment generation
						return &gollem.Response{
							Texts: []string{"Let's investigate this security incident together! 🔍"},
						}, nil
					} else {
						// This is for ticket metadata
						return &gollem.Response{
							Texts: []string{
								`{"title": "Test Security Incident", "description": "Generated description", "summary": "Generated summary"}`,
							},
						}, nil
					}
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

	// Create usecase instance
	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackNotifier(slackSvc),
		usecase.WithLLMClient(llmMock),
	)

	// Test data
	user := slack.User{ID: "test-user", Name: "Test User"}

	// Execute test - this should create ticket in new thread and add comment
	err = uc.HandleSlackInteractionBlockActions(
		ctx,
		user,
		thread,
		slack.ActionIDAckAlert,
		testAlert.ID.String(),
		"trigger-id",
	)

	// Verify results
	gt.NoError(t, err)

	// Verify that a comment was posted
	gt.Value(t, commentPosted).Equal(true)

	// Verify multiple messages were posted (ticket + comment + link)
	gt.Value(t, postMessageCount >= 3).Equal(true)

	// Verify ticket was created
	tickets, err := repo.GetTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusOpen}, 0, 0)
	gt.NoError(t, err)
	gt.Array(t, tickets).Length(1)
}
