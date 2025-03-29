package list

import (
	"context"
	_ "embed"
	"strings"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	svc "github.com/secmon-lab/warren/pkg/service/slack"

	"github.com/secmon-lab/warren/pkg/utils/msg"
)

type Service struct {
	repo interfaces.Repository
}

func New(repo interfaces.Repository) *Service {
	return &Service{
		repo: repo,
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
			result = append(result, str)
		}
	}
	return result
}

func (x *Service) Run(ctx context.Context, repo interfaces.Repository, th *svc.ThreadService, user *slack.User, input string) (types.AlertListID, error) {
	commands := strings.Split(input, "|")
	if len(commands) == 0 {
		showHelp(ctx)
		return types.EmptyAlertListID, nil
	}

	ctx = msg.NewTrace(ctx, "🤖 Creating alert list...")

	args := strings.Split(commands[0], " ")
	args = filterEmptyStrings(args)
	src, err := parseArgsToSource(ctx, args)
	if err != nil {
		msg.Trace(ctx, "💥 Error: %s", err)
		showHelp(ctx)
		return types.EmptyAlertListID, err
	}

	nextCommands := commands[1:]
	var pipeline *pipeline
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

	alerts, err := src(ctx, x.repo)
	if err != nil {
		msg.Trace(ctx, "💥 Get alerts: %s", err)
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

	alertList := alert.NewList(ctx, slack.Thread{
		ChannelID: th.ChannelID(),
		ThreadID:  th.ThreadID(),
	}, user, alerts)

	if err := x.repo.PutAlertList(ctx, alertList); err != nil {
		msg.Trace(ctx, "💥 Create alert list: %s", err)
		return types.EmptyAlertListID, err
	}

	if err := th.PostAlertList(ctx, &alertList); err != nil {
		msg.Trace(ctx, "💥 Post alert list: %s", err)
		return types.EmptyAlertListID, err
	}

	return alertList.ID, nil
}
