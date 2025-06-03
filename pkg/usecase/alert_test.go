package usecase_test

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gollem"
	gollem_mock "github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/clock"

	slack_sdk "github.com/slack-go/slack"
)

func TestHandleAlert_NoSimilarAlert(t *testing.T) {
	// Test case: No similar alert found, should post to new thread
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	var postAlertCalled bool
	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			postAlertCalled = true
			return "test-channel", "test-thread", nil
		},
		UploadFileV2ContextFunc: func(ctx context.Context, params slack_sdk.UploadFileV2Parameters) (*slack_sdk.FileSummary, error) {
			return &slack_sdk.FileSummary{}, nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
	}

	llmMock := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &gollem_mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{
							`{"title": "test title", "description": "test description"}`,
						},
					}, nil
				},
			}, nil
		},
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, texts []string) ([][]float64, error) {
			return [][]float64{{0.1, 0.2, 0.3}}, nil
		},
	}

	policyMock := &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, data any, result any, queryOptions ...opaq.QueryOption) error {
			if queryResult, ok := result.(*alert.QueryOutput); ok {
				queryResult.Alert = []alert.Metadata{
					{
						Title:       "Test Alert",
						Description: "Test Description",
					},
				}
			}
			return nil
		},
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
		usecase.WithPolicyClient(policyMock),
	)

	// Execute
	result, err := uc.HandleAlert(ctx, types.AlertSchema("test"), map[string]interface{}{"key": "value"})

	// Verify
	gt.NoError(t, err)
	gt.Array(t, result).Length(1).Required()
	gt.Value(t, postAlertCalled).Equal(true)
	gt.NotNil(t, result[0].SlackThread)
}

func TestHandleAlert_SimilarAlertFound(t *testing.T) {
	// Test case: Similar alert found, should post to existing thread
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	// Create existing alert (within 24 hours, not bound to ticket)
	existingAlert := alert.Alert{
		ID:        types.NewAlertID(),
		TicketID:  types.EmptyTicketID,
		Schema:    "test",
		CreatedAt: now.Add(-1 * time.Hour),
		Metadata: alert.Metadata{
			Title:       "Existing Alert",
			Description: "Existing Description",
		},
		SlackThread: &slack.Thread{
			ChannelID: "existing-channel",
			ThreadID:  "existing-thread",
		},
		Embedding: firestore.Vector32{0.1, 0.2, 0.3}, // Same embedding for high similarity
		Data:      map[string]interface{}{"key": "existing"},
	}

	gt.NoError(t, repo.PutAlert(ctx, existingAlert))

	var postAlertCalled bool
	var usedChannelID string

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			// Check if this is posting to existing channel
			if channelID == "existing-channel" {
				usedChannelID = channelID
			} else {
				postAlertCalled = true
			}

			return channelID, "test-thread", nil
		},
		UploadFileV2ContextFunc: func(ctx context.Context, params slack_sdk.UploadFileV2Parameters) (*slack_sdk.FileSummary, error) {
			if usedChannelID == "" {
				usedChannelID = params.Channel
			}
			return &slack_sdk.FileSummary{}, nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
	}

	llmMock := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &gollem_mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{
							`{"title": "test title", "description": "test description"}`,
						},
					}, nil
				},
			}, nil
		},
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, texts []string) ([][]float64, error) {
			return [][]float64{{0.1, 0.2, 0.3}}, nil
		},
	}

	policyMock := &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, data any, result any, queryOptions ...opaq.QueryOption) error {
			if queryResult, ok := result.(*alert.QueryOutput); ok {
				queryResult.Alert = []alert.Metadata{
					{
						Title:       "Test Alert",
						Description: "Test Description",
					},
				}
			}
			return nil
		},
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
		usecase.WithPolicyClient(policyMock),
	)

	// Execute
	result, err := uc.HandleAlert(ctx, types.AlertSchema("test"), map[string]interface{}{"key": "new"})

	// Verify
	gt.NoError(t, err)
	gt.Array(t, result).Length(1)

	// Should post to existing thread, not create new thread
	gt.Value(t, postAlertCalled).Equal(false)
	gt.Value(t, usedChannelID).Equal("existing-channel")
	gt.Value(t, result[0].SlackThread.ChannelID).Equal("existing-channel")
	gt.Value(t, result[0].SlackThread.ThreadID).Equal("existing-thread")
}

func TestHandleAlert_SimilarAlertBoundToTicket(t *testing.T) {
	// Test case: Similar alert found but bound to ticket, should post to new thread
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	// Create existing alert bound to ticket
	existingAlert := alert.Alert{
		ID:        types.NewAlertID(),
		TicketID:  types.NewTicketID(), // Bound to ticket
		Schema:    "test",
		CreatedAt: now.Add(-1 * time.Hour),
		Metadata: alert.Metadata{
			Title:       "Existing Alert",
			Description: "Existing Description",
		},
		SlackThread: &slack.Thread{
			ChannelID: "existing-channel",
			ThreadID:  "existing-thread",
		},
		Embedding: firestore.Vector32{0.1, 0.2, 0.3},
		Data:      map[string]interface{}{"key": "existing"},
	}

	gt.NoError(t, repo.PutAlert(ctx, existingAlert))

	var postAlertCalled bool

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			postAlertCalled = true
			return "test-channel", "test-thread", nil
		},
		UploadFileV2ContextFunc: func(ctx context.Context, params slack_sdk.UploadFileV2Parameters) (*slack_sdk.FileSummary, error) {
			return &slack_sdk.FileSummary{}, nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
	}

	llmMock := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &gollem_mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{
							`{"title": "test title", "description": "test description"}`,
						},
					}, nil
				},
			}, nil
		},
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, texts []string) ([][]float64, error) {
			return [][]float64{{0.1, 0.2, 0.3}}, nil
		},
	}

	policyMock := &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, data any, result any, queryOptions ...opaq.QueryOption) error {
			if queryResult, ok := result.(*alert.QueryOutput); ok {
				queryResult.Alert = []alert.Metadata{
					{
						Title:       "Test Alert",
						Description: "Test Description",
					},
				}
			}
			return nil
		},
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
		usecase.WithPolicyClient(policyMock),
	)

	// Execute
	result, err := uc.HandleAlert(ctx, types.AlertSchema("test"), map[string]interface{}{"key": "new"})

	// Verify - should post to new thread because existing alert is bound to ticket
	gt.NoError(t, err)
	gt.Array(t, result).Length(1)
	gt.Value(t, postAlertCalled).Equal(true)
	gt.Value(t, result[0].SlackThread.ChannelID).Equal("test-channel")
}

func TestHandleAlert_LowSimilarity(t *testing.T) {
	// Test case: Similar alert found but similarity < 0.99, should post to new thread
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	// Create existing alert with different embedding
	existingAlert := alert.Alert{
		ID:        types.NewAlertID(),
		TicketID:  types.EmptyTicketID,
		Schema:    "test",
		CreatedAt: now.Add(-1 * time.Hour),
		Metadata: alert.Metadata{
			Title:       "Existing Alert",
			Description: "Existing Description",
		},
		SlackThread: &slack.Thread{
			ChannelID: "existing-channel",
			ThreadID:  "existing-thread",
		},
		Embedding: firestore.Vector32{0.9, 0.1, 0.1}, // Different embedding for low similarity
		Data:      map[string]interface{}{"key": "existing"},
	}

	gt.NoError(t, repo.PutAlert(ctx, existingAlert))

	var postAlertCalled bool

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			postAlertCalled = true
			return "test-channel", "test-thread", nil
		},
		UploadFileV2ContextFunc: func(ctx context.Context, params slack_sdk.UploadFileV2Parameters) (*slack_sdk.FileSummary, error) {
			return &slack_sdk.FileSummary{}, nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
	}

	llmMock := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &gollem_mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{
							`{"title": "test title", "description": "test description"}`,
						},
					}, nil
				},
			}, nil
		},
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, texts []string) ([][]float64, error) {
			return [][]float64{{0.1, 0.2, 0.3}}, nil
		},
	}

	policyMock := &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, data any, result any, queryOptions ...opaq.QueryOption) error {
			if queryResult, ok := result.(*alert.QueryOutput); ok {
				queryResult.Alert = []alert.Metadata{
					{
						Title:       "Test Alert",
						Description: "Test Description",
					},
				}
			}
			return nil
		},
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
		usecase.WithPolicyClient(policyMock),
	)

	// Execute
	result, err := uc.HandleAlert(ctx, types.AlertSchema("test"), map[string]interface{}{"key": "new"})

	// Verify - should post to new thread because similarity is too low
	gt.NoError(t, err)
	gt.Array(t, result).Length(1)
	gt.Value(t, postAlertCalled).Equal(true)
	gt.Value(t, result[0].SlackThread.ChannelID).Equal("test-channel")
}
