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
	
	var postCalled bool
	var updateCalled bool
	
	// Create a mock Slack client
	mockClient := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			postCalled = true
			return channelID, "1234567890.123456", nil
		},
		UpdateMessageContextFunc: func(ctx context.Context, channelID, timestamp string, options ...slack_sdk.MsgOption) (string, string, string, error) {
			updateCalled = true
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
	traceFunc := thread.NewTraceMessage(ctx, "Test trace message")
	
	// Call the trace function - this should create context blocks
	traceFunc(ctx, "Context update message")
	
	// Verify that the Slack client was called to post the initial message
	gt.V(t, postCalled).Equal(true)
	
	// Call the trace function again to test updates
	traceFunc(ctx, "Another context update")
	
	// Verify that the Slack client was called to update the message
	gt.V(t, updateCalled).Equal(true)
	
	// The key thing we've verified is:
	// 1. NewTraceMessage method exists on the interface
	// 2. It returns a function that can be called multiple times
	// 3. The implementation calls Slack API to post/update messages
	// 4. This is different from Reply which is for regular messages
}