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
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
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
		usecase.WithSlackNotifier(slackSvc),
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
		usecase.WithSlackNotifier(slackSvc),
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
		usecase.WithSlackNotifier(slackSvc),
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
		usecase.WithSlackNotifier(slackSvc),
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
		usecase.WithSlackNotifier(slackSvc),
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
		usecase.WithSlackNotifier(slackSvc),
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
		usecase.WithSlackNotifier(usecase.NewDiscardSlackNotifier()),
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
				usecase.WithSlackNotifier(slackSvc),
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
