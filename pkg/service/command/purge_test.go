package command_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	mock "github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/command"
	slackSDK "github.com/slack-go/slack"
)

func TestPurgeCommand(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully purges bot messages except thread parent and Ticket posts", func(t *testing.T) {
		// Setup
		repo := &mock.RepositoryMock{}
		slackClient := &mock.SlackClientMock{}
		threadService := &mock.SlackThreadServiceMock{}

		thread := &slack.Thread{
			ChannelID: "C0123456789",
			ThreadID:  "1234567890.000000", // This is the parent message (top-level alert) - will be protected
		}

		threadService.EntityFunc = func() *slack.Thread {
			return thread
		}
		ticketID := types.NewTicketID()

		// Create test ticket
		testTicket := &ticket.Ticket{
			ID:             ticketID,
			SlackMessageID: "1234567890.123456", // Protected message ID
			SlackThread:    thread,
			Metadata: ticket.Metadata{
				Title:       "Test Ticket",
				Description: "Test Description",
			},
			Status:    types.TicketStatusOpen,
			CreatedAt: time.Now(),
		}

		// Mock repository
		repo.GetAlertsByThreadFunc = func(ctx context.Context, t slack.Thread) (alert.Alerts, error) {
			// Return alert with message ID matching thread parent
			testAlert := &alert.Alert{
				ID:             types.NewAlertID(),
				SlackThread:    &t,
				SlackMessageID: "1234567890.000000", // Top-level alert message ID
			}
			return alert.Alerts{testAlert}, nil
		}

		repo.GetTicketByThreadFunc = func(ctx context.Context, thread slack.Thread) (*ticket.Ticket, error) {
			return testTicket, nil
		}

		// Mock Slack client
		slackClient.AuthTestFunc = func() (*slackSDK.AuthTestResponse, error) {
			return &slackSDK.AuthTestResponse{
				UserID: "UBOT123456",
			}, nil
		}

		slackClient.GetConversationRepliesContextFunc = func(ctx context.Context, params *slackSDK.GetConversationRepliesParameters) ([]slackSDK.Message, bool, string, error) {
			return []slackSDK.Message{
				{Msg: slackSDK.Msg{Timestamp: "1234567890.000000", User: "UBOT123456"}}, // Thread parent (top-level alert) - protected
				{Msg: slackSDK.Msg{Timestamp: "1234567890.123456", User: "UBOT123456"}}, // Ticket message - protected
				{Msg: slackSDK.Msg{Timestamp: "1234567890.333333", User: "UBOT123456"}}, // Bot message - should delete
				{Msg: slackSDK.Msg{Timestamp: "1234567890.444444", User: "UBOT123456"}}, // Bot message - should delete
				{Msg: slackSDK.Msg{Timestamp: "1234567890.555555", User: "UHUMAN123"}},  // Human message - skip
			}, false, "", nil
		}

		deletedMessages := []string{}
		slackClient.DeleteMessageContextFunc = func(ctx context.Context, channelID, timestamp string) (string, string, error) {
			deletedMessages = append(deletedMessages, timestamp)
			return channelID, timestamp, nil
		}

		// Create command service
		cmdSvc := command.NewWithUseCase(repo, nil, threadService, nil, slackClient, nil)

		// Execute purge command
		msg := &slack.Message{}
		err := cmdSvc.Execute(ctx, msg, "purge")
		gt.NoError(t, err)

		// Verify only non-protected bot messages were deleted
		gt.A(t, deletedMessages).Length(2)
		// Check that both expected messages are in the deleted list
		hasMsg1 := false
		hasMsg2 := false
		for _, msg := range deletedMessages {
			if msg == "1234567890.333333" {
				hasMsg1 = true
			}
			if msg == "1234567890.444444" {
				hasMsg2 = true
			}
		}
		gt.True(t, hasMsg1)
		gt.True(t, hasMsg2)
	})

	t.Run("works without ticket (only protects thread parent)", func(t *testing.T) {
		// Setup
		repo := &mock.RepositoryMock{}
		slackClient := &mock.SlackClientMock{}
		threadService := &mock.SlackThreadServiceMock{}

		thread := &slack.Thread{
			ChannelID: "C0123456789",
			ThreadID:  "1234567890.111111", // Thread parent is the top-level alert
		}

		threadService.EntityFunc = func() *slack.Thread {
			return thread
		}

		// Mock repository to return alert with message ID and nil ticket
		repo.GetAlertsByThreadFunc = func(ctx context.Context, t slack.Thread) (alert.Alerts, error) {
			// Return alert with message ID matching thread parent
			testAlert := &alert.Alert{
				ID:             types.NewAlertID(),
				SlackThread:    &t,
				SlackMessageID: "1234567890.111111", // Top-level alert message ID
			}
			return alert.Alerts{testAlert}, nil
		}

		repo.GetTicketByThreadFunc = func(ctx context.Context, thread slack.Thread) (*ticket.Ticket, error) {
			return nil, nil
		}

		slackClient.AuthTestFunc = func() (*slackSDK.AuthTestResponse, error) {
			return &slackSDK.AuthTestResponse{
				UserID: "UBOT123456",
			}, nil
		}

		slackClient.GetConversationRepliesContextFunc = func(ctx context.Context, params *slackSDK.GetConversationRepliesParameters) ([]slackSDK.Message, bool, string, error) {
			return []slackSDK.Message{
				{Msg: slackSDK.Msg{Timestamp: "1234567890.111111", User: "UBOT123456"}}, // Top-level alert - protected
				{Msg: slackSDK.Msg{Timestamp: "1234567890.222222", User: "UBOT123456"}}, // Bot message - should delete
			}, false, "", nil
		}

		deletedMessages := []string{}
		slackClient.DeleteMessageContextFunc = func(ctx context.Context, channelID, timestamp string) (string, string, error) {
			deletedMessages = append(deletedMessages, timestamp)
			return channelID, timestamp, nil
		}

		// Create service
		cmdSvc := command.NewWithUseCase(repo, nil, threadService, nil, slackClient, nil)

		// Execute purge command
		msg := &slack.Message{}
		err := cmdSvc.Execute(ctx, msg, "purge")
		gt.NoError(t, err)

		// Verify only non-protected bot messages were deleted
		gt.A(t, deletedMessages).Length(1)
		gt.V(t, deletedMessages[0]).Equal("1234567890.222222")
	})

	t.Run("handles empty message list gracefully", func(t *testing.T) {
		// Setup
		repo := &mock.RepositoryMock{}
		slackClient := &mock.SlackClientMock{}
		threadService := &mock.SlackThreadServiceMock{}

		thread := &slack.Thread{
			ChannelID: "C0123456789",
			ThreadID:  "1234567890.000000",
		}

		threadService.EntityFunc = func() *slack.Thread {
			return thread
		}
		ticketID := types.NewTicketID()

		testTicket := &ticket.Ticket{
			ID:             ticketID,
			SlackMessageID: "1234567890.123456",
			SlackThread:    thread,
			Status:         types.TicketStatusOpen,
			CreatedAt:      time.Now(),
		}

		repo.GetTicketByThreadFunc = func(ctx context.Context, thread slack.Thread) (*ticket.Ticket, error) {
			return testTicket, nil
		}

		repo.GetAlertsByThreadFunc = func(ctx context.Context, thread slack.Thread) (alert.Alerts, error) {
			return alert.Alerts{}, nil
		}

		slackClient.AuthTestFunc = func() (*slackSDK.AuthTestResponse, error) {
			return &slackSDK.AuthTestResponse{
				UserID: "UBOT123456",
			}, nil
		}

		slackClient.GetConversationRepliesContextFunc = func(ctx context.Context, params *slackSDK.GetConversationRepliesParameters) ([]slackSDK.Message, bool, string, error) {
			return []slackSDK.Message{}, false, "", nil
		}

		// Create service
		cmdSvc := command.NewWithUseCase(repo, nil, threadService, nil, slackClient, nil)

		// Execute purge command
		msg := &slack.Message{}
		err := cmdSvc.Execute(ctx, msg, "purge")

		// Should not error
		gt.NoError(t, err)
	})
}
