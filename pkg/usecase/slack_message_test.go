package usecase_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	slack_sdk "github.com/slack-go/slack"
)

// TestSlackThreadService_NewTraceMessage verifies that SlackThreadService has NewTraceMessage method
// and that it's used for context block handling instead of regular messages
func TestSlackThreadService_NewTraceMessage(t *testing.T) {
	ctx := context.Background()

	// Create mock thread service to test the interface
	var traceMessageCalled bool
	var traceCallCount int

	mockThreadService := &mock.SlackThreadServiceMock{
		ReplyFunc: func(ctx context.Context, message string) {
			// Regular message handler - should NOT be called for trace messages
		},
		NewTraceMessageFunc: func(ctx context.Context, initialMessage string) func(ctx context.Context, traceMsg string) {
			traceMessageCalled = true
			return func(ctx context.Context, traceMsg string) {
				traceCallCount++
				// This function should be called when msg.Trace is used
				// In real implementation, this would create context blocks
			}
		},
	}

	// Verify the interface compliance
	var _ interfaces.SlackThreadService = mockThreadService

	// Test NewTraceMessage functionality
	gt.V(t, traceMessageCalled).Equal(false) // Should be false initially

	// Call NewTraceMessage to get the trace function
	traceFunc := mockThreadService.NewTraceMessage(ctx, "Initial trace message")
	gt.V(t, traceMessageCalled).Equal(true) // Should be true after setup

	// Test the returned trace function
	traceFunc(ctx, "First trace update")
	traceFunc(ctx, "Second trace update")

	gt.V(t, traceCallCount).Equal(2) // Should have been called twice

	// Verify mock calls tracking
	calls := mockThreadService.NewTraceMessageCalls()
	gt.V(t, len(calls)).Equal(1)
	gt.V(t, calls[0].InitialMessage).Equal("Initial trace message")
}

// TestSlackService_NewTraceMessage tests that the actual Slack service implements the method
func TestSlackService_NewTraceMessage(t *testing.T) {
	// This test verifies that our actual slack service implementation
	// has the NewTraceMessage method and it creates proper context blocks

	var postCallCount int
	var updateCallCount int
	// Create a mock Slack client
	mockClient := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			postCallCount++
			return channelID, "1234567890.123456", nil
		},
		UpdateMessageContextFunc: func(ctx context.Context, channelID, timestamp string, options ...slack_sdk.MsgOption) (string, string, string, error) {
			updateCallCount++
			return channelID, timestamp, "1234567890.123457", nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				TeamID: "T123456",
				Team:   "test-team",
				UserID: "U123456",
				BotID:  "B123456",
			}, nil
		},
		GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
			return &slack_sdk.TeamInfo{
				Domain: "test-workspace",
			}, nil
		},
	}

	// Create real slack service to test actual implementation
	slackSvc, err := slack_svc.New(mockClient, "C123456")
	gt.NoError(t, err)

	thread := slackSvc.NewThread(slack.Thread{
		ChannelID: "C123456",
		ThreadID:  "1234567890.123456",
		TeamID:    "T123456",
	})

	// Test that NewTraceMessage method exists and works
	ctx := context.Background()

	// CRITICAL: The initial message should be posted immediately when NewTraceMessage is called
	traceFunc := thread.NewTraceMessage(ctx, "Initial trace message")

	// Verify that the initial message was posted immediately (this was the bug - it wasn't being posted)
	gt.V(t, postCallCount).Equal(1)   // Should be 1 after NewTraceMessage call
	gt.V(t, updateCallCount).Equal(0) // Should still be 0

	// Call the trace function - this should update the existing message
	traceFunc(ctx, "Context update message")

	// Verify that the message was updated (not posted again)
	gt.V(t, postCallCount).Equal(1)   // Should still be 1
	gt.V(t, updateCallCount).Equal(1) // Should now be 1

	// Call the trace function again to test more updates
	traceFunc(ctx, "Another context update")

	// Verify that the message was updated again
	gt.V(t, postCallCount).Equal(1)   // Should still be 1
	gt.V(t, updateCallCount).Equal(2) // Should now be 2
	// The key behavior we've verified is:
	// 1. Initial message is posted immediately when NewTraceMessage is called
	// 2. Subsequent calls to the returned function update the existing message
	// 3. This matches the behavior of NewStateFunc and ensures no messages are lost
}

// TestSlackService_NewTraceMessage_EmptyInitial tests behavior with empty initial message
func TestSlackService_NewTraceMessage_EmptyInitial(t *testing.T) {
	var postCallCount int
	var updateCallCount int
	// Create a mock Slack client
	mockClient := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			postCallCount++
			return channelID, "1234567890.123456", nil
		},
		UpdateMessageContextFunc: func(ctx context.Context, channelID, timestamp string, options ...slack_sdk.MsgOption) (string, string, string, error) {
			updateCallCount++
			return channelID, timestamp, "1234567890.123457", nil
		},
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				TeamID: "T123456",
				Team:   "test-team",
				UserID: "U123456",
				BotID:  "B123456",
			}, nil
		},
		GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
			return &slack_sdk.TeamInfo{
				Domain: "test-workspace",
			}, nil
		},
	}

	// Create real slack service to test actual implementation
	slackSvc, err := slack_svc.New(mockClient, "C123456")
	gt.NoError(t, err)

	thread := slackSvc.NewThread(slack.Thread{
		ChannelID: "C123456",
		ThreadID:  "1234567890.123456",
		TeamID:    "T123456",
	})

	ctx := context.Background()

	// Test with empty initial message - should not post anything initially
	traceFunc := thread.NewTraceMessage(ctx, "")

	// Verify that no message was posted for empty initial message
	gt.V(t, postCallCount).Equal(0)
	gt.V(t, updateCallCount).Equal(0)

	// Call the trace function with actual content - this should post the first message
	traceFunc(ctx, "First actual message")

	// Verify that the message was posted (since there was no initial message)
	gt.V(t, postCallCount).Equal(1)
	gt.V(t, updateCallCount).Equal(0)

	// Call the trace function again - this should update the existing message
	traceFunc(ctx, "Update message")

	// Verify that the message was updated
	gt.V(t, postCallCount).Equal(1)
	gt.V(t, updateCallCount).Equal(1)
}
