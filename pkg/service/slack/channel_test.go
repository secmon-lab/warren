package slack_test

import (
	"context"
	"errors"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/slack"
	slack_sdk "github.com/slack-go/slack"
)

func TestPostAlertWithChannel(t *testing.T) {
	tests := []struct {
		name            string
		alertChannel    string
		defaultChannel  string
		expectedChannel string
		postError       error
		expectFallback  bool
	}{
		{
			name:            "with channel specified",
			alertChannel:    "security-critical",
			defaultChannel:  "general",
			expectedChannel: "security-critical",
		},
		{
			name:            "with channel specified with # prefix",
			alertChannel:    "#security-critical",
			defaultChannel:  "general",
			expectedChannel: "security-critical",
		},
		{
			name:            "without channel specified",
			alertChannel:    "",
			defaultChannel:  "general",
			expectedChannel: "general",
		},
		{
			name:            "channel not found fallback",
			alertChannel:    "non-existent",
			defaultChannel:  "general",
			expectedChannel: "non-existent",
			postError:       errors.New("channel_not_found"),
			expectFallback:  true,
		},
		{
			name:            "not in channel fallback",
			alertChannel:    "private-channel",
			defaultChannel:  "general",
			expectedChannel: "private-channel",
			postError:       errors.New("not_in_channel"),
			expectFallback:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var actualChannel string
			var callCount int

			mockClient := &mock.SlackClientMock{
				PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
					callCount++
					if callCount == 1 {
						actualChannel = channelID
						if tc.postError != nil && channelID != tc.defaultChannel {
							return "", "", tc.postError
						}
					}
					return channelID, "1234567890.123456", nil
				},
				UploadFileV2ContextFunc: func(ctx context.Context, params slack_sdk.UploadFileV2Parameters) (*slack_sdk.FileSummary, error) {
					return &slack_sdk.FileSummary{}, nil
				},
				AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
					return &slack_sdk.AuthTestResponse{
						UserID: "U123456",
						BotID:  "B123456",
					}, nil
				},
				GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
					return &slack_sdk.TeamInfo{
						ID:   "T123456",
						Name: "test-team",
					}, nil
				},
			}

			svc, err := slack.New(mockClient, tc.defaultChannel)
			gt.NoError(t, err)

			testAlert := &alert.Alert{
				ID: types.NewAlertID(),
				Metadata: alert.Metadata{
					Title:       "Test Alert",
					Description: "Test Description",
					Channel:     tc.alertChannel,
				},
				Data: map[string]any{"test": "data"},
			}

			thread, err := svc.PostAlert(context.Background(), testAlert)

			if tc.expectFallback {
				// Should succeed after fallback
				gt.NoError(t, err)
				gt.NotNil(t, thread)
				gt.Equal(t, callCount, 2) // First attempt + fallback
			} else {
				gt.NoError(t, err)
				gt.NotNil(t, thread)
				gt.Equal(t, actualChannel, tc.expectedChannel)
				gt.Equal(t, callCount, 1) // Only one attempt
			}
		})
	}
}

func TestNormalizeChannel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal channel name",
			input:    "general",
			expected: "general",
		},
		{
			name:     "channel with # prefix",
			input:    "#general",
			expected: "general",
		},
		{
			name:     "channel with spaces",
			input:    "  general  ",
			expected: "general",
		},
		{
			name:     "channel with # and spaces",
			input:    "  #general  ",
			expected: "general",
		},
		{
			name:     "empty channel",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mock.SlackClientMock{
				AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
					return &slack_sdk.AuthTestResponse{}, nil
				},
				GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
					return &slack_sdk.TeamInfo{}, nil
				},
			}

			svc, err := slack.New(mockClient, "default")
			gt.NoError(t, err)

			// Export the normalizeChannel method for testing
			result := svc.NormalizeChannel(tc.input)
			gt.Equal(t, result, tc.expected)
		})
	}
}
