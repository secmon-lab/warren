package slack_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"

	slackagent "github.com/secmon-lab/warren/pkg/agents/slack"
	domainmock "github.com/secmon-lab/warren/pkg/domain/mock"
)

func TestToolSet_ID(t *testing.T) {
	slackClient := &domainmock.SlackClientMock{}
	ts := slackagent.NewToolSetForTest(slackClient)

	gt.V(t, ts.ID()).Equal("slack_agent")
}

func TestToolSet_Description(t *testing.T) {
	slackClient := &domainmock.SlackClientMock{}
	ts := slackagent.NewToolSetForTest(slackClient)

	description := ts.Description()
	gt.V(t, description).NotEqual("")
	gt.True(t, len(description) > 0)
}

func TestToolSet_Prompt(t *testing.T) {
	slackClient := &domainmock.SlackClientMock{}
	ts := slackagent.NewToolSetForTest(slackClient)

	ctx := context.Background()
	prompt, err := ts.Prompt(ctx)
	gt.NoError(t, err)
	gt.V(t, prompt).NotEqual("")
}

func TestToolSet_Specs(t *testing.T) {
	slackClient := &domainmock.SlackClientMock{}
	ts := slackagent.NewToolSetForTest(slackClient)

	ctx := context.Background()
	specs, err := ts.Specs(ctx)
	gt.NoError(t, err)
	gt.N(t, len(specs)).Equal(3) // slack_search_messages, slack_get_thread_messages, slack_get_context_messages
}
