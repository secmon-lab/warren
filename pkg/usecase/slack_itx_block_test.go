package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	gollem_mock "github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/newmo-oss/go-caller"
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

func checkMockCaller(t testing.TB, skip int, pkgPath string, funcName string) {
	t.Helper()
	gt.A(t, caller.New(3+skip)).Longer(0).At(0, func(t testing.TB, v caller.Frame) {
		t.Helper()
		gt.Equal(t, v.PkgPath(), pkgPath)
		gt.Equal(t, v.FuncName(), funcName)
	})
}

// make256DimEmbedding creates a 256-dimensional embedding with test values
func make256DimEmbedding() []float32 {
	embedding := make([]float32, 256)
	for i := range embedding {
		embedding[i] = 0.1 + float32(i)*0.01
	}
	return embedding
}

func TestSlackActionAckAlert(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	// Setup mocks and repositories
	repo := repository.NewMemory()

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			return "test-ts", "test-channel", nil
		},
		UpdateMessageContextFunc: func(ctx context.Context, channelID string, timestamp string, options ...slack_sdk.MsgOption) (string, string, string, error) {
			return "test-ts", "test-channel", "test-thread", nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
	}
	// For single alert tickets, no LLM calls should be made since metadata is inherited
	llmMock := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &gollem_mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					t.Fatal("LLM should not be called for single alert ticket - metadata should be inherited")
					return nil, nil
				},
			}, nil
		},
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, inputs []string) ([][]float64, error) {
			t.Fatal("LLM embedding should not be called for single alert ticket - embedding should be inherited")
			return nil, nil
		},
	}

	// Create test alert with 256-dimensional embedding
	testAlertEmbedding := make([]float32, 256)
	for i := range testAlertEmbedding {
		testAlertEmbedding[i] = 0.1 + float32(i)*0.01
	}
	testAlert := &alert.Alert{
		ID:        types.AlertID("test-alert-1"),
		CreatedAt: now,
		Metadata:  alert.Metadata{Title: "Test Alert", Description: "Test Description"},
		SlackThread: &slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  "test-thread",
		},
		Embedding: testAlertEmbedding,
	}

	// Store test alert
	if err := repo.PutAlert(ctx, *testAlert); err != nil {
		t.Fatal(err)
	}

	// Use fast interval for testing
	slackSvc, err := slack_svc.New(slackMock, "#test-channel",
		slack_svc.WithUpdaterOptions(
			slack_svc.WithInterval(1*time.Millisecond),
			slack_svc.WithRetryInterval(1*time.Millisecond)))
	gt.NoError(t, err)

	// Create usecase instance
	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
	)

	// Test data
	user := slack.User{ID: "test-user"}
	thread := slack.Thread{
		ChannelID: "test-channel",
		ThreadID:  "test-thread",
	}

	// Execute test
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

	// Verify ticket was created
	tickets, err := repo.GetTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusOpen}, 0, 0)
	gt.NoError(t, err)
	gt.Array(t, tickets).Length(1)

	ticket := tickets[0]
	gt.Value(t, ticket.Assignee.ID).Equal(user.ID)
	gt.Value(t, ticket.SlackThread.ChannelID).Equal(thread.ChannelID)
	gt.Value(t, ticket.SlackThread.ThreadID).Equal(thread.ThreadID)

	// Verify alert was updated with ticket ID
	updatedAlert, err := repo.GetAlert(ctx, testAlert.ID)
	gt.NoError(t, err)
	gt.Value(t, updatedAlert.TicketID).Equal(ticket.ID)

	// Wait for async alert updates to complete
	time.Sleep(200 * time.Millisecond)

	// Verify Slack interactions for single alert inheritance
	// Note: 2 PostMessage calls (ticket post only, no comment for single alert) + 1 UpdateMessage for alert update
	gt.Value(t, len(slackMock.PostMessageContextCalls())).Equal(2)
	gt.Value(t, len(slackMock.UpdateMessageContextCalls())).Equal(1)
}

func TestSlackActionAckList(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	// Setup mocks and repositories
	repo := repository.NewMemory()

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			return "test-ts", "test-channel", nil
		},
		UpdateMessageContextFunc: func(ctx context.Context, channelID string, timestamp string, options ...slack_sdk.MsgOption) (string, string, string, error) {
			return "test-ts", "test-channel", "test-thread", nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
	}
	llmCallCount := 0
	llmMock := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &gollem_mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					llmCallCount++
					switch llmCallCount {
					case 1:
						return &gollem.Response{
							Texts: []string{
								`{"title": "test title", "description": "test description"}`,
							},
						}, nil
					case 2:
						return &gollem.Response{
							Texts: []string{
								`{"title": "test title", "description": "test description", "summary": "test summary"}`,
							},
						}, nil
					case 3:
						return &gollem.Response{
							Texts: []string{
								`{"title": "test title", "description": "test description", "summary": "test summary"}`,
							},
						}, nil
					case 4:
						return &gollem.Response{
							Texts: []string{
								`{"title": "test title", "description": "test description", "summary": "test summary"}`,
							},
						}, nil
					default:
						return nil, goerr.New("unexpected call")
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

	// Create test alerts with 256-dimensional embeddings
	testAlert1Embedding := make([]float32, 256)
	for i := range testAlert1Embedding {
		testAlert1Embedding[i] = 0.1 + float32(i)*0.01
	}
	testAlert2Embedding := make([]float32, 256)
	for i := range testAlert2Embedding {
		testAlert2Embedding[i] = 0.2 + float32(i)*0.01
	}
	testAlerts := alert.Alerts{
		&alert.Alert{
			ID:        types.AlertID("test-alert-1"),
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: "Test Alert 1"},
			Embedding: testAlert1Embedding,
			SlackThread: &slack.Thread{
				ChannelID: "test-channel",
				ThreadID:  "test-thread",
			},
		},
		&alert.Alert{
			ID:        types.AlertID("test-alert-2"),
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: "Test Alert 2"},
			Embedding: testAlert2Embedding,
			SlackThread: &slack.Thread{
				ChannelID: "test-channel",
				ThreadID:  "test-thread",
			},
		},
	}

	// Create test alert list
	thread := slack.Thread{
		ChannelID: "test-channel",
		ThreadID:  "test-thread",
	}
	user := slack.User{ID: "test-user"}
	testList := alert.NewList(ctx, thread, &user, testAlerts)

	// Store test alerts and list
	for _, alert := range testAlerts {
		if err := repo.PutAlert(ctx, *alert); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.PutAlertList(ctx, testList); err != nil {
		t.Fatal(err)
	}

	// Use fast interval for testing
	slackSvc, err := slack_svc.New(slackMock, "#test-channel",
		slack_svc.WithUpdaterOptions(
			slack_svc.WithInterval(1*time.Millisecond),
			slack_svc.WithRetryInterval(1*time.Millisecond)))
	gt.NoError(t, err)

	// Create usecase instance
	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
	)

	// Execute test
	err = uc.HandleSlackInteractionBlockActions(
		ctx,
		user,
		thread,
		slack.ActionIDAckList,
		testList.ID.String(),
		"trigger-id",
	)

	// Verify results
	gt.NoError(t, err)

	// Verify ticket was created
	tickets, err := repo.GetTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusOpen}, 0, 0)
	gt.NoError(t, err)
	gt.Array(t, tickets).Length(1)

	ticket := tickets[0]
	gt.Value(t, ticket.Assignee.ID).Equal(user.ID)
	gt.Value(t, ticket.SlackThread.ChannelID).Equal(thread.ChannelID)
	gt.Value(t, ticket.SlackThread.ThreadID).Equal(thread.ThreadID)

	// Verify alerts were updated with ticket ID
	for _, alert := range testAlerts {
		updatedAlert, err := repo.GetAlert(ctx, alert.ID)
		gt.NoError(t, err)
		gt.Value(t, updatedAlert.TicketID).Equal(ticket.ID)
	}

	// Verify alert list status was updated to bound
	updatedList, err := repo.GetAlertList(ctx, testList.ID)
	gt.NoError(t, err)
	gt.Value(t, updatedList.Status).Equal(alert.ListStatusBound)

	// Wait for async alert updates to complete
	time.Sleep(200 * time.Millisecond)

	// Verify Slack interactions (updated for unified CreateTicketFromAlerts)
	// Note: 5 PostMessage calls due to ticket creation flow (trace messages now post instead of update) + 2 UpdateMessage (2 alerts)
	// Changed from 4 PostMessage + 3 UpdateMessage because msg.Trace now always posts new messages
	gt.Value(t, len(slackMock.PostMessageContextCalls())).Equal(5)
	gt.Value(t, len(slackMock.UpdateMessageContextCalls())).Equal(2)
}

func TestSlackActionBindAlert(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	// Setup mocks and repositories
	repo := repository.NewMemory()

	slackMock := &mock.SlackClientMock{
		OpenViewFunc: func(triggerID string, view slack_sdk.ModalViewRequest) (*slack_sdk.ViewResponse, error) {
			checkMockCaller(t, 0,
				"github.com/secmon-lab/warren/pkg/service/slack",
				"(*Service).ShowBindToTicketModal",
			)

			return &slack_sdk.ViewResponse{}, nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
	}

	// Create test alert
	testAlert := alert.Alert{
		ID:        types.NewAlertID(),
		CreatedAt: now,
		Metadata: alert.Metadata{
			Title:       "Test Alert",
			Description: "Test Description",
		},
		Embedding: make256DimEmbedding(),
		SlackThread: &slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  "test-thread",
		},
		Data: map[string]interface{}{"key": "value"},
	}

	// Create test ticket
	testTicket := ticket.Ticket{
		ID:        types.TicketID("test-ticket-1"),
		CreatedAt: now,
		Status:    types.TicketStatusOpen,
		Embedding: []float32{0.15, 0.25, 0.35},
		SlackThread: &slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  "test-thread",
		},
	}

	// Store test alert and ticket
	if err := repo.PutAlert(ctx, testAlert); err != nil {
		t.Fatal(err)
	}
	if err := repo.PutTicket(ctx, testTicket); err != nil {
		t.Fatal(err)
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	// Create usecase instance
	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
	)

	// Execute test
	err = uc.HandleSlackInteractionBlockActions(
		ctx,
		slack.User{},
		slack.Thread{},
		slack.ActionIDBindAlert,
		testAlert.ID.String(),
		"trigger-id",
	)

	// Verify results
	gt.NoError(t, err)
	gt.Value(t, len(slackMock.OpenViewCalls())).Equal(1)
}

func TestSlackActionBindList(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	// Setup mocks and repositories
	repo := repository.NewMemory()

	slackMock := &mock.SlackClientMock{
		OpenViewFunc: func(triggerID string, view slack_sdk.ModalViewRequest) (*slack_sdk.ViewResponse, error) {
			return &slack_sdk.ViewResponse{}, nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
	}

	// Create test alerts
	testAlerts := alert.Alerts{
		&alert.Alert{
			ID:        types.AlertID("test-alert-1"),
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: "Test Alert 1"},
			Embedding: []float32{0.1, 0.2, 0.3},
			SlackThread: &slack.Thread{
				ChannelID: "test-channel",
				ThreadID:  "test-thread",
			},
		},
		&alert.Alert{
			ID:        types.AlertID("test-alert-2"),
			CreatedAt: now,
			Metadata:  alert.Metadata{Title: "Test Alert 2"},
			Embedding: []float32{0.2, 0.3, 0.4},
			SlackThread: &slack.Thread{
				ChannelID: "test-channel",
				ThreadID:  "test-thread",
			},
		},
	}

	// Create test alert list
	thread := slack.Thread{
		ChannelID: "test-channel",
		ThreadID:  "test-thread",
	}
	user := slack.User{ID: "test-user"}
	testList := alert.NewList(ctx, thread, &user, testAlerts)

	// Create test ticket
	testTicket := ticket.Ticket{
		ID:        types.TicketID("test-ticket-1"),
		CreatedAt: now,
		Status:    types.TicketStatusOpen,
		Embedding: []float32{0.15, 0.25, 0.35},
	}

	// Store test data
	for _, alert := range testAlerts {
		if err := repo.PutAlert(ctx, *alert); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.PutAlertList(ctx, testList); err != nil {
		t.Fatal(err)
	}
	if err := repo.PutTicket(ctx, testTicket); err != nil {
		t.Fatal(err)
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	// Create usecase instance
	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
	)

	// Execute test
	err = uc.HandleSlackInteractionBlockActions(
		ctx,
		user,
		thread,
		slack.ActionIDBindList,
		testList.ID.String(),
		"trigger-id",
	)

	// Verify results
	gt.NoError(t, err)
	gt.Value(t, len(slackMock.OpenViewCalls())).Equal(1)
}

func TestSlackActionResolveTicket(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	// Setup mocks and repositories
	repo := repository.NewMemory()

	slackMock := &mock.SlackClientMock{
		OpenViewFunc: func(triggerID string, view slack_sdk.ModalViewRequest) (*slack_sdk.ViewResponse, error) {
			return &slack_sdk.ViewResponse{}, nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
	}

	// Create test ticket
	testTicket := ticket.Ticket{
		ID:        types.TicketID("test-ticket-1"),
		CreatedAt: now,
		Status:    types.TicketStatusOpen,
		Embedding: []float32{0.15, 0.25, 0.35},
		SlackThread: &slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  "test-thread",
		},
	}

	// Store test ticket
	if err := repo.PutTicket(ctx, testTicket); err != nil {
		t.Fatal(err)
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
	gt.NoError(t, err)

	// Create usecase instance
	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
	)

	// Test data
	user := slack.User{ID: "test-user"}
	thread := slack.Thread{
		ChannelID: "test-channel",
		ThreadID:  "test-thread",
	}

	// Execute test
	err = uc.HandleSlackInteractionBlockActions(
		ctx,
		user,
		thread,
		slack.ActionIDResolveTicket,
		testTicket.ID.String(),
		"trigger-id",
	)

	// Verify results
	gt.NoError(t, err)
	gt.Value(t, len(slackMock.OpenViewCalls())).Equal(1)
}

func TestSlackActionAckAlert_MultipleAlertLists(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	// Setup mocks and repositories
	repo := repository.NewMemory()

	// Create test alert
	testAlert := alert.Alert{
		ID:        types.NewAlertID(),
		CreatedAt: now,
		Metadata: alert.Metadata{
			Title:       "Test Alert",
			Description: "Test Description",
		},
		SlackThread: &slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  "test-thread",
		},
		Embedding: make256DimEmbedding(),
		Data:      map[string]interface{}{"key": "value"},
	}

	// Store test alert
	if err := repo.PutAlert(ctx, testAlert); err != nil {
		t.Fatal(err)
	}

	// Create multiple alert lists in the same thread
	alertList1 := &alert.List{
		ID: types.NewAlertListID(),
		Metadata: alert.Metadata{
			Title:       "Alert List 1",
			Description: "First alert list",
		},
		AlertIDs: []types.AlertID{testAlert.ID},
		SlackThread: &slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  "test-thread",
		},
		CreatedAt: now.Add(-time.Hour),
		CreatedBy: &slack.User{ID: "user1", Name: "User 1"},
	}

	alertList2 := &alert.List{
		ID: types.NewAlertListID(),
		Metadata: alert.Metadata{
			Title:       "Alert List 2",
			Description: "Second alert list",
		},
		AlertIDs: []types.AlertID{testAlert.ID},
		SlackThread: &slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  "test-thread",
		},
		CreatedAt: now.Add(-30 * time.Minute),
		CreatedBy: &slack.User{ID: "user2", Name: "User 2"},
	}

	// Store alert lists
	if err := repo.PutAlertList(ctx, alertList1); err != nil {
		t.Fatal(err)
	}
	if err := repo.PutAlertList(ctx, alertList2); err != nil {
		t.Fatal(err)
	}

	var postMessageCalls []slack_sdk.MsgOption
	var postMessageCallCount int

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			postMessageCallCount++
			postMessageCalls = append(postMessageCalls, options...)

			// Return different timestamps for different calls
			return "test-channel", fmt.Sprintf("timestamp-%d", postMessageCallCount), nil
		},
		UpdateMessageContextFunc: func(ctx context.Context, channelID, timestamp string, options ...slack_sdk.MsgOption) (string, string, string, error) {
			return channelID, timestamp, "updated-timestamp", nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "test-user",
				TeamID: "test-team",
				Team:   "test-team-name",
			}, nil
		},
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel",
		slack_svc.WithUpdaterOptions(
			slack_svc.WithInterval(1*time.Millisecond),
			slack_svc.WithRetryInterval(1*time.Millisecond)))
	gt.NoError(t, err)

	// Mock LLM client
	llmMock := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					// Return appropriate JSON response for ticket metadata
					return &gollem.Response{
						Texts: []string{`{"title": "Test Ticket", "description": "Test Description", "summary": "Test Summary"}`},
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

	// Create usecase instance
	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
	)

	// Test data
	user := slack.User{ID: "test-user", Name: "Test User"}
	thread := slack.Thread{
		ChannelID: "test-channel",
		ThreadID:  "test-thread",
	}

	// Execute test
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

	// Verify ticket was created
	tickets, err := repo.GetTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusOpen}, 0, 0)
	gt.NoError(t, err)
	gt.Array(t, tickets).Length(1)

	ticket := tickets[0]
	gt.Value(t, ticket.Assignee.ID).Equal(user.ID)

	// Verify ticket is in new thread, not original thread
	gt.Value(t, ticket.SlackThread.ChannelID).Equal("test-channel")
	// The ticket should NOT be in the original thread
	gt.Value(t, ticket.SlackThread.ThreadID).NotEqual("test-thread")

	// Verify that multiple PostMessage calls were made (ticket + link + other processing)
	gt.Value(t, postMessageCallCount >= 2).Equal(true)

	// Verify alert was updated with ticket ID
	updatedAlert, err := repo.GetAlert(ctx, testAlert.ID)
	gt.NoError(t, err)
	gt.Value(t, updatedAlert.TicketID).Equal(ticket.ID)

	// Wait for async alert updates to complete
	time.Sleep(200 * time.Millisecond)
}

func TestSlackActionAckAlert_SingleAlertList(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	// Setup mocks and repositories
	repo := repository.NewMemory()

	// Create test alert
	testAlert := alert.Alert{
		ID:        types.NewAlertID(),
		CreatedAt: now,
		Metadata: alert.Metadata{
			Title:       "Test Alert",
			Description: "Test Description",
		},
		SlackThread: &slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  "test-thread",
		},
		Embedding: make256DimEmbedding(),
		Data:      map[string]interface{}{"key": "value"},
	}

	// Store test alert
	if err := repo.PutAlert(ctx, testAlert); err != nil {
		t.Fatal(err)
	}

	// Create only one alert list in the thread
	alertList := &alert.List{
		ID: types.NewAlertListID(),
		Metadata: alert.Metadata{
			Title:       "Alert List",
			Description: "Single alert list",
		},
		AlertIDs: []types.AlertID{testAlert.ID},
		SlackThread: &slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  "test-thread",
		},
		CreatedAt: now.Add(-time.Hour),
		CreatedBy: &slack.User{ID: "user1", Name: "User 1"},
	}

	// Store alert list
	if err := repo.PutAlertList(ctx, alertList); err != nil {
		t.Fatal(err)
	}

	var postMessageCallCount int

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			postMessageCallCount++
			return "test-channel", fmt.Sprintf("timestamp-%d", postMessageCallCount), nil
		},
		UpdateMessageContextFunc: func(ctx context.Context, channelID, timestamp string, options ...slack_sdk.MsgOption) (string, string, string, error) {
			return channelID, timestamp, "updated-timestamp", nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "test-user",
				TeamID: "test-team",
				Team:   "test-team-name",
			}, nil
		},
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel",
		slack_svc.WithUpdaterOptions(
			slack_svc.WithInterval(1*time.Millisecond),
			slack_svc.WithRetryInterval(1*time.Millisecond)))
	gt.NoError(t, err)

	// Mock LLM client
	llmMock := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					// Return appropriate JSON response for ticket metadata
					return &gollem.Response{
						Texts: []string{`{"title": "Test Ticket", "description": "Test Description", "summary": "Test Summary"}`},
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

	// Create usecase instance
	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
	)

	// Test data
	user := slack.User{ID: "test-user", Name: "Test User"}
	thread := slack.Thread{
		ChannelID: "test-channel",
		ThreadID:  "test-thread",
	}

	// Execute test
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

	// Verify ticket was created
	tickets, err := repo.GetTicketsByStatus(ctx, []types.TicketStatus{types.TicketStatusOpen}, 0, 0)
	gt.NoError(t, err)
	gt.Array(t, tickets).Length(1)

	ticket := tickets[0]
	gt.Value(t, ticket.Assignee.ID).Equal(user.ID)

	// Verify ticket is in the SAME thread (original thread)
	gt.Value(t, ticket.SlackThread.ChannelID).Equal("test-channel")
	gt.Value(t, ticket.SlackThread.ThreadID).Equal("test-thread")

	// Verify alert was updated with ticket ID
	updatedAlert, err := repo.GetAlert(ctx, testAlert.ID)
	gt.NoError(t, err)
	gt.Value(t, updatedAlert.TicketID).Equal(ticket.ID)

	// Wait for async alert updates to complete
	time.Sleep(200 * time.Millisecond)
}

func TestSlackActionAckAlert_UpdatesAlertList(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	// Setup mocks and repositories
	repo := repository.NewMemory()

	var updateMessageCount int
	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			return "test-channel", "test-timestamp", nil
		},
		UpdateMessageContextFunc: func(ctx context.Context, channelID, timestamp string, options ...slack_sdk.MsgOption) (string, string, string, error) {
			updateMessageCount++
			return channelID, timestamp, "updated-timestamp", nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "test-user",
				TeamID: "test-team",
				Team:   "test-team-name",
			}, nil
		},
		GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
			return &slack_sdk.TeamInfo{
				ID:     "test-team",
				Domain: "test-domain",
			}, nil
		},
	}

	// Mock LLM client - for single alert tickets, LLM should not be called
	llmMock := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					t.Fatal("LLM should not be called for single alert ticket - metadata should be inherited")
					return nil, nil
				},
			}, nil
		},
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, inputs []string) ([][]float64, error) {
			t.Fatal("LLM embedding should not be called for single alert ticket - embedding should be inherited")
			return nil, nil
		},
	}

	// Create test alert
	testAlert := alert.Alert{
		ID:        types.NewAlertID(),
		CreatedAt: now,
		Metadata: alert.Metadata{
			Title:       "Test Alert",
			Description: "Test Description",
		},
		SlackThread: &slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  "test-thread",
		},
		Embedding: make256DimEmbedding(),
		Data:      map[string]interface{}{"key": "value"},
	}

	// Store test alert
	if err := repo.PutAlert(ctx, testAlert); err != nil {
		t.Fatal(err)
	}

	// Create alert list with SlackMessageID
	alertList := &alert.List{
		ID: types.NewAlertListID(),
		Metadata: alert.Metadata{
			Title:       "Test Alert List",
			Description: "Alert list for testing",
		},
		AlertIDs: []types.AlertID{testAlert.ID},
		SlackThread: &slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  "test-thread",
		},
		SlackMessageID: "alert-list-message-id",
		CreatedAt:      now.Add(-time.Hour),
		CreatedBy:      &slack.User{ID: "user1", Name: "User 1"},
	}

	// Store alert list
	if err := repo.PutAlertList(ctx, alertList); err != nil {
		t.Fatal(err)
	}

	// Use fast interval for testing
	slackSvc, err := slack_svc.New(slackMock, "#test-channel",
		slack_svc.WithUpdaterOptions(
			slack_svc.WithInterval(1*time.Millisecond),
			slack_svc.WithRetryInterval(1*time.Millisecond)))
	gt.NoError(t, err)

	// Create usecase instance
	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slackSvc),
		usecase.WithLLMClient(llmMock),
	)

	// Execute test
	err = uc.HandleSlackInteractionBlockActions(
		ctx,
		slack.User{ID: "test-user"},
		slack.Thread{ChannelID: "test-channel", ThreadID: "test-thread"},
		slack.ActionIDAckAlert,
		testAlert.ID.String(),
		"",
	)

	// Verify results
	gt.NoError(t, err)

	// Wait for async alert updates to complete
	time.Sleep(200 * time.Millisecond)

	// Verify that alert list message was updated
	gt.Number(t, updateMessageCount).Greater(0)

	// Verify alert was bound to ticket
	updatedAlert, err := repo.GetAlert(ctx, testAlert.ID)
	gt.NoError(t, err)
	gt.Value(t, updatedAlert.TicketID).NotEqual(types.EmptyTicketID)
}
