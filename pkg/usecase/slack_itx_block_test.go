package usecase_test

import (
	"context"
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
	llmCallCount := 0
	llmMock := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &gollem_mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					llmCallCount++
					switch llmCallCount {
					case 1:
						// creating summary
						checkMockCaller(t, 2,
							"github.com/secmon-lab/warren/pkg/domain/model/ticket",
							"(*Ticket).FillMetadata",
						)
						return &gollem.Response{
							Texts: []string{
								`{"title": "test title", "description": "test description"}`,
							},
						}, nil

					case 2:
						// summarize ticket
						checkMockCaller(t, 2,
							"github.com/secmon-lab/warren/pkg/domain/model/ticket",
							"(*Ticket).FillMetadata",
						)
						return &gollem.Response{
							Texts: []string{
								`{"title": "test title", "description": "test description", "summary": "test summary"}`,
							},
						}, nil

					case 3:
						// ack alerts
						checkMockCaller(t, 2,
							"github.com/secmon-lab/warren/pkg/usecase",
							"(*UseCases).ackAlerts",
						)
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
	}

	// Create test alert
	testAlert := &alert.Alert{
		ID:        types.AlertID("test-alert-1"),
		CreatedAt: now,
		Metadata:  alert.Metadata{Title: "Test Alert"},
		SlackThread: &slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  "test-thread",
		},
		Embedding: []float32{0.1, 0.2, 0.3},
	}

	// Store test alert
	if err := repo.PutAlert(ctx, *testAlert); err != nil {
		t.Fatal(err)
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
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

	// Verify Slack interactions
	gt.Value(t, len(slackMock.PostMessageContextCalls())).Equal(4)
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

	// Store test alerts and list
	for _, alert := range testAlerts {
		if err := repo.PutAlert(ctx, *alert); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.PutAlertList(ctx, testList); err != nil {
		t.Fatal(err)
	}

	slackSvc, err := slack_svc.New(slackMock, "#test-channel")
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

	// Verify Slack interactions
	gt.Value(t, len(slackMock.PostMessageContextCalls())).Equal(4)
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
		ID:        types.AlertID("test-alert-1"),
		CreatedAt: now,
		Metadata:  alert.Metadata{Title: "Test Alert"},
		Embedding: []float32{0.1, 0.2, 0.3},
		SlackThread: &slack.Thread{
			ChannelID: "test-channel",
			ThreadID:  "test-thread",
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
