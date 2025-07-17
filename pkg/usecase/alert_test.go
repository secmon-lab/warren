package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gollem"
	gollem_mock "github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
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

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
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

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
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

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
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

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
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
