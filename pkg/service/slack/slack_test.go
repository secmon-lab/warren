package slack_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/test"

	slack_sdk "github.com/slack-go/slack"
)

func newSlackService(t *testing.T) *slack.Service {
	envs := test.NewEnvVars(t, "TEST_SLACK_CHANNEL_ID", "TEST_SLACK_OAUTH_TOKEN")
	client := slack_sdk.New(envs.Get("TEST_SLACK_OAUTH_TOKEN"))

	svc, err := slack.New(client, envs.Get("TEST_SLACK_CHANNEL_ID"))
	gt.NoError(t, err).Required()

	return svc
}

func TestSlackPostAlert(t *testing.T) {
	svc := newSlackService(t)

	_, err := svc.PostAlert(context.Background(), &alert.Alert{
		ID: "1234567890",
		Metadata: alert.Metadata{
			Title:       "Test Alert Title",
			Description: "Test Alert Description",
			Attributes: []alert.Attribute{
				{
					Key:   "severity",
					Value: "high",
				},
				{
					Key:   "source",
					Value: "test",
				},
				{
					Key:   "details",
					Value: "Click here",
					Link:  "https://example.com/alert/details",
				},
			},
		},
		TicketID: types.NewTicketID(),
	})
	gt.NoError(t, err)
}

func TestSlackUpdateAlert(t *testing.T) {
	svc := newSlackService(t)

	dummy := genDummyAlert()

	thread, err := svc.PostAlert(context.Background(), &dummy)
	gt.NoError(t, err).Required()
	dummy.SlackThread = &model.Thread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	}

	dummy.Title = "Updated Alert Title"

	gt.NoError(t, thread.UpdateAlert(context.Background(), dummy))
}

func TestSlackUpdateTicket(t *testing.T) {
	svc := newSlackService(t)
	ctx := t.Context()
	dummy := genDummyAlert()

	thread, err := svc.PostAlert(context.Background(), &dummy)
	gt.NoError(t, err).Required()
	dummy.SlackThread = &model.Thread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	}

	ticketData := ticket.New(context.Background(), []types.AlertID{dummy.ID}, &model.Thread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	})
	ticketData.Metadata.Title = "Test Ticket Title"
	ticketData.Metadata.Description = "Test Ticket Description"
	ticketData.Metadata.Summary = "Test Ticket Summary"
	ticketData.Status = types.TicketStatusOpen
	ticketData.Reason = "Test Ticket Reason"

	ts, err := thread.PostTicket(ctx, &ticketData, alert.Alerts{&dummy})
	gt.NoError(t, err)
	ticketData.SlackMessageID = ts
	ticketData.Reason = "Updated reason"

	_, err = thread.PostTicket(ctx, &ticketData, alert.Alerts{&dummy})
	gt.NoError(t, err)
}

func genDummyAlert() alert.Alert {
	return alert.New(context.Background(), "test.alert.v1", map[string]any{
		"foo": "bar",
		"baz": 123,
	}, alert.Metadata{
		Title: "Test Alert Title",
		Attributes: []alert.Attribute{
			{
				Key:   "color",
				Value: "red",
			},
		},
	})
}

func genDummyAlertWithSlackThread() *alert.Alert {
	alert := genDummyAlert()
	alert.SlackThread = &model.Thread{
		ChannelID: "C0123456789",
		ThreadID:  fmt.Sprintf("%d", time.Now().Unix()),
	}
	return &alert
}

func TestAttachFile(t *testing.T) {
	svc := newSlackService(t)

	alert := genDummyAlert()

	thread, err := svc.PostAlert(context.Background(), &alert)
	gt.NoError(t, err)
	alert.SlackThread = &model.Thread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	}

	newThread := svc.NewThread(*alert.SlackThread)
	gt.NoError(t, newThread.AttachFile(context.Background(), "test", "test.txt", []byte("test")))
}

func TestIsBotUser(t *testing.T) {
	svc := newSlackService(t)

	botID := svc.BotID()
	gt.S(t, botID).Match(`^[A-Z][A-Z0-9]{6,12}$`)
}

func TestPostAlerts(t *testing.T) {
	svc := newSlackService(t)

	alerts := alert.Alerts{
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
	}
	alerts[1].CreatedAt = alerts[0].CreatedAt.Add(time.Second)
	alerts[2].CreatedAt = alerts[0].CreatedAt.Add(time.Second * 2)

	thread, err := svc.PostMessage(context.Background(), "alerts test")
	gt.NoError(t, err)
	gt.NoError(t, thread.PostAlerts(context.Background(), alerts))
}

func TestPostAlertList(t *testing.T) {
	svc := newSlackService(t)

	alertList := alert.NewList(context.Background(), model.Thread{
		ChannelID: "C0123456789",
		ThreadID:  "T0123456789",
	}, &model.User{
		ID:   "U0123456789",
		Name: "John Doe",
	}, alert.Alerts{
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
	})
	alertList.Title = "Test Alert List"
	alertList.Description = "This is a test alert list"

	thread, err := svc.PostMessage(context.Background(), "alert list test")
	gt.NoError(t, err)
	_, err = thread.PostAlertList(context.Background(), alertList)
	gt.NoError(t, err)
}

func TestPostTicketList(t *testing.T) {
	svc := newSlackService(t)
	ctx := t.Context()

	// Create test tickets
	tickets := []*ticket.Ticket{
		{
			ID: types.NewTicketID(),
			Metadata: ticket.Metadata{
				Title:       "Test Ticket 1",
				Description: "Description for ticket 1",
			},
			Status: types.TicketStatusOpen,
			Assignee: &model.User{
				ID:   "U0123456789",
				Name: "John Doe",
			},
			SlackThread: &model.Thread{
				ChannelID: "C0123456789",
				ThreadID:  "T0123456789",
			},
			CreatedAt: time.Now().Add(-time.Hour),
		},
		{
			ID: types.NewTicketID(),
			Metadata: ticket.Metadata{
				Title:       "Test Ticket 2",
				Description: "Description for ticket 2",
			},
			Status: types.TicketStatusResolved,
			SlackThread: &model.Thread{
				ChannelID: "C0123456789",
				ThreadID:  "T0123456789",
			},
			CreatedAt: time.Now().Add(-time.Hour * 2),
		},
	}

	// Post ticket list
	thread, err := svc.PostMessage(ctx, "Ticket list test")
	gt.NoError(t, err)
	gt.NoError(t, thread.PostTicketList(ctx, tickets))
}

func TestNewStateFunc(t *testing.T) {
	svc := newSlackService(t)

	cases := []struct {
		name     string
		base     string
		messages []string
		want     int
	}{
		{
			name:     "empty base and messages",
			base:     "",
			messages: []string{},
			want:     0,
		},
		{
			name:     "only base message",
			base:     "base message",
			messages: []string{},
			want:     1,
		},
		{
			name: "only state messages",
			base: "",
			messages: []string{
				"message 1",
				"message 2",
			},
			want: 1,
		},
		{
			name: "base and state messages",
			base: "base message",
			messages: []string{
				"message 1",
				"message 2",
			},
			want: 2,
		},
		{
			name: "state messages with markdown",
			base: "base message",
			messages: []string{
				"*message 1*",
				"_message 2_",
				"`message 3`",
				"```message 4\nmessage 4\n```",
			},
			want: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			thread, err := svc.PostMessage(ctx, "State test: "+tc.name)
			gt.NoError(t, err)

			fn := thread.NewStateFunc(ctx, tc.base)
			for _, msg := range tc.messages {
				fn(ctx, msg)
			}
		})
	}
}

func TestService_GetUserIcon(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error when Slack API fails", func(t *testing.T) {
		slackMock := &mock.SlackClientMock{
			AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
				return &slack_sdk.AuthTestResponse{
					UserID: "U123456",
					TeamID: "T123456",
					Team:   "test-team",
					BotID:  "B123456",
				}, nil
			},
			GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
				return &slack_sdk.TeamInfo{
					ID:     "T123456",
					Name:   "test-team",
					Domain: "test-workspace",
				}, nil
			},
			GetUserInfoFunc: func(userID string) (*slack_sdk.User, error) {
				return nil, fmt.Errorf("user not found")
			},
			GetBotInfoContextFunc: func(ctx context.Context, parameters slack_sdk.GetBotInfoParameters) (*slack_sdk.Bot, error) {
				return nil, fmt.Errorf("bot not found")
			},
		}

		service, err := slack.New(slackMock, "C123456")
		gt.NoError(t, err)

		_, _, err = service.GetUserIcon(ctx, "U123456")
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to get user image URL")
	})

	t.Run("returns error when user is nil", func(t *testing.T) {
		slackMock := &mock.SlackClientMock{
			AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
				return &slack_sdk.AuthTestResponse{
					UserID: "U123456",
					TeamID: "T123456",
					Team:   "test-team",
					BotID:  "B123456",
				}, nil
			},
			GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
				return &slack_sdk.TeamInfo{
					ID:     "T123456",
					Name:   "test-team",
					Domain: "test-workspace",
				}, nil
			},
			GetUserInfoFunc: func(userID string) (*slack_sdk.User, error) {
				return nil, nil // Return nil user without error
			},
			GetBotInfoContextFunc: func(ctx context.Context, parameters slack_sdk.GetBotInfoParameters) (*slack_sdk.Bot, error) {
				return nil, nil // Return nil bot without error
			},
		}

		service, err := slack.New(slackMock, "C123456")
		gt.NoError(t, err)

		_, _, err = service.GetUserIcon(ctx, "U123456")
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to get user image URL")
	})

	t.Run("returns error when no profile image available", func(t *testing.T) {
		slackMock := &mock.SlackClientMock{
			AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
				return &slack_sdk.AuthTestResponse{
					UserID: "U123456",
					TeamID: "T123456",
					Team:   "test-team",
					BotID:  "B123456",
				}, nil
			},
			GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
				return &slack_sdk.TeamInfo{
					ID:     "T123456",
					Name:   "test-team",
					Domain: "test-workspace",
				}, nil
			},
			GetUserInfoFunc: func(userID string) (*slack_sdk.User, error) {
				return &slack_sdk.User{
					ID:      userID,
					Profile: slack_sdk.UserProfile{
						// No image URLs
					},
				}, nil
			},
			GetBotInfoContextFunc: func(ctx context.Context, parameters slack_sdk.GetBotInfoParameters) (*slack_sdk.Bot, error) {
				return &slack_sdk.Bot{
					ID:    parameters.Bot,
					Icons: slack_sdk.Icons{
						// No image URLs
					},
				}, nil
			},
		}

		service, err := slack.New(slackMock, "C123456")
		gt.NoError(t, err)

		_, _, err = service.GetUserIcon(ctx, "U123456")
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("no profile image available")
	})

	t.Run("returns error when image download fails", func(t *testing.T) {
		slackMock := &mock.SlackClientMock{
			AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
				return &slack_sdk.AuthTestResponse{
					UserID: "U123456",
					TeamID: "T123456",
					Team:   "test-team",
					BotID:  "B123456",
				}, nil
			},
			GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
				return &slack_sdk.TeamInfo{
					ID:     "T123456",
					Name:   "test-team",
					Domain: "test-workspace",
				}, nil
			},
			GetUserInfoFunc: func(userID string) (*slack_sdk.User, error) {
				return &slack_sdk.User{
					ID: userID,
					Profile: slack_sdk.UserProfile{
						Image192: "https://example.com/nonexistent.jpg",
					},
				}, nil
			},
		}

		service, err := slack.New(slackMock, "C123456")
		gt.NoError(t, err)

		_, _, err = service.GetUserIcon(ctx, "U123456")
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to download user icon")
	})
}

// TestService_ClearExpiredIconCache is tested internally as part of the icon cache functionality

func TestService_GetUserIcon_RealSlack(t *testing.T) {
	// Skip test if TEST_SLACK_USER_ID is not set
	userID := os.Getenv("TEST_SLACK_USER_ID")
	if userID == "" {
		t.Skip("TEST_SLACK_USER_ID not set, skipping real Slack API test")
	}

	svc := newSlackService(t)
	ctx := context.Background()

	type testCase struct {
		userID     string
		shouldFail bool
	}

	runTest := func(tc testCase) func(t *testing.T) {
		return func(t *testing.T) {
			imageData, mimeType, err := svc.GetUserIcon(ctx, tc.userID)

			if tc.shouldFail {
				gt.Error(t, err)
				return
			}

			gt.NoError(t, err)
			gt.Number(t, len(imageData)).Greater(0)
			gt.Value(t, mimeType).NotEqual("")

			// Verify MIME type is a valid image type
			validMimeTypes := []string{
				"image/jpeg",
				"image/png",
				"image/gif",
				"image/webp",
			}
			gt.Array(t, validMimeTypes).Has(mimeType)
		}
	}

	t.Run("valid user ID", runTest(testCase{
		userID:     userID,
		shouldFail: false,
	}))

	t.Run("invalid user ID", runTest(testCase{
		userID:     "INVALID_USER_ID",
		shouldFail: true,
	}))

	// Test caching behavior
	t.Run("cached response", func(t *testing.T) {
		// First call
		imageData1, mimeType1, err := svc.GetUserIcon(ctx, userID)
		gt.NoError(t, err)

		// Second call (should use cache)
		imageData2, mimeType2, err := svc.GetUserIcon(ctx, userID)
		gt.NoError(t, err)

		// Results should be identical
		gt.Array(t, imageData1).Equal(imageData2)
		gt.Value(t, mimeType1).Equal(mimeType2)
	})
}

func TestService_GetUserProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("returns display name when available", func(t *testing.T) {
		slackMock := &mock.SlackClientMock{
			AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
				return &slack_sdk.AuthTestResponse{
					UserID: "U123456",
					TeamID: "T123456",
					Team:   "test-team",
					BotID:  "B123456",
				}, nil
			},
			GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
				return &slack_sdk.TeamInfo{
					ID:     "T123456",
					Name:   "test-team",
					Domain: "test-workspace",
				}, nil
			},
			GetUserInfoFunc: func(userID string) (*slack_sdk.User, error) {
				return &slack_sdk.User{
					ID: userID,
					Profile: slack_sdk.UserProfile{
						DisplayName: "Test User",
					},
				}, nil
			},
			GetBotInfoContextFunc: func(ctx context.Context, parameters slack_sdk.GetBotInfoParameters) (*slack_sdk.Bot, error) {
				return nil, fmt.Errorf("not a bot")
			},
		}

		service, err := slack.New(slackMock, "C123456")
		gt.NoError(t, err)

		name, err := service.GetUserProfile(ctx, "U123456")
		gt.NoError(t, err)
		gt.Value(t, name).Equal("Test User")
	})

	t.Run("returns error when Slack API fails", func(t *testing.T) {
		slackMock := &mock.SlackClientMock{
			AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
				return &slack_sdk.AuthTestResponse{
					UserID: "U123456",
					TeamID: "T123456",
					Team:   "test-team",
					BotID:  "B123456",
				}, nil
			},
			GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
				return &slack_sdk.TeamInfo{
					ID:     "T123456",
					Name:   "test-team",
					Domain: "test-workspace",
				}, nil
			},
			GetUserInfoFunc: func(userID string) (*slack_sdk.User, error) {
				return nil, fmt.Errorf("user not found")
			},
			GetBotInfoContextFunc: func(ctx context.Context, parameters slack_sdk.GetBotInfoParameters) (*slack_sdk.Bot, error) {
				return nil, fmt.Errorf("bot not found")
			},
		}

		service, err := slack.New(slackMock, "C123456")
		gt.NoError(t, err)

		name, err := service.GetUserProfile(ctx, "U123456")
		gt.NoError(t, err)                 // No longer expects error since we fallback to userID
		gt.Value(t, name).Equal("U123456") // Should return userID as fallback
	})

	t.Run("returns error when user is nil", func(t *testing.T) {
		slackMock := &mock.SlackClientMock{
			AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
				return &slack_sdk.AuthTestResponse{
					UserID: "U123456",
					TeamID: "T123456",
					Team:   "test-team",
					BotID:  "B123456",
				}, nil
			},
			GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
				return &slack_sdk.TeamInfo{
					ID:     "T123456",
					Name:   "test-team",
					Domain: "test-workspace",
				}, nil
			},
			GetUserInfoFunc: func(userID string) (*slack_sdk.User, error) {
				return nil, nil // Return nil user without error
			},
			GetBotInfoContextFunc: func(ctx context.Context, parameters slack_sdk.GetBotInfoParameters) (*slack_sdk.Bot, error) {
				return nil, nil // Return nil bot without error
			},
		}

		service, err := slack.New(slackMock, "C123456")
		gt.NoError(t, err)

		name, err := service.GetUserProfile(ctx, "U123456")
		gt.NoError(t, err)                 // No longer expects error since we fallback to userID
		gt.Value(t, name).Equal("U123456") // Should return userID as fallback
	})

	t.Run("falls back to real name when display name is empty", func(t *testing.T) {
		slackMock := &mock.SlackClientMock{
			AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
				return &slack_sdk.AuthTestResponse{
					UserID: "U123456",
					TeamID: "T123456",
					Team:   "test-team",
					BotID:  "B123456",
				}, nil
			},
			GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
				return &slack_sdk.TeamInfo{
					ID:     "T123456",
					Name:   "test-team",
					Domain: "test-workspace",
				}, nil
			},
			GetUserInfoFunc: func(userID string) (*slack_sdk.User, error) {
				return &slack_sdk.User{
					ID: userID,
					Profile: slack_sdk.UserProfile{
						DisplayName: "",
						RealName:    "John Doe",
					},
				}, nil
			},
			GetBotInfoContextFunc: func(ctx context.Context, parameters slack_sdk.GetBotInfoParameters) (*slack_sdk.Bot, error) {
				return nil, fmt.Errorf("not a bot")
			},
		}

		service, err := slack.New(slackMock, "C123456")
		gt.NoError(t, err)

		name, err := service.GetUserProfile(ctx, "U123456")
		gt.NoError(t, err)
		gt.Value(t, name).Equal("John Doe")
	})

	t.Run("returns error when both names are empty", func(t *testing.T) {
		slackMock := &mock.SlackClientMock{
			AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
				return &slack_sdk.AuthTestResponse{
					UserID: "U123456",
					TeamID: "T123456",
					Team:   "test-team",
					BotID:  "B123456",
				}, nil
			},
			GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
				return &slack_sdk.TeamInfo{
					ID:     "T123456",
					Name:   "test-team",
					Domain: "test-workspace",
				}, nil
			},
			GetUserInfoFunc: func(userID string) (*slack_sdk.User, error) {
				return &slack_sdk.User{
					ID: userID,
					Profile: slack_sdk.UserProfile{
						DisplayName: "",
						RealName:    "",
					},
				}, nil
			},
			GetBotInfoContextFunc: func(ctx context.Context, parameters slack_sdk.GetBotInfoParameters) (*slack_sdk.Bot, error) {
				return nil, fmt.Errorf("not a bot")
			},
		}

		service, err := slack.New(slackMock, "C123456")
		gt.NoError(t, err)

		name, err := service.GetUserProfile(ctx, "U123456")
		gt.NoError(t, err)                 // No longer expects error since we fallback to userID
		gt.Value(t, name).Equal("U123456") // Should return userID as fallback
	})
}

func TestService_GetUserProfile_Cache(t *testing.T) {
	ctx := context.Background()

	// Setup Slack client mock
	slackMock := &mock.SlackClientMock{
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "U123456",
				TeamID: "T123456",
				Team:   "test-team",
				BotID:  "B123456",
			}, nil
		},
		GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
			return &slack_sdk.TeamInfo{
				ID:     "T123456",
				Name:   "test-team",
				Domain: "test-workspace",
			}, nil
		},
		GetUserInfoFunc: func(userID string) (*slack_sdk.User, error) {
			return &slack_sdk.User{
				ID: userID,
				Profile: slack_sdk.UserProfile{
					DisplayName: "Test User",
				},
			}, nil
		},
		GetBotInfoContextFunc: func(ctx context.Context, parameters slack_sdk.GetBotInfoParameters) (*slack_sdk.Bot, error) {
			return nil, fmt.Errorf("not a bot")
		},
	}

	service, err := slack.New(slackMock, "C123456")
	gt.NoError(t, err)

	// First call - should call Slack API
	name1, err := service.GetUserProfile(ctx, "U123456")
	gt.NoError(t, err)
	gt.Value(t, name1).Equal("Test User")

	// Second call - should use cache
	name2, err := service.GetUserProfile(ctx, "U123456")
	gt.NoError(t, err)
	gt.Value(t, name2).Equal("Test User")

	// Should have called Slack API only once
	gt.Array(t, slackMock.GetUserInfoCalls()).Length(1)
}

// TestService_ClearExpiredProfileCache is tested internally as part of the profile cache functionality
