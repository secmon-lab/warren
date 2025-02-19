package usecase_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/mock"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/slack-go/slack"
)

func TestHandleSlackInteraction(t *testing.T) {
	testCases := map[string]struct {
		interaction slack.InteractionCallback
		checkAlert  func(t *testing.T, alert model.Alert)
		wantErr     bool
	}{
		"close": {
			interaction: slack.InteractionCallback{
				Type: slack.InteractionTypeBlockActions,
				ActionCallback: slack.ActionCallbacks{
					BlockActions: []*slack.BlockAction{
						{
							ActionID: "close",
						},
					},
				},
			},
			checkAlert: func(t *testing.T, alert model.Alert) {
				gt.Equal(t, alert.Status, model.AlertStatusNew) // still new
				gt.Equal(t, alert.Conclusion, "")               // not set yet
				gt.Equal(t, alert.Comment, "")                  // not set yet
			},
			wantErr: false,
		},
		"ack": {
			interaction: slack.InteractionCallback{
				Type: slack.InteractionTypeBlockActions,
				User: slack.User{
					ID:   "test-user-id",
					Name: "test-user-name",
				},
				ActionCallback: slack.ActionCallbacks{
					BlockActions: []*slack.BlockAction{
						{
							ActionID: "ack",
						},
					},
				},
			},
			checkAlert: func(t *testing.T, alert model.Alert) {
				gt.Equal(t, alert.Status, model.AlertStatusAcknowledged)
				gt.Equal(t, alert.Assignee.ID, "test-user-id")
				gt.Equal(t, alert.Assignee.Name, "test-user-name")
			},
			wantErr: false,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			alert := model.NewAlert(ctx, "test-alert-id", model.PolicyAlert{
				Title:       "test-alert-title",
				Description: "test-alert-description",
				Data:        map[string]interface{}{},
			})

			repo := repository.NewMemory()
			gt.NoError(t, repo.PutAlert(ctx, alert)).Must()
			testCase.interaction.ActionCallback.BlockActions[0].Value = alert.ID.String()

			slackMock := &mock.SlackServiceMock{
				NewThreadFunc: func(alert model.Alert) interfaces.SlackThreadService {
					return &mock.SlackThreadServiceMock{
						ReplyFunc: func(ctx context.Context, message string) {
							// do nothing
						},
						UpdateAlertFunc: func(ctx context.Context, alert model.Alert) error {
							return nil
						},
					}
				},
				ShowCloseAlertModalFunc: func(ctx context.Context, alert model.Alert, triggerID string) error {
					return nil
				},
			}

			uc := usecase.New(nil, usecase.WithRepository(repo), usecase.WithSlackService(slackMock))
			err := uc.HandleSlackInteraction(ctx, testCase.interaction)

			if testCase.wantErr {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err).Must()
				alert, err := repo.GetAlert(ctx, alert.ID)
				gt.NoError(t, err).Must()
				testCase.checkAlert(t, *alert)
			}
		})
	}
}
