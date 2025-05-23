package list

import (
	"context"
	_ "embed"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
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

func (x *Service) Run(ctx context.Context, th *svc.ThreadService, user *slack.User, input string) (types.AlertListID, error) {
	commands := strings.Split(input, "|")
	pipelineCommands := [][]string{}
	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		parts := strings.Split(command, " ")
		if len(parts) == 0 {
			continue
		}
		pipelineCommands = append(pipelineCommands, parts)
	}

	var pipeline *pipeline
	var err error
	if len(pipelineCommands) > 0 {
		pipeline, err = buildPipeline(pipelineCommands)
		if err != nil {
			msg.Trace(ctx, "💥 Building pipeline: %s", err)
			return types.EmptyAlertListID, err
		}
	} else {
		showHelp(ctx)
		return types.EmptyAlertListID, nil
	}

	msg.Trace(ctx, "🤖 Getting unbound alerts with ticket...")

	alerts, err := x.repo.GetAlertWithoutTicket(ctx)
	if err != nil {
		msg.Trace(ctx, "💥 Get alerts without ticket: %s", err)
		return types.EmptyAlertListID, goerr.Wrap(err, "failed to get unbound alerts")
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
