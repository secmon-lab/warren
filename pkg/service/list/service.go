package list

import (
	"context"
	"encoding/json"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/m-mizutani/goerr/v2"
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

type action func(ctx context.Context, alerts []model.Alert) ([]model.Alert, error)

func (x *Service) actionGrep(args []string) action {
	return func(ctx context.Context, alerts []model.Alert) ([]model.Alert, error) {
		filtered := []model.Alert{}

		for _, alert := range alerts {
			raw, err := json.Marshal(alert)
			if err != nil {
				return nil, goerr.Wrap(err, "failed to marshal alerts", goerr.V("alert", alert))
			}

			for _, keyword := range args {
				if strings.Contains(string(raw), keyword) {
					filtered = append(filtered, alert)
				}
			}
		}

		return filtered, nil
	}
}

func (x *Service) actionSort(args []string) action {
	var comp func(a, b model.Alert) bool

	if len(args) != 1 {
		return func(ctx context.Context, alerts []model.Alert) ([]model.Alert, error) {
			return nil, goerr.New("invalid sort key. (valid keys: created_at, updated_at)", goerr.V("key", args))
		}
	}

	key := args[0]
	switch key {
	case "created_at":
		comp = func(a, b model.Alert) bool {
			return a.CreatedAt.Before(b.CreatedAt)
		}
	case "updated_at":
		comp = func(a, b model.Alert) bool {
			return a.UpdatedAt.Before(b.UpdatedAt)
		}
	default:
		return func(ctx context.Context, alerts []model.Alert) ([]model.Alert, error) {
			return nil, goerr.New("invalid sort key. (valid keys: created_at, updated_at)", goerr.V("key", key))
		}
	}

	return func(ctx context.Context, alerts []model.Alert) ([]model.Alert, error) {
		sort.Slice(alerts, func(i, j int) bool {
			return comp(alerts[i], alerts[j])
		})
		return alerts, nil
	}
}

func (x *Service) actionLimit(args []string) action {
	if len(args) != 1 {
		return func(ctx context.Context, alerts []model.Alert) ([]model.Alert, error) {
			return nil, goerr.New("argument required. (valid arguments: limit [number])", goerr.V("args", args))
		}
	}

	limit, err := strconv.Atoi(args[0])
	if err != nil {
		return func(ctx context.Context, alerts []model.Alert) ([]model.Alert, error) {
			return nil, goerr.Wrap(err, "failed to convert limit to int", goerr.V("limit", args[0]))
		}
	}

	return func(ctx context.Context, alerts []model.Alert) ([]model.Alert, error) {
		if limit > len(alerts) {
			return alerts, nil
		}
		return alerts[:limit], nil
	}
}

func (x *Service) actionOffset(args []string) action {
	if len(args) != 1 {
		return func(ctx context.Context, alerts []model.Alert) ([]model.Alert, error) {
			return nil, goerr.New("argument required. (valid arguments: offset [number])", goerr.V("args", args))
		}
	}

	offset, err := strconv.Atoi(args[0])
	if err != nil {
		return func(ctx context.Context, alerts []model.Alert) ([]model.Alert, error) {
			return nil, goerr.Wrap(err, "failed to convert offset to int", goerr.V("offset", args[0]))
		}
	}

	return func(ctx context.Context, alerts []model.Alert) ([]model.Alert, error) {
		if offset > len(alerts) {
			return []model.Alert{}, nil
		}
		return alerts[offset:], nil
	}
}

func extractUserID(arg string) (string, error) {
	if !strings.HasPrefix(arg, "<@") || !strings.HasSuffix(arg, ">") {
		return "", goerr.New("invalid user id", goerr.V("arg", arg))
	}

	userID := strings.TrimPrefix(arg, "<@")
	userID = strings.TrimSuffix(userID, ">")
	if userID == "" {
		return "", goerr.New("invalid user id", goerr.V("arg", arg))
	}

	return userID, nil
}

func (x *Service) actionUser(args []string) action {
	return func(ctx context.Context, alerts []model.Alert) ([]model.Alert, error) {
		if len(args) != 1 {
			return nil, goerr.New("argument required. (valid arguments: user [user_id])")
		}

		userID, err := extractUserID(args[0])
		if err != nil {
			return nil, err
		}

		filtered := []model.Alert{}
		for _, alert := range alerts {
			if alert.Assignee.ID == userID {
				filtered = append(filtered, alert)
			}
		}
		return filtered, nil
	}
}

func (x *Service) actionStatus(args []string) action {
	return func(ctx context.Context, alerts []model.Alert) ([]model.Alert, error) {
		if len(args) == 0 {
			return nil, goerr.New("argument required. (valid arguments: status [status])")
		}

		statusList := []model.AlertStatus{}

		for _, arg := range args {
			switch arg {
			case "new":
				statusList = append(statusList, model.AlertStatusNew)
			case "ack", "acked", "acknowledged":
				statusList = append(statusList, model.AlertStatusAcknowledged)
			case "closed":
				statusList = append(statusList, model.AlertStatusClosed)
			case "merged":
				statusList = append(statusList, model.AlertStatusMerged)
			default:
				return nil, goerr.New("invalid status, status must be one of new, ack, closed, merged", goerr.V("status", arg))
			}
		}

		filtered := []model.Alert{}
		for _, alert := range alerts {
			if slices.Contains(statusList, alert.Status) {
				filtered = append(filtered, alert)
			}
		}

		return filtered, nil
	}
}

func (x *Service) actionQuery(args []string) action {
	return func(ctx context.Context, alerts []model.Alert) ([]model.Alert, error) {
		if len(args) == 0 {
			return nil, goerr.New("argument required. (valid arguments: query [string...])")
		}

		query := strings.Join(args, " ")
		p, err := prompt.BuildFilterQueryPrompt(ctx, query, alerts)
		if err != nil {
			return nil, err
		}

		const maxRetry = 3
		for range maxRetry {
			response, err := service.AskChat[prompt.FilterQueryPromptResult](ctx, x.llm, p)
			if err != nil {
				if goerr.HasTag(err, model.ErrTagInvalidLLMResponse) {
					p = "Invalid response. Please try again: " + err.Error()
					thread.Reply(ctx, "🔄 Invalid response. Retry...\n> "+err.Error())
					continue
				}
				return nil, err
			}

			alertIDs := response.AlertIDs
			alerts, err := x.repo.BatchGetAlerts(ctx, alertIDs)
			if err != nil {
				p = "Failed to get alerts that you specified. Please try again: " + err.Error()
				thread.Reply(ctx, "🔄 Failed to get alerts. Retry...\n> "+err.Error())
				continue
			}
			return alerts, nil
		}

		return nil, goerr.New("failed to get alerts that you specified.", goerr.V("query", query))
	}
}

func (x *Service) newPipeline(ctx context.Context, args []string) (*pipeline, error) {
	logger := logging.From(ctx)

	// Arguments Example:
	// `| filter | sort CreatedAt | limit 10 | offset 10`

	actionMap := map[string]func([]string) action{
		"grep":   x.actionGrep,
		"sort":   x.actionSort,
		"limit":  x.actionLimit,
		"offset": x.actionOffset,
		"query":  x.actionQuery,
		"user":   x.actionUser,
		"status": x.actionStatus,
	}

	var actions []action
	var currentName string
	var currentArgs []string

	// Find action with prefix match and return matched function
	findMatchedAction := func(name string, actionMap map[string]func([]string) action) (func([]string) action, error) {
		var matchedFn func([]string) action
		var longestMatch string
		for n, fn := range actionMap {
			if strings.HasPrefix(n, name) {
				if len(n) > len(longestMatch) {
					matchedFn = fn
					longestMatch = n
				}
			}
		}
		if matchedFn == nil {
			return nil, goerr.New("unknown command", goerr.V("command", name))
		}
		return matchedFn, nil
	}

	// Process current command and append action
	processCommand := func() error {
		if currentName != "" {
			matchedFn, err := findMatchedAction(currentName, actionMap)
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
