package usecase_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gollem"
	gollem_mock "github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/notice"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/prompt"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/clock"

	slack_sdk "github.com/slack-go/slack"

	tagmodel "github.com/secmon-lab/warren/pkg/domain/model/tag"
	tagservice "github.com/secmon-lab/warren/pkg/service/tag"
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

func TestGetUnboundAlertsFiltered_SimilarityThreshold(t *testing.T) {
	// Test similarity-based filtering for salvage functionality
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	// Create a ticket with embedding
	testTicket := ticket.New(ctx, []types.AlertID{}, nil)
	testTicket.Metadata.Title = "Test Ticket"
	testTicket.Metadata.Description = "Test Description"
	testTicket.Embedding = firestore.Vector32{0.5, 0.5, 0.5}
	gt.NoError(t, repo.PutTicket(ctx, testTicket))

	// Create unbound alerts with different similarity levels
	alerts := []*alert.Alert{
		// High similarity (~1.0)
		{
			ID:        types.NewAlertID(),
			TicketID:  types.EmptyTicketID,
			Schema:    "test",
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: "High Similarity Alert"},
			Embedding: firestore.Vector32{0.6, 0.6, 0.6}, // ~1.0 similarity
			Data:      map[string]interface{}{"key": "high"},
		},
		// Medium similarity (~0.92)
		{
			ID:        types.NewAlertID(),
			TicketID:  types.EmptyTicketID,
			Schema:    "test",
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: "Medium Similarity Alert"},
			Embedding: firestore.Vector32{0.7, 0.3, 0.3}, // ~0.92 similarity
			Data:      map[string]interface{}{"key": "medium"},
		},
		// Low similarity (~0.58)
		{
			ID:        types.NewAlertID(),
			TicketID:  types.EmptyTicketID,
			Schema:    "test",
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: "Low Similarity Alert"},
			Embedding: firestore.Vector32{1.0, 0.0, 0.0}, // ~0.58 similarity
			Data:      map[string]interface{}{"key": "low"},
		},
		// Very low similarity (should be below most thresholds)
		{
			ID:        types.NewAlertID(),
			TicketID:  types.EmptyTicketID,
			Schema:    "test",
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: "Very Low Similarity Alert"},
			Embedding: firestore.Vector32{-1.0, 0.0, 0.0}, // Very low similarity
			Data:      map[string]interface{}{"key": "verylow"},
		},
		// Alert bound to ticket (should be excluded)
		{
			ID:        types.NewAlertID(),
			TicketID:  types.NewTicketID(), // Bound to ticket
			Schema:    "test",
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: "Bound Alert"},
			Embedding: firestore.Vector32{0.6, 0.6, 0.6}, // High similarity but bound
			Data:      map[string]interface{}{"key": "bound"},
		},
	}

	for _, a := range alerts {
		gt.NoError(t, repo.PutAlert(ctx, *a))
	}

	uc := usecase.New(usecase.WithRepository(repo))

	// Test with high threshold (0.95) - should return only high similarity alert
	threshold := 0.95
	result, total, err := uc.GetUnboundAlertsFiltered(ctx, &threshold, nil, &testTicket.ID, 0, 10)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(1)
	gt.Array(t, result).Length(1)
	gt.Value(t, result[0].Metadata.Title).Equal("High Similarity Alert")

	// Test with medium threshold (0.8) - should return high and medium similarity alerts
	threshold = 0.8
	result, total, err = uc.GetUnboundAlertsFiltered(ctx, &threshold, nil, &testTicket.ID, 0, 10)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(2) // High + Medium
	gt.Array(t, result).Length(2)

	// Test with low threshold (0.5) - should return high, medium, and low similarity alerts
	threshold = 0.5
	result, total, err = uc.GetUnboundAlertsFiltered(ctx, &threshold, nil, &testTicket.ID, 0, 10)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(3) // High + Medium + Low
	gt.Array(t, result).Length(3)

	// Test with zero threshold - should return alerts with similarity >= 0
	threshold = 0.0
	result, total, err = uc.GetUnboundAlertsFiltered(ctx, &threshold, nil, &testTicket.ID, 0, 10)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(3) // High + Medium + Low (Very Low has negative similarity)
	gt.Array(t, result).Length(3)

	// Test with negative threshold - should return all unbound alerts
	threshold = -1.0
	result, total, err = uc.GetUnboundAlertsFiltered(ctx, &threshold, nil, &testTicket.ID, 0, 10)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(4) // All unbound alerts (excluding bound one)
	gt.Array(t, result).Length(4)
}

func TestGetUnboundAlertsFiltered_KeywordFiltering(t *testing.T) {
	// Test keyword-based filtering
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	// Create unbound alerts with different content
	alerts := []*alert.Alert{
		{
			ID:        types.NewAlertID(),
			TicketID:  types.EmptyTicketID,
			Schema:    "test",
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: "Database Connection Error", Description: "MySQL connection failed"},
			Data:      map[string]interface{}{"service": "database", "error": "connection timeout"},
		},
		{
			ID:        types.NewAlertID(),
			TicketID:  types.EmptyTicketID,
			Schema:    "test",
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: "Web Server Alert", Description: "HTTP 500 error"},
			Data:      map[string]interface{}{"service": "web", "status": "error"},
		},
		{
			ID:        types.NewAlertID(),
			TicketID:  types.EmptyTicketID,
			Schema:    "test",
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: "Network Issue", Description: "Packet loss detected"},
			Data:      map[string]interface{}{"service": "network", "type": "connectivity"},
		},
	}

	for _, a := range alerts {
		gt.NoError(t, repo.PutAlert(ctx, *a))
	}

	uc := usecase.New(usecase.WithRepository(repo))

	// Test keyword search for "database"
	keyword := "database"
	result, total, err := uc.GetUnboundAlertsFiltered(ctx, nil, &keyword, nil, 0, 10)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(1)
	gt.Array(t, result).Length(1)
	gt.Value(t, result[0].Metadata.Title).Equal("Database Connection Error")

	// Test keyword search for "error" (should match multiple alerts)
	keyword = "error"
	result, total, err = uc.GetUnboundAlertsFiltered(ctx, nil, &keyword, nil, 0, 10)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(2) // Database and Web Server alerts
	gt.Array(t, result).Length(2)

	// Test keyword search with no matches
	keyword = "nonexistent"
	result, total, err = uc.GetUnboundAlertsFiltered(ctx, nil, &keyword, nil, 0, 10)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(0)
	gt.Array(t, result).Length(0)
}

func TestGetUnboundAlertsFiltered_Combined(t *testing.T) {
	// Test combined similarity and keyword filtering
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	// Create a ticket
	testTicket := ticket.New(ctx, []types.AlertID{}, nil)
	testTicket.Metadata.Title = "Database Issues"
	testTicket.Metadata.Description = "Database connectivity problems"
	testTicket.Embedding = firestore.Vector32{0.5, 0.5, 0.5}
	gt.NoError(t, repo.PutTicket(ctx, testTicket))

	// Create unbound alerts
	alerts := []*alert.Alert{
		// High similarity + contains keyword
		{
			ID:        types.NewAlertID(),
			TicketID:  types.EmptyTicketID,
			Schema:    "test",
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: "Database Connection Error", Description: "MySQL connection failed"},
			Embedding: firestore.Vector32{0.6, 0.6, 0.6}, // High similarity
			Data:      map[string]interface{}{"service": "database", "error": "timeout"},
		},
		// High similarity + does not contain keyword
		{
			ID:        types.NewAlertID(),
			TicketID:  types.EmptyTicketID,
			Schema:    "test",
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: "Web Server Alert", Description: "HTTP 500 error"},
			Embedding: firestore.Vector32{0.6, 0.6, 0.6}, // High similarity
			Data:      map[string]interface{}{"service": "web", "status": "error"},
		},
		// Low similarity + contains keyword
		{
			ID:        types.NewAlertID(),
			TicketID:  types.EmptyTicketID,
			Schema:    "test",
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: "Database Maintenance", Description: "Scheduled database restart"},
			Embedding: firestore.Vector32{1.0, 0.0, 0.0}, // Low similarity
			Data:      map[string]interface{}{"service": "database", "type": "maintenance"},
		},
	}

	for _, a := range alerts {
		gt.NoError(t, repo.PutAlert(ctx, *a))
	}

	uc := usecase.New(usecase.WithRepository(repo))

	// Test with both similarity threshold and keyword
	threshold := 0.8
	keyword := "database"
	result, total, err := uc.GetUnboundAlertsFiltered(ctx, &threshold, &keyword, &testTicket.ID, 0, 10)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(1) // Only high similarity + keyword match
	gt.Array(t, result).Length(1)
	gt.Value(t, result[0].Metadata.Title).Equal("Database Connection Error")

	// Test with lower threshold and same keyword
	threshold = 0.0
	result, total, err = uc.GetUnboundAlertsFiltered(ctx, &threshold, &keyword, &testTicket.ID, 0, 10)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(2) // Both database alerts (regardless of similarity)
	gt.Array(t, result).Length(2)
}

func TestGetUnboundAlertsFiltered_Pagination(t *testing.T) {
	// Test pagination functionality
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	// Create multiple unbound alerts
	for i := 0; i < 15; i++ {
		alert := &alert.Alert{
			ID:        types.NewAlertID(),
			TicketID:  types.EmptyTicketID,
			Schema:    "test",
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: fmt.Sprintf("Alert %d", i)},
			Data:      map[string]interface{}{"index": i},
		}
		gt.NoError(t, repo.PutAlert(ctx, *alert))
	}

	uc := usecase.New(usecase.WithRepository(repo))

	// Test first page
	result, total, err := uc.GetUnboundAlertsFiltered(ctx, nil, nil, nil, 0, 5)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(15)
	gt.Array(t, result).Length(5)

	// Test second page
	result, total, err = uc.GetUnboundAlertsFiltered(ctx, nil, nil, nil, 5, 5)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(15)
	gt.Array(t, result).Length(5)

	// Test last page (partial)
	result, total, err = uc.GetUnboundAlertsFiltered(ctx, nil, nil, nil, 10, 5)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(15)
	gt.Array(t, result).Length(5)

	// Test beyond available data
	result, total, err = uc.GetUnboundAlertsFiltered(ctx, nil, nil, nil, 20, 5)
	gt.NoError(t, err)
	gt.Value(t, total).Equal(15)
	gt.Array(t, result).Length(0)
}

func TestBindAlertsToTicket_MetadataAndSlackUpdate(t *testing.T) {
	// Test that BindAlertsToTicket updates metadata and Slack display
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	// Create a ticket with existing metadata
	testTicket := ticket.New(ctx, []types.AlertID{}, nil)
	testTicket.Metadata.Title = "Original Ticket Title"
	testTicket.Metadata.Description = "Original ticket description"
	testTicket.Metadata.TitleSource = types.SourceAI
	testTicket.Metadata.DescriptionSource = types.SourceAI
	testTicket.SlackThread = &slack.Thread{
		ChannelID: "ticket-channel",
		ThreadID:  "ticket-thread",
	}
	gt.NoError(t, repo.PutTicket(ctx, testTicket))

	// Create unbound alerts to bind to the ticket
	alert1 := alert.Alert{
		ID:        types.NewAlertID(),
		TicketID:  types.EmptyTicketID,
		Schema:    "test",
		CreatedAt: now,
		Metadata: alert.Metadata{
			Title:       "New Alert 1",
			Description: "First new alert description",
		},
		SlackThread: &slack.Thread{
			ChannelID: "alert1-channel",
			ThreadID:  "1234567890.123456", // Valid Slack message timestamp format
		},
		Data: map[string]interface{}{"key": "value1"},
	}

	alert2 := alert.Alert{
		ID:        types.NewAlertID(),
		TicketID:  types.EmptyTicketID,
		Schema:    "test",
		CreatedAt: now,
		Metadata: alert.Metadata{
			Title:       "New Alert 2",
			Description: "Second new alert description",
		},
		SlackThread: &slack.Thread{
			ChannelID: "alert2-channel",
			ThreadID:  "1234567890.123457", // Valid Slack message timestamp format
		},
		Data: map[string]interface{}{"key": "value2"},
	}

	gt.NoError(t, repo.PutAlert(ctx, alert1))
	gt.NoError(t, repo.PutAlert(ctx, alert2))

	// Track LLM calls for metadata filling
	var fillMetadataCalled bool
	var generatedTitle string
	var generatedDescription string

	llmMock := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &gollem_mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					fillMetadataCalled = true
					// Simulate LLM updating metadata based on existing + new alert info
					generatedTitle = "Updated Ticket Title with New Alerts"
					generatedDescription = "Updated description incorporating new alert information"
					return &gollem.Response{
						Texts: []string{
							fmt.Sprintf(`{"title": "%s", "description": "%s"}`, generatedTitle, generatedDescription),
						},
					}, nil
				},
			}, nil
		},
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, texts []string) ([][]float64, error) {
			return [][]float64{{0.1, 0.2, 0.3}}, nil
		},
	}

	// Track Slack update calls
	var slackTicketUpdateCalled bool
	var slackAlertUpdateCalled int

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			if channelID == "ticket-channel" {
				slackTicketUpdateCalled = true
			}
			return channelID, "updated-thread", nil
		},
		UpdateMessageContextFunc: func(ctx context.Context, channelID, timestamp string, options ...slack_sdk.MsgOption) (string, string, string, error) {
			slackAlertUpdateCalled++
			return channelID, timestamp, "updated", nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
		GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
			return &slack_sdk.TeamInfo{
				ID:   "team-id",
				Name: "test-team",
			}, nil
		},
		GetUserInfoFunc: func(userID string) (*slack_sdk.User, error) {
			return &slack_sdk.User{
				ID:   userID,
				Name: "test-user",
			}, nil
		},
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel",
		slack_svc.WithUpdaterOptions(slack_svc.WithInterval(10*time.Millisecond)))
	gt.NoError(t, err)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
	)

	// Execute: Bind alerts to ticket
	err = uc.BindAlertsToTicket(ctx, testTicket.ID, []types.AlertID{alert1.ID, alert2.ID})
	gt.NoError(t, err)

	// Wait for async alert update processing (rate-limited updater with 10ms interval)
	time.Sleep(100 * time.Millisecond)

	// Verify: Check that alerts are bound
	boundAlert1, err := repo.GetAlert(ctx, alert1.ID)
	gt.NoError(t, err)
	gt.Value(t, boundAlert1.TicketID).Equal(testTicket.ID)

	boundAlert2, err := repo.GetAlert(ctx, alert2.ID)
	gt.NoError(t, err)
	gt.Value(t, boundAlert2.TicketID).Equal(testTicket.ID)

	// Verify: Check that ticket has the alerts
	updatedTicket, err := repo.GetTicket(ctx, testTicket.ID)
	gt.NoError(t, err)
	gt.Array(t, updatedTicket.AlertIDs).Length(2)

	// Check that both alerts are in the ticket
	gt.Value(t, slices.Contains(updatedTicket.AlertIDs, alert1.ID)).Equal(true)
	gt.Value(t, slices.Contains(updatedTicket.AlertIDs, alert2.ID)).Equal(true)

	// Verify: Check that FillMetadata was called (metadata update)
	gt.Value(t, fillMetadataCalled).Equal(true)
	gt.Value(t, updatedTicket.Metadata.Title).Equal(generatedTitle)
	gt.Value(t, updatedTicket.Metadata.Description).Equal(generatedDescription)

	// Verify: Check that Slack was updated for the ticket and individual alerts
	gt.Value(t, slackTicketUpdateCalled).Equal(true)

	// Check that individual alert messages were updated
	updateCalls := slackMock.UpdateMessageContextCalls()
	gt.Value(t, len(updateCalls)).Equal(2)

	// Verify: Check that embedding was recalculated (not nil/empty)
	gt.Value(t, len(updatedTicket.Embedding) > 0).Equal(true)
}

func TestBindAlertsToTicket_MetadataUpdateBasic(t *testing.T) {
	// Simple test that metadata update happens when binding alerts
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	// Create a ticket with AI metadata source
	testTicket := ticket.New(ctx, []types.AlertID{}, nil)
	testTicket.Metadata.Title = "Original Title"
	testTicket.Metadata.TitleSource = types.SourceAI
	testTicket.Metadata.DescriptionSource = types.SourceAI
	gt.NoError(t, repo.PutTicket(ctx, testTicket))

	// Create an alert to bind
	testAlert := alert.Alert{
		ID:        types.NewAlertID(),
		TicketID:  types.EmptyTicketID,
		Schema:    "test",
		CreatedAt: now,
		Metadata: alert.Metadata{
			Title:       "New Alert",
			Description: "New alert description",
		},
		Data: map[string]interface{}{"key": "value"},
	}
	gt.NoError(t, repo.PutAlert(ctx, testAlert))

	// Track if FillMetadata was called
	var fillMetadataCalled bool

	llmMock := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &gollem_mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					fillMetadataCalled = true
					return &gollem.Response{
						Texts: []string{`{"title": "Updated with New Alert Info", "description": "Updated description"}`},
					}, nil
				},
			}, nil
		},
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, texts []string) ([][]float64, error) {
			return [][]float64{{0.1, 0.2, 0.3}}, nil
		},
	}

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithLLMClient(llmMock),
	)

	// Execute: Bind alert to ticket
	err := uc.BindAlertsToTicket(ctx, testTicket.ID, []types.AlertID{testAlert.ID})
	gt.NoError(t, err)

	// Verify: FillMetadata was called
	gt.Value(t, fillMetadataCalled).Equal(true)

	// Verify: Ticket metadata was updated
	updatedTicket, err := repo.GetTicket(ctx, testTicket.ID)
	gt.NoError(t, err)
	gt.Value(t, updatedTicket.Metadata.Title).Equal("Updated with New Alert Info")
	gt.Value(t, updatedTicket.Metadata.Description).Equal("Updated description")
}

func TestHandleAlert_DefaultPolicyMode(t *testing.T) {
	// Test case: No policy package exists, default mode (strictAlert=false)
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
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

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	llmMock := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &gollem_mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{
							`{"title": "Test Alert", "description": "Test Description"}`,
						},
					}, nil
				},
			}, nil
		},
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, texts []string) ([][]float64, error) {
			return [][]float64{{0.1, 0.2, 0.3}}, nil
		},
	}

	// PolicyClient that returns ErrNoEvalResult (package not found)
	policyMock := &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, data any, result any, queryOptions ...opaq.QueryOption) error {
			// Return ErrNoEvalResult to simulate missing package
			return opaq.ErrNoEvalResult
		},
	}

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
		usecase.WithPolicyClient(policyMock),
		usecase.WithStrictAlert(false), // Default mode
	)

	// Call HandleAlert with a schema that doesn't have a policy
	alerts, err := uc.HandleAlert(ctx, "nonexistent-schema", map[string]any{
		"test": "data",
	})

	// Should not error in default mode
	gt.NoError(t, err)
	gt.V(t, len(alerts)).Equal(1)

	// Check that the alert was created with default handling
	createdAlert := alerts[0]
	gt.V(t, createdAlert.Schema).Equal(types.AlertSchema("nonexistent-schema"))
	gt.V(t, createdAlert.Metadata.Title).Equal("Test Alert")
	gt.V(t, createdAlert.Metadata.Description).Equal("Test Description")
	gt.V(t, createdAlert.Data).Equal(map[string]any{"test": "data"})
}

func TestHandleAlert_StrictMode(t *testing.T) {
	// Test case: No policy package exists, strict mode (strictAlert=true)
	ctx := context.Background()

	// PolicyClient that returns ErrNoEvalResult (package not found)
	policyMock := &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, data any, result any, queryOptions ...opaq.QueryOption) error {
			// Return ErrNoEvalResult to simulate missing package
			return opaq.ErrNoEvalResult
		},
	}

	uc := usecase.New(
		usecase.WithRepository(repository.NewMemory()),
		usecase.WithLLMClient(&gollem_mock.LLMClientMock{}),
		usecase.WithPolicyClient(policyMock),
		usecase.WithStrictAlert(true), // Strict mode
	)

	// Call HandleAlert with a schema that doesn't have a policy
	alerts, err := uc.HandleAlert(ctx, "nonexistent-schema", map[string]any{
		"test": "data",
	})

	// Should error in strict mode
	gt.Error(t, err)
	gt.V(t, alerts).Nil()
	gt.S(t, err.Error()).Contains("no policy package found")
}

func TestHandleAlert_ExistingPolicyUnchanged(t *testing.T) {
	// Test case: Policy package exists, behavior should be unchanged
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
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

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	llmMock := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &gollem_mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{
							`{"title": "Policy Alert", "description": "Policy Description"}`,
						},
					}, nil
				},
			}, nil
		},
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, texts []string) ([][]float64, error) {
			return [][]float64{{0.1, 0.2, 0.3}}, nil
		},
	}

	// PolicyClient that returns success (package exists)
	policyMock := &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, data any, result any, queryOptions ...opaq.QueryOption) error {
			if queryResult, ok := result.(*alert.QueryOutput); ok {
				queryResult.Alert = []alert.Metadata{
					{
						Title:       "Policy Title",
						Description: "Policy Description",
					},
				}
			}
			return nil
		},
	}

	// Test both modes - behavior should be the same when policy exists
	for _, strictMode := range []bool{true, false} {
		name := "strictAlert=false"
		if strictMode {
			name = "strictAlert=true"
		}
		t.Run(name, func(t *testing.T) {
			uc := usecase.New(
				usecase.WithRepository(repo),
				usecase.WithSlackService(slackSvc),
				usecase.WithLLMClient(llmMock),
				usecase.WithPolicyClient(policyMock),
				usecase.WithStrictAlert(strictMode),
			)

			alerts, err := uc.HandleAlert(ctx, "existing-schema", map[string]any{
				"test": "data",
			})

			// Should succeed regardless of strict mode when policy exists
			gt.NoError(t, err)
			gt.V(t, len(alerts)).Equal(1)

			// Check that the alert was created from policy
			createdAlert := alerts[0]
			gt.V(t, createdAlert.Schema).Equal(types.AlertSchema("existing-schema"))
			// Title should come from policy (not overwritten by LLM in this case)
			gt.V(t, createdAlert.Metadata.Title).Equal("Policy Title")
		})
	}
}

func TestHandleAlert_PolicyWithTags(t *testing.T) {
	// Test case: Alert policy returns tags, should create tags and assign to alert
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
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

	// Policy returns alert with tags
	policyMock := &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, data any, result any, queryOptions ...opaq.QueryOption) error {
			if queryResult, ok := result.(*alert.QueryOutput); ok {
				// Return tags as string slice (for alert.Metadata)
				queryResult.Alert = []alert.Metadata{
					{
						Title:       "Security Alert",
						Description: "Network security incident detected",
						Tags:        []string{"security", "high-priority", "network"},
					},
				}
			}
			return nil
		},
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	// Create TagService
	tagSvc := tagservice.New(repo)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
		usecase.WithPolicyClient(policyMock),
		usecase.WithTagService(tagSvc),
	)

	// Execute
	result, err := uc.HandleAlert(ctx, types.AlertSchema("test"), map[string]interface{}{"key": "value"})

	// Verify alert creation
	gt.NoError(t, err)
	gt.Array(t, result).Length(1).Required()

	createdAlert := result[0]
	gt.Value(t, createdAlert.Metadata.Title).Equal("Security Alert")
	gt.Value(t, len(createdAlert.Tags)).Equal(3)

	// Verify tags are assigned to alert
	expectedTags := []string{"security", "high-priority", "network"}

	// Get actual tag names from alert using compatibility method
	actualTagNames, err := createdAlert.GetTagNames(ctx, func(ctx context.Context, tagIDs []string) ([]*tagmodel.Tag, error) {
		return repo.GetTagsByIDs(ctx, tagIDs)
	})
	gt.NoError(t, err)
	gt.Equal(t, len(actualTagNames), 3)

	// Verify all expected tags are present
	actualTagMap := make(map[string]bool)
	for _, name := range actualTagNames {
		actualTagMap[name] = true
	}
	for _, expectedTag := range expectedTags {
		gt.Value(t, actualTagMap[expectedTag]).Equal(true)
	}

	// Verify tags were created in repository by checking each tag individually
	// Since we moved to ID-based tags, we verify by looking up each expected tag
	for _, expectedTag := range expectedTags {
		tag, err := repo.GetTagByName(ctx, expectedTag)
		gt.NoError(t, err)
		gt.NotNil(t, tag)
		gt.Value(t, tag.Name).Equal(expectedTag)
		gt.Value(t, tag.Color).NotEqual("") // Verify tag has a color
	}
}

func TestHandleAlert_PolicyWithNewAndExistingTags(t *testing.T) {
	// Test case: Policy returns mix of new and existing tags
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	// Create TagService first
	tagSvc := tagservice.New(repo)

	// Create existing tag using new system
	createdTag, err := tagSvc.CreateTagWithCustomColor(ctx, "existing-tag", "Existing tag", "#0066cc", "test")
	gt.NoError(t, err)
	gt.NotNil(t, createdTag)

	// Get the tag from repository to ensure we have correct timestamps
	existingTag, err := repo.GetTagByName(ctx, "existing-tag")
	gt.NoError(t, err)
	gt.NotNil(t, existingTag)

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
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

	// Policy returns alert with mix of new and existing tags
	policyMock := &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, data any, result any, queryOptions ...opaq.QueryOption) error {
			if queryResult, ok := result.(*alert.QueryOutput); ok {
				queryResult.Alert = []alert.Metadata{
					{
						Title:       "Mixed Tags Alert",
						Description: "Alert with existing and new tags",
						Tags:        []string{"existing-tag", "new-tag-1", "new-tag-2"},
					},
				}
			}
			return nil
		},
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	// TagService already created above

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
		usecase.WithPolicyClient(policyMock),
		usecase.WithTagService(tagSvc),
	)

	// Execute
	result, err := uc.HandleAlert(ctx, types.AlertSchema("test"), map[string]interface{}{"key": "value"})

	// Verify
	gt.NoError(t, err)
	gt.Array(t, result).Length(1)

	createdAlert := result[0]
	gt.Value(t, len(createdAlert.Tags)).Equal(3)

	// Verify all tags are assigned to alert
	expectedTags := []string{"existing-tag", "new-tag-1", "new-tag-2"}

	// Get actual tag names from alert using compatibility method
	actualTagNames, err := createdAlert.GetTagNames(ctx, func(ctx context.Context, tagIDs []string) ([]*tagmodel.Tag, error) {
		return repo.GetTagsByIDs(ctx, tagIDs)
	})
	gt.NoError(t, err)
	gt.Equal(t, len(actualTagNames), 3)

	// Verify all expected tags are present
	actualTagMap := make(map[string]bool)
	for _, name := range actualTagNames {
		actualTagMap[name] = true
	}
	for _, expectedTag := range expectedTags {
		gt.Value(t, actualTagMap[expectedTag]).Equal(true)
	}

	// Verify individual tags exist in repository
	allExpectedTags := []string{"existing-tag", "new-tag-1", "new-tag-2"}
	for _, expectedTag := range allExpectedTags {
		tag, err := repo.GetTagByName(ctx, expectedTag)
		gt.NoError(t, err)
		gt.NotNil(t, tag)
		gt.Value(t, tag.Name).Equal(expectedTag)
		gt.Value(t, tag.Color).NotEqual("") // Verify tag has a color
	}

	// Verify existing tag unchanged
	existingTagAfter, err := repo.GetTagByName(ctx, "existing-tag")
	gt.NoError(t, err)
	gt.NotNil(t, existingTagAfter)
	gt.Value(t, existingTagAfter.ID).Equal(existingTag.ID)
	gt.Value(t, existingTagAfter.CreatedAt).Equal(existingTag.CreatedAt)
}

func TestHandleAlert_PolicyTagDuplicationPrevention(t *testing.T) {
	// This test specifically verifies that policy-based tag generation doesn't create duplicate tags
	// when the same tags are specified in multiple alerts processed consecutively
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	// Generate random tag names to avoid conflicts with other tests
	randomSuffix := now.UnixNano()
	commonTag := fmt.Sprintf("security-%d", randomSuffix)
	alertSpecificTag1 := fmt.Sprintf("alert1-tag-%d", randomSuffix)
	alertSpecificTag2 := fmt.Sprintf("alert2-tag-%d", randomSuffix)

	// Create TagService
	tagSvc := tagservice.New(repo)

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
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

	// Policy mock that returns alerts with overlapping tags
	callCount := 0
	policyMock := &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, data any, result any, queryOptions ...opaq.QueryOption) error {
			if queryResult, ok := result.(*alert.QueryOutput); ok {
				callCount++
				if callCount == 1 {
					// First alert: common tag + specific tag
					queryResult.Alert = []alert.Metadata{
						{
							Title:       "First Security Alert",
							Description: "First security incident",
							Tags:        []string{commonTag, alertSpecificTag1},
						},
					}
				} else {
					// Second alert: same common tag + different specific tag
					queryResult.Alert = []alert.Metadata{
						{
							Title:       "Second Security Alert",
							Description: "Second security incident",
							Tags:        []string{commonTag, alertSpecificTag2}, // commonTag appears again
						},
					}
				}
			}
			return nil
		},
	}

	slackSvc, err := slack_svc.New(slackMock, "test-channel")
	gt.NoError(t, err)

	// Create usecase
	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
		usecase.WithPolicyClient(policyMock),
		usecase.WithTagService(tagSvc),
	)

	// Count tags before processing any alerts
	tagsBefore, err := repo.ListAllTags(ctx)
	gt.NoError(t, err)
	initialTagCount := len(tagsBefore)

	// Process first alert
	alerts1, err := uc.HandleAlert(ctx, "test", map[string]any{"type": "security", "alert": 1})
	gt.NoError(t, err)
	gt.Array(t, alerts1).Length(1)

	// Count tags after first alert
	tagsAfterFirst, err := repo.ListAllTags(ctx)
	gt.NoError(t, err)
	expectedAfterFirst := initialTagCount + 2 // commonTag + alertSpecificTag1
	gt.Array(t, tagsAfterFirst).Length(expectedAfterFirst)

	// Process second alert with overlapping tag names
	alerts2, err := uc.HandleAlert(ctx, "test", map[string]any{"type": "security", "alert": 2})
	gt.NoError(t, err)
	gt.Array(t, alerts2).Length(1)

	// CRITICAL: Verify no duplicate tags were created
	tagsAfterSecond, err := repo.ListAllTags(ctx)
	gt.NoError(t, err)
	expectedAfterSecond := initialTagCount + 3 // commonTag (reused) + alertSpecificTag1 + alertSpecificTag2
	gt.Array(t, tagsAfterSecond).Length(expectedAfterSecond)

	// Verify both alerts have the common tag assigned
	alert1 := alerts1[0]
	alert2 := alerts2[0]

	// Both alerts should have 2 tags each
	gt.Number(t, len(alert1.TagIDs)).Equal(2)
	gt.Number(t, len(alert2.TagIDs)).Equal(2)

	// Get the common tag to verify it's shared
	commonTagObj, err := repo.GetTagByName(ctx, commonTag)
	gt.NoError(t, err)
	gt.NotNil(t, commonTagObj)

	// Both alerts should reference the same common tag ID
	gt.Value(t, alert1.TagIDs[commonTagObj.ID]).Equal(true)
	gt.Value(t, alert2.TagIDs[commonTagObj.ID]).Equal(true)

	// Verify specific tags exist and are correctly assigned
	tag1Obj, err := repo.GetTagByName(ctx, alertSpecificTag1)
	gt.NoError(t, err)
	gt.NotNil(t, tag1Obj)
	gt.Value(t, alert1.TagIDs[tag1Obj.ID]).Equal(true)
	gt.Value(t, alert2.TagIDs[tag1Obj.ID]).Equal(false) // Should not be in alert2

	tag2Obj, err := repo.GetTagByName(ctx, alertSpecificTag2)
	gt.NoError(t, err)
	gt.NotNil(t, tag2Obj)
	gt.Value(t, alert2.TagIDs[tag2Obj.ID]).Equal(true)
	gt.Value(t, alert1.TagIDs[tag2Obj.ID]).Equal(false) // Should not be in alert1

	// Verify tag names in repository
	finalTags, err := repo.ListAllTags(ctx)
	gt.NoError(t, err)
	tagNames := make([]string, len(finalTags))
	for i, tag := range finalTags {
		tagNames[i] = tag.Name
	}

	// Verify all expected tags exist exactly once
	commonTagCount := 0
	tag1Count := 0
	tag2Count := 0
	for _, name := range tagNames {
		if name == commonTag {
			commonTagCount++
		} else if name == alertSpecificTag1 {
			tag1Count++
		} else if name == alertSpecificTag2 {
			tag2Count++
		}
	}

	// Each tag should appear exactly once in the repository
	gt.Number(t, commonTagCount).Equal(1)
	gt.Number(t, tag1Count).Equal(1)
	gt.Number(t, tag2Count).Equal(1)
}

func TestProcessGenAI(t *testing.T) {
	repo := repository.NewMemory()
	ctx := context.Background()

	t.Run("no GenAI config", func(t *testing.T) {
		uc := usecase.New(usecase.WithRepository(repo))

		alert := &alert.Alert{
			Metadata: alert.Metadata{
				Title: "Test Alert",
			},
		}

		response, err := uc.ProcessGenAI(ctx, alert)
		gt.NoError(t, err)
		gt.Equal(t, response, "")
	})

	t.Run("no prompt service configured", func(t *testing.T) {
		uc := usecase.New(usecase.WithRepository(repo))

		alert := &alert.Alert{
			Metadata: alert.Metadata{
				Title: "Test Alert",
				GenAI: &alert.GenAIConfig{
					Prompt: "test_prompt.tmpl",
					Type:   "text",
				},
			},
		}

		_, err := uc.ProcessGenAI(ctx, alert)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("prompt service not configured")
	})

	t.Run("with GenAI config and prompt service", func(t *testing.T) {
		// Create prompt service with test template
		promptService, err := prompt.New("testdata/prompts")
		gt.NoError(t, err)

		// Mock LLM session and response
		mockLLM := &gollem_mock.LLMClientMock{
			NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
				return &gollem_mock.SessionMock{
					GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
						return &gollem.Response{
							Texts: []string{"This is a test LLM response"},
						}, nil
					},
				}, nil
			},
		}

		uc := usecase.New(
			usecase.WithRepository(repo),
			usecase.WithLLMClient(mockLLM),
			usecase.WithPromptService(promptService),
		)

		alert := &alert.Alert{
			Metadata: alert.Metadata{
				Title: "Test Alert",
				GenAI: &alert.GenAIConfig{
					Prompt: "test_prompt.tmpl",
					Type:   "text",
				},
			},
		}

		response, err := uc.ProcessGenAI(ctx, alert)
		gt.NoError(t, err)
		gt.Equal(t, response, "This is a test LLM response")
	})

	t.Run("with mock prompt service", func(t *testing.T) {
		// Mock prompt service
		mockPromptService := &mock.PromptServiceMock{
			GeneratePromptFunc: func(ctx context.Context, templateName string, alert *alert.Alert) (string, error) {
				return "Generated prompt for " + templateName, nil
			},
		}

		// Mock LLM session and response
		mockLLM := &gollem_mock.LLMClientMock{
			NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
				return &gollem_mock.SessionMock{
					GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
						return &gollem.Response{
							Texts: []string{"Mock LLM response"},
						}, nil
					},
				}, nil
			},
		}

		uc := usecase.New(
			usecase.WithRepository(repo),
			usecase.WithLLMClient(mockLLM),
			usecase.WithPromptService(mockPromptService),
		)

		alert := &alert.Alert{
			Metadata: alert.Metadata{
				Title: "Test Alert",
				GenAI: &alert.GenAIConfig{
					Prompt: "mock_template",
					Type:   "text",
				},
			},
		}

		response, err := uc.ProcessGenAI(ctx, alert)
		gt.NoError(t, err)
		gt.Equal(t, response, "Mock LLM response")

		// Verify prompt service was called correctly
		calls := mockPromptService.GeneratePromptCalls()
		gt.Array(t, calls).Length(1)
		gt.Equal(t, calls[0].TemplateName, "mock_template")
	})
}

func TestActionEvaluator(t *testing.T) {
	ctx := context.Background()

	t.Run("default action when no policy", func(t *testing.T) {
		policyClient := &mock.PolicyClientMock{
			QueryFunc: func(ctx context.Context, query string, data any, result any, queryOptions ...opaq.QueryOption) error {
				return opaq.ErrNoEvalResult
			},
		}
		alert := &alert.Alert{
			Metadata: alert.Metadata{Title: "Test"},
		}

		result, err := usecase.EvaluateAction(ctx, policyClient, alert, "test response")
		gt.NoError(t, err)
		gt.Equal(t, result.Publish, action.PublishTypeAlert)
	})

	t.Run("policy returns notice", func(t *testing.T) {
		policyClient := &mock.PolicyClientMock{
			QueryFunc: func(ctx context.Context, query string, data any, result any, queryOptions ...opaq.QueryOption) error {
				if policyResult, ok := result.(*action.PolicyResult); ok {
					policyResult.Publish = action.PublishTypeNotice
					policyResult.Channel = []string{"test-channel"}
				}
				return nil
			},
		}
		alert := &alert.Alert{
			Metadata: alert.Metadata{Title: "Test"},
		}

		result, err := usecase.EvaluateAction(ctx, policyClient, alert, "test response")
		gt.NoError(t, err)
		gt.Equal(t, result.Publish, action.PublishTypeNotice)
		gt.Equal(t, result.Channel, []string{"test-channel"})
	})
}

func TestHandleNotice(t *testing.T) {
	ctx := context.Background()

	t.Run("creates notice and sends Slack notification", func(t *testing.T) {
		// Create fresh mocks for this test
		repoMock := &mock.RepositoryMock{
			CreateNoticeFunc: func(ctx context.Context, notice *notice.Notice) error {
				// Verify the notice structure
				gt.NotNil(t, notice)
				gt.NotEqual(t, notice.ID, types.EmptyNoticeID)
				gt.True(t, !notice.CreatedAt.IsZero())
				gt.False(t, notice.Escalated)
				return nil
			},
			UpdateNoticeFunc: func(ctx context.Context, notice *notice.Notice) error {
				return nil
			},
		}

		slackMock := &mock.SlackClientMock{
			PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
				return channelID, "test-timestamp", nil
			},
			AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
				return &slack_sdk.AuthTestResponse{
					UserID: "test-user",
				}, nil
			},
			GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
				return &slack_sdk.TeamInfo{
					Domain: "test-workspace",
				}, nil
			},
		}

		slackSvc, err := slack_svc.New(slackMock, "#test-channel")
		gt.NoError(t, err)

		uc := usecase.New(
			usecase.WithRepository(repoMock),
			usecase.WithSlackService(slackSvc),
		)
		testAlert := &alert.Alert{
			ID: types.NewAlertID(),
			Metadata: alert.Metadata{
				Title:       "Test Alert",
				Description: "Test Description",
			},
			Data:   map[string]interface{}{"test": "data"},
			Schema: "test.schema",
		}

		err = uc.HandleNotice(ctx, testAlert, []string{"test-channel"})
		gt.NoError(t, err)

		// Verify repository interaction - notice was created
		createCalls := repoMock.CreateNoticeCalls()
		gt.Array(t, createCalls).Length(1)

		createdNotice := createCalls[0].NoticeMoqParam
		gt.Equal(t, createdNotice.Alert.ID, testAlert.ID)
		gt.Equal(t, createdNotice.Alert.Metadata.Title, "Test Alert")
		gt.Equal(t, createdNotice.Alert.Metadata.Description, "Test Description")
		gt.False(t, createdNotice.Escalated)

		// Verify Slack interaction - message was posted
		postCalls := slackMock.PostMessageContextCalls()
		gt.Array(t, postCalls).Length(1)
		gt.Equal(t, postCalls[0].ChannelID, "test-channel")
	})

	t.Run("uses default channel when no channels specified", func(t *testing.T) {
		// Create fresh mocks for this test
		repoMock := &mock.RepositoryMock{
			CreateNoticeFunc: func(ctx context.Context, notice *notice.Notice) error {
				return nil
			},
			UpdateNoticeFunc: func(ctx context.Context, notice *notice.Notice) error {
				return nil
			},
		}

		slackMock := &mock.SlackClientMock{
			PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
				return channelID, "test-timestamp", nil
			},
			AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
				return &slack_sdk.AuthTestResponse{
					UserID: "test-user",
				}, nil
			},
			GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
				return &slack_sdk.TeamInfo{
					Domain: "test-workspace",
				}, nil
			},
		}

		slackSvc, err := slack_svc.New(slackMock, "#test-channel")
		gt.NoError(t, err)

		uc := usecase.New(
			usecase.WithRepository(repoMock),
			usecase.WithSlackService(slackSvc),
		)
		testAlert := &alert.Alert{
			ID: types.NewAlertID(),
			Metadata: alert.Metadata{
				Title: "Default Channel Test",
			},
		}

		err = uc.HandleNotice(ctx, testAlert, []string{})
		gt.NoError(t, err)

		// Verify Slack was called with default channel (empty string becomes default)
		postCalls := slackMock.PostMessageContextCalls()
		gt.Array(t, postCalls).Length(1)
		gt.Equal(t, postCalls[0].ChannelID, "#test-channel")
	})

	t.Run("handles multiple channels", func(t *testing.T) {
		// Create fresh mocks for this test
		repoMock := &mock.RepositoryMock{
			CreateNoticeFunc: func(ctx context.Context, notice *notice.Notice) error {
				return nil
			},
			UpdateNoticeFunc: func(ctx context.Context, notice *notice.Notice) error {
				return nil
			},
		}

		slackMock := &mock.SlackClientMock{
			PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
				return channelID, "test-timestamp", nil
			},
			AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
				return &slack_sdk.AuthTestResponse{
					UserID: "test-user",
				}, nil
			},
			GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
				return &slack_sdk.TeamInfo{
					Domain: "test-workspace",
				}, nil
			},
		}

		slackSvc, err := slack_svc.New(slackMock, "#test-channel")
		gt.NoError(t, err)

		uc := usecase.New(
			usecase.WithRepository(repoMock),
			usecase.WithSlackService(slackSvc),
		)
		testAlert := &alert.Alert{
			ID: types.NewAlertID(),
			Metadata: alert.Metadata{
				Title: "Multi Channel Test",
			},
		}

		err = uc.HandleNotice(ctx, testAlert, []string{"channel-1", "channel-2"})
		gt.NoError(t, err)

		// Verify notice was created once
		createCalls := repoMock.CreateNoticeCalls()
		gt.Array(t, createCalls).Length(1)

		// Verify Slack was called for each channel
		postCalls := slackMock.PostMessageContextCalls()
		gt.Array(t, postCalls).Length(2)
		gt.Equal(t, postCalls[0].ChannelID, "channel-1")
		gt.Equal(t, postCalls[1].ChannelID, "channel-2")
	})
}

func TestEscalateNotice(t *testing.T) {
	// Helper Driven Testing: Test the complete workflow from notice creation to escalation
	ctx := context.Background()
	repo := repository.NewMemory()

	// Setup mocks for LLM and Slack
	llmMock := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &gollem_mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{"LLM processed alert"},
					}, nil
				},
			}, nil
		},
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, texts []string) ([][]float64, error) {
			return [][]float64{{0.1, 0.2, 0.3}}, nil
		},
	}

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			return channelID, fmt.Sprintf("msg-%d", time.Now().UnixNano()), nil
		},
		UploadFileV2ContextFunc: func(ctx context.Context, params slack_sdk.UploadFileV2Parameters) (*slack_sdk.FileSummary, error) {
			return &slack_sdk.FileSummary{ID: "file-123"}, nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{UserID: "test-user"}, nil
		},
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	// Create prompt service for GenAI processing
	promptService, err := prompt.New("testdata/prompts")
	gt.NoError(t, err)

	// Create policy client that returns "notice" for testing
	policyMock := &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, data any, result any, queryOptions ...opaq.QueryOption) error {
			if policyResult, ok := result.(*action.PolicyResult); ok {
				policyResult.Publish = action.PublishTypeNotice
				policyResult.Channel = []string{"#alerts"}
			}
			return nil
		},
	}

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithLLMClient(llmMock),
		usecase.WithSlackService(slackSvc),
		usecase.WithPromptService(promptService),
		usecase.WithPolicyClient(policyMock),
	)

	t.Run("notice creation and escalation workflow", func(t *testing.T) {
		// Step 1: Create a notice (simulates what would happen when policy returns "notice")
		alertData := map[string]interface{}{
			"severity": "medium",
			"source":   "test-system",
			"message":  "Test security event",
		}

		// Use random ID to avoid test conflicts (CLAUDE.md requirement)
		noticeID := types.NoticeID(fmt.Sprintf("notice-%d", time.Now().UnixNano()))
		testNotice := &notice.Notice{
			ID: noticeID,
			Alert: alert.Alert{
				ID: types.NewAlertID(),
				Metadata: alert.Metadata{
					Title:       "Security Notice",
					Description: "This notice needs escalation",
				},
				Data:   alertData,
				Schema: "test.alert",
			},
			CreatedAt: time.Now(),
			Escalated: false,
		}

		err := repo.CreateNotice(ctx, testNotice)
		gt.NoError(t, err)

		// Step 2: Execute escalation (simulates Slack button click or mention)
		err = uc.EscalateNotice(ctx, noticeID)
		gt.NoError(t, err)

		// Step 3: Verify escalation results
		// Check that notice is marked as escalated
		escalatedNotice, err := repo.GetNotice(ctx, noticeID)
		gt.NoError(t, err)
		gt.True(t, escalatedNotice.Escalated)

		// Verify that Slack was called to post the escalated alert
		// The mock should have been called for posting the full alert
		postCalls := slackMock.PostMessageContextCalls()
		gt.True(t, len(postCalls) >= 1)

		// Verify the escalated alert contains the original alert data
		gt.S(t, escalatedNotice.Alert.Metadata.Title).Equal("Security Notice")
		gt.V(t, escalatedNotice.Alert.Data).Equal(alertData)
	})

	t.Run("escalate nonexistent notice", func(t *testing.T) {
		nonexistentID := types.NewNoticeID()
		err := uc.EscalateNotice(ctx, nonexistentID)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to get notice")
	})

	t.Run("escalate already escalated notice", func(t *testing.T) {
		// Create an already escalated notice
		noticeID := types.NewNoticeID()
		alreadyEscalated := &notice.Notice{
			ID: noticeID,
			Alert: alert.Alert{
				ID: types.NewAlertID(),
				Metadata: alert.Metadata{
					Title: "Already Escalated Notice",
				},
			},
			CreatedAt: time.Now(),
			Escalated: true, // Already escalated
		}

		err := repo.CreateNotice(ctx, alreadyEscalated)
		gt.NoError(t, err)

		// Try to escalate again - should succeed but not do duplicate work
		err = uc.EscalateNotice(ctx, noticeID)
		gt.NoError(t, err)

		// Verify it's still marked as escalated
		notice, err := repo.GetNotice(ctx, noticeID)
		gt.NoError(t, err)
		gt.True(t, notice.Escalated)
	})
}
