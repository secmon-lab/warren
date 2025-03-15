package list

import (
	"context"

	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/prompt"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/service/source"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/thread"
)

type Service struct {
	repo interfaces.Repository
	llm  interfaces.LLMClient
}

type Option func(*Service)

func WithLLM(llm interfaces.LLMClient) Option {
	return func(s *Service) {
		s.llm = llm
	}
}

func New(repo interfaces.Repository, opts ...Option) *Service {
	s := &Service{
		repo: repo,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (x *Service) Run(ctx context.Context, th interfaces.SlackThreadService, user *model.SlackUser, src source.Source, args []string) error {
	alerts, err := src(ctx, x.repo)
	if err != nil {
		return err
	}

	pipe, err := x.newPipeline(ctx, args)
	if err != nil {
		return err
	}

	newAlerts, err := pipe.Run(ctx, alerts)
	if err != nil {
		return err
	}

	slackThread := model.SlackThread{
		ChannelID: th.ChannelID(),
		ThreadID:  th.ThreadID(),
	}
	alertList := model.NewAlertList(ctx, slackThread, user, newAlerts)

	p, err := prompt.BuildMetaListPrompt(ctx, alertList)
	if err != nil {
		return err
	}

	thread.Reply(ctx, "🤖 Generating meta data of alert list...")
	resp, err := service.AskPrompt[prompt.MetaListPromptResult](ctx, x.llm, p)
	if err != nil {
		thread.Reply(ctx, "💥 Failed to generate meta data of alert list.")
	} else {
		alertList.Title = resp.Title
		alertList.Description = resp.Description
	}

	if err := x.repo.PutAlertList(ctx, alertList); err != nil {
		return err
	}

	if err := th.PostAlertList(ctx, &alertList); err != nil {
		return err
	}

	return nil
}

type pipeline struct {
	actions []action
}

func (p *pipeline) Run(ctx context.Context, alerts []model.Alert) ([]model.Alert, error) {
	for _, action := range p.actions {
		newAlerts, err := action(ctx, alerts)
		if err != nil {
			return nil, err
		}
		alerts = newAlerts
	}
	return alerts, nil
}

func (x *Service) newPipeline(ctx context.Context, args []string) (*pipeline, error) {
	logger := logging.From(ctx)

	// Arguments Example:
	// `| filter | sort CreatedAt | limit 10 | offset 10`

	var actions []action
	var currentName string
	var currentArgs []string

	actionMap := newActionMapping(x)

	// Find action with prefix match and return matched function

	// Process current command and append action
	processCommand := func() error {
		if currentName != "" {
			matchedFn, err := actionMap.findMatchedAction(currentName)
			if err != nil {
				return err
			}
			logger.Debug("Matched action", "action", currentName, "args", currentArgs)
			actions = append(actions, matchedFn(currentArgs))
			currentName = ""
			currentArgs = nil
		}
		return nil
	}

	for _, arg := range args {
		if arg == "|" {
			if err := processCommand(); err != nil {
				return nil, err
			}
			continue
		}

		if currentName == "" {
			currentName = arg
		} else {
			currentArgs = append(currentArgs, arg)
		}
	}

	// Process last command
	if err := processCommand(); err != nil {
		return nil, err
	}

	return &pipeline{actions: actions}, nil
}
