package list

import (
	"context"
	_ "embed"
	"strings"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/source"
	"github.com/secmon-lab/warren/pkg/domain/types"
	svc "github.com/secmon-lab/warren/pkg/service/slack"

	"github.com/secmon-lab/warren/pkg/utils/msg"
)

type Service struct {
	repo interfaces.Repository
	llm  gollem.LLMClient
}

func New(repo interfaces.Repository, llm gollem.LLMClient) *Service {
	return &Service{
		repo: repo,
		llm:  llm,
	}
}

//go:embed embed/help.md
var helpMessage string

func showHelp(ctx context.Context) {
	msg.Notify(ctx, "%s", helpMessage)
}

func filterEmptyStrings(s []string) []string {
	var result []string
	for _, str := range s {
		if str != "" {
			result = append(result, strings.TrimSpace(str))
		}
	}
	return result
}

func (x *Service) Run(ctx context.Context, th *svc.ThreadService, user *slack.User, input string) (types.AlertListID, error) {
	commands := strings.Split(input, "|")
	if len(commands) == 0 {
		showHelp(ctx)
		return types.EmptyAlertListID, nil
	}

	ctx = msg.NewTrace(ctx, "🤖 Creating alert list...")

	nextCommands := commands[1:]
	var pipeline *pipeline
	var err error
	if len(nextCommands) > 0 {
		pipelineCommands := [][]string{}
		for _, command := range nextCommands {
			command = strings.TrimSpace(command)
			if command == "" {
				continue
			}
			pipelineCommands = append(pipelineCommands, strings.Split(command, " "))
		}

		pipeline, err = buildPipeline(pipelineCommands)
		if err != nil {
			msg.Trace(ctx, "💥 Building pipeline: %s", err)
			return types.EmptyAlertListID, err
		}
	}

	alerts, err := source.Unbound()(ctx, x.repo)
	if err != nil {
		msg.Trace(ctx, "💥 Get alerts without ticket: %s", err)
		return types.EmptyAlertListID, err
	}

	if alerts == nil {
		alerts = alert.Alerts{}
	}

	if pipeline != nil {
		alerts, err = pipeline.Execute(ctx, alerts)
		if err != nil {
			msg.Trace(ctx, "💥 Execute pipeline: %s", err)
			return types.EmptyAlertListID, err
		}
	}

	alertList, err := CreateList(ctx, x.repo, x.llm, slack.Thread{
		ChannelID: th.ChannelID(),
		ThreadID:  th.ThreadID(),
	}, user, alerts)
	if err != nil {
		msg.Trace(ctx, "💥 Create alert list: %s", err)
		return types.EmptyAlertListID, err
	}

	if err := th.PostAlertList(ctx, alertList); err != nil {
		msg.Trace(ctx, "💥 Post alert list: %s", err)
		return types.EmptyAlertListID, err
	}

	return alertList.ID, nil
}
