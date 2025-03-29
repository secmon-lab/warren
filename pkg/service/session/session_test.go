package session_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/adapter/gemini"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	model "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/session"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	slack_sdk "github.com/slack-go/slack"
)

func TestSessionChat(t *testing.T) {
	ctx := t.Context()
	ssn := model.New(ctx, &slack.User{}, &slack.Thread{}, []types.AlertID{types.NewAlertID()})
	geminiClient := gemini.NewTestClient(t)
	slackClient := &interfaces.SlackClientMock{
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "test-user",
				TeamID: "test-team",
				Team:   "test-team",
			}, nil
		},
	}
	slackService, err := slack_svc.New(slackClient, "test-channel")
	gt.NoError(t, err)

	svc := session.New(repository.NewMemory(), geminiClient, slackService, ssn)

	err = svc.Chat(ctx, "Hello, world!")
	gt.NoError(t, err)
}
