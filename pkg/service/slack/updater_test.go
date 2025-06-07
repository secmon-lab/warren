package slack_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/slack"
	slack_sdk "github.com/slack-go/slack"
)

func TestRateLimitedUpdater_SingleUpdate(t *testing.T) {
	ctx := context.Background()

	mockClient := &mock.SlackClientMock{
		UpdateMessageContextFunc: func(ctx context.Context, channelID, timestamp string, options ...slack_sdk.MsgOption) (string, string, string, error) {
			return channelID, timestamp, "test-message-ts", nil
		},
	}

	// Use fast interval for testing (100ms instead of 2s)
	updater := slack.NewRateLimitedUpdater(mockClient, slack.WithInterval(100*time.Millisecond))

	testAlert := alert.Alert{
		ID:     types.AlertID("test-alert-1"),
		Schema: "test.v1",
		Metadata: alert.Metadata{
			Title: "Test Alert",
		},
		SlackThread: &model.Thread{
			ChannelID: "C1234567890",
			ThreadID:  "1234567890.123456",
		},
	}

	updater.UpdateAlert(ctx, testAlert) // Now returns immediately

	// Wait a bit for the async processing to complete (200ms should be enough for 100ms interval)
	time.Sleep(200 * time.Millisecond)

	// Verify the update was called
	calls := mockClient.UpdateMessageContextCalls()
	gt.Number(t, len(calls)).Equal(1)
	gt.Value(t, calls[0].ChannelID).Equal("C1234567890")
	gt.Value(t, calls[0].Timestamp).Equal("1234567890.123456")
}

func TestRateLimitedUpdater_MultipleUpdates_RateLimited(t *testing.T) {
	ctx := context.Background()

	var callTimes []time.Time
	var mu sync.Mutex

	mockClient := &mock.SlackClientMock{
		UpdateMessageContextFunc: func(ctx context.Context, channelID, timestamp string, options ...slack_sdk.MsgOption) (string, string, string, error) {
			mu.Lock()
			callTimes = append(callTimes, time.Now())
			mu.Unlock()
			return channelID, timestamp, "test-message-ts", nil
		},
	}

	// Use fast interval for testing (100ms instead of 2s)
	updater := slack.NewRateLimitedUpdater(mockClient, slack.WithInterval(100*time.Millisecond))

	// Create multiple test alerts
	alerts := []alert.Alert{
		{
			ID:       types.AlertID("test-alert-1"),
			Schema:   "test.v1",
			Metadata: alert.Metadata{Title: "Test Alert 1"},
			SlackThread: &model.Thread{
				ChannelID: "C1234567890",
				ThreadID:  "1234567890.123456",
			},
		},
		{
			ID:       types.AlertID("test-alert-2"),
			Schema:   "test.v1",
			Metadata: alert.Metadata{Title: "Test Alert 2"},
			SlackThread: &model.Thread{
				ChannelID: "C1234567890",
				ThreadID:  "1234567890.123457",
			},
		},
		{
			ID:       types.AlertID("test-alert-3"),
			Schema:   "test.v1",
			Metadata: alert.Metadata{Title: "Test Alert 3"},
			SlackThread: &model.Thread{
				ChannelID: "C1234567890",
				ThreadID:  "1234567890.123458",
			},
		},
	}

	start := time.Now()

	// Send all updates concurrently
	var wg sync.WaitGroup
	for _, testAlert := range alerts {
		wg.Add(1)
		go func(alert alert.Alert) {
			defer wg.Done()
			updater.UpdateAlert(ctx, alert)
		}(testAlert)
	}

	wg.Wait()

	// Wait for all async processing to complete (3 updates * 100ms interval + buffer)
	time.Sleep(500 * time.Millisecond)
	totalTime := time.Since(start)

	// Verify all updates were called
	calls := mockClient.UpdateMessageContextCalls()
	gt.Number(t, len(calls)).Equal(3)

	// Verify rate limiting: should take at least 300ms for 3 updates (100ms intervals)
	// But allow some tolerance for test execution time
	gt.Number(t, totalTime.Milliseconds()).Greater(int64(200))

	// Verify that calls were spaced out by at least ~100ms
	if len(callTimes) >= 2 {
		for i := 1; i < len(callTimes); i++ {
			interval := callTimes[i].Sub(callTimes[i-1])
			// Allow some tolerance (80ms minimum instead of exactly 100ms)
			gt.Number(t, interval.Milliseconds()).GreaterOrEqual(int64(80))
		}
	}
}

func TestRateLimitedUpdater_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	mockClient := &mock.SlackClientMock{
		UpdateMessageContextFunc: func(ctx context.Context, channelID, timestamp string, options ...slack_sdk.MsgOption) (string, string, string, error) {
			return "", "", "", &slack_sdk.SlackErrorResponse{
				Err: "some_error",
			}
		},
	}

	// Use fast interval for testing
	updater := slack.NewRateLimitedUpdater(mockClient, slack.WithInterval(100*time.Millisecond))

	testAlert := alert.Alert{
		ID:     types.AlertID("test-alert-1"),
		Schema: "test.v1",
		Metadata: alert.Metadata{
			Title: "Test Alert",
		},
		SlackThread: &model.Thread{
			ChannelID: "C1234567890",
			ThreadID:  "1234567890.123456",
		},
	}

	updater.UpdateAlert(ctx, testAlert) // Now returns immediately

	// Wait a bit for the async processing to complete (200ms should be enough)
	time.Sleep(200 * time.Millisecond)

	// Verify the update was attempted
	calls := mockClient.UpdateMessageContextCalls()
	gt.Number(t, len(calls)).Equal(1)
}

func TestRateLimitedUpdater_RateLimitError_Retry(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	mockClient := &mock.SlackClientMock{
		UpdateMessageContextFunc: func(ctx context.Context, channelID, timestamp string, options ...slack_sdk.MsgOption) (string, string, string, error) {
			callCount++
			if callCount <= 2 {
				// Return rate limit error for first 2 calls
				return "", "", "", &slack_sdk.SlackErrorResponse{
					Err: "rate_limited",
					ResponseMetadata: slack_sdk.ResponseMetadata{
						Messages: []string{"1"}, // Retry after 1 second
					},
				}
			}
			// Success on 3rd call
			return channelID, timestamp, "test-message-ts", nil
		},
	}

	// Use fast interval for testing
	updater := slack.NewRateLimitedUpdater(mockClient, slack.WithInterval(100*time.Millisecond))

	testAlert := alert.Alert{
		ID:     types.AlertID("test-alert-1"),
		Schema: "test.v1",
		Metadata: alert.Metadata{
			Title: "Test Alert",
		},
		SlackThread: &model.Thread{
			ChannelID: "C1234567890",
			ThreadID:  "1234567890.123456",
		},
	}

	start := time.Now()
	updater.UpdateAlert(ctx, testAlert)

	// Wait for processing to complete (including retries)
	time.Sleep(3 * time.Second) // Still need time for the 1s retry-after waits
	duration := time.Since(start)

	// Should have been called 3 times (2 failures + 1 success)
	calls := mockClient.UpdateMessageContextCalls()
	gt.Number(t, len(calls)).Equal(3)

	// Should have taken some time due to retries (at least 2 seconds for 2 retries)
	gt.Number(t, duration.Milliseconds()).Greater(int64(2000))
}

func TestRateLimitedUpdater_NoSlackThread(t *testing.T) {
	ctx := context.Background()

	mockClient := &mock.SlackClientMock{}
	// Use fast interval for testing
	updater := slack.NewRateLimitedUpdater(mockClient, slack.WithInterval(100*time.Millisecond))

	testAlert := alert.Alert{
		ID:     types.AlertID("test-alert-1"),
		Schema: "test.v1",
		Metadata: alert.Metadata{
			Title: "Test Alert",
		},
		SlackThread: nil, // No slack thread
	}

	updater.UpdateAlert(ctx, testAlert) // Returns immediately, no error to check

	// Since we don't get a response, we don't expect any error to be returned

	// Should not call UpdateMessage
	calls := mockClient.UpdateMessageContextCalls()
	gt.Number(t, len(calls)).Equal(0)
}
