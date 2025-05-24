package command

import (
	"context"
	_ "embed"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

//go:embed help/list.md
var helpListMessage string

type pipeline struct {
	actions []actionFunc
}

func (p *pipeline) Execute(ctx context.Context, alerts alert.Alerts) (alert.Alerts, error) {
	for _, action := range p.actions {
		results, err := action(ctx, alerts)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to run action")
		}
		alerts = results
	}

	return alerts, nil
}

func buildPipeline(commands [][]string) (*pipeline, error) {
	actions := []actionFunc{}
	for _, command := range commands {
		if len(command) == 0 {
			continue
		}

		init, err := findMatchedInitFunc(command[0])
		if err != nil {
			return nil, goerr.Wrap(err, "failed to find matched action")
		}

		actionFunc, err := init(command[1:])
		if err != nil {
			return nil, goerr.Wrap(err, "failed to initialize action")
		}

		actions = append(actions, actionFunc)
	}

	return &pipeline{
		actions: actions,
	}, nil
}

type actionFunc func(ctx context.Context, alerts alert.Alerts) (alert.Alerts, error)
type initFunc func(args []string) (actionFunc, error)

var actionMapping = map[string]initFunc{
	"limit":  actionLimit,
	"offset": actionOffset,
	"grep":   actionGrep,
	"sort":   actionSort,
}

func findMatchedInitFunc(command string) (initFunc, error) {
	var longestMatch string
	var matchedAction initFunc
	command = strings.ToLower(command)

	for actionName, action := range actionMapping {
		actionNameLower := strings.ToLower(actionName)
		if strings.HasPrefix(command, actionNameLower) {
			if len(actionName) > len(longestMatch) {
				longestMatch = actionName
				matchedAction = action
			}
		}
	}

	if matchedAction == nil {
		return nil, goerr.New("unknown action", goerr.V("action", command))
	}

	return matchedAction, nil
}

func actionLimit(args []string) (actionFunc, error) {
	if len(args) != 1 {
		return nil, goerr.New("limit: requires one argument")
	}

	limit, err := strconv.Atoi(args[0])
	if err != nil {
		return nil, goerr.Wrap(err, "limit: failed to convert limit to int")
	}
	return func(ctx context.Context, alerts alert.Alerts) (alert.Alerts, error) {
		if limit <= 0 {
			return alerts, nil
		}
		if limit > len(alerts) {
			return alerts, nil
		}
		return alerts[:limit], nil
	}, nil
}

func actionOffset(args []string) (actionFunc, error) {
	if len(args) != 1 {
		return nil, goerr.New("offset: requires one argument")
	}

	offset, err := strconv.Atoi(args[0])
	if err != nil {
		return nil, goerr.Wrap(err, "offset: failed to convert offset to int")
	}
	return func(ctx context.Context, alerts alert.Alerts) (alert.Alerts, error) {
		if offset <= 0 {
			return alerts, nil
		}
		if offset >= len(alerts) {
			return alert.Alerts{}, nil
		}
		return alerts[offset:], nil
	}, nil
}

func actionGrep(args []string) (actionFunc, error) {
	if len(args) != 1 {
		return nil, goerr.New("grep: requires one argument")
	}

	pattern := args[0]
	return func(ctx context.Context, alerts alert.Alerts) (alert.Alerts, error) {
		if pattern == "" {
			return alerts, nil
		}
		var filtered alert.Alerts
		for _, a := range alerts {
			if strings.Contains(strings.ToLower(a.Metadata.Title), strings.ToLower(pattern)) ||
				strings.Contains(strings.ToLower(a.Metadata.Description), strings.ToLower(pattern)) {
				filtered = append(filtered, a)
				continue
			}
			if a.Data != nil {
				jsonData, err := json.Marshal(a.Data)
				if err == nil && strings.Contains(strings.ToLower(string(jsonData)), strings.ToLower(pattern)) {
					filtered = append(filtered, a)
				}
			}
		}
		return filtered, nil
	}, nil
}

func actionSort(args []string) (actionFunc, error) {
	if len(args) != 1 {
		return nil, goerr.New("sort: requires one argument")
	}

	field := args[0]
	return func(ctx context.Context, alerts alert.Alerts) (alert.Alerts, error) {
		if field == "" {
			return alerts, nil
		}

		sorted := make(alert.Alerts, len(alerts))
		copy(sorted, alerts)

		switch field {
		case "CreatedAt":
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].CreatedAt.Before(sorted[j].CreatedAt)
			})
		default:
			return nil, goerr.New("sort: invalid field", goerr.V("field", field))
		}

		return sorted, nil
	}, nil
}

// CreateList creates a new AlertList from the given alerts and registers it to the repository.
// It also generates meta data (title and description) for the list using LLM if possible.
func (x *Service) CreateList(ctx context.Context, thread slack.Thread, user *slack.User, alerts alert.Alerts) (*alert.List, error) {
	list := alert.NewList(ctx, thread, user, alerts)

	if err := list.FillMetadata(ctx, x.llm); err != nil {
		return nil, goerr.Wrap(err, "failed to fill metadata")
	}

	// Register the list to the repository
	if err := x.repo.PutAlertList(ctx, list); err != nil {
		return nil, goerr.Wrap(err, "failed to put alert list")
	}

	return list, nil
}

// RunList runs the list command with the given input.
func (x *Service) RunList(ctx context.Context, th *svc.ThreadService, user *slack.User, input string) (types.AlertListID, error) {
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
		msg.Notify(ctx, "%s", helpListMessage)
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

	alertList, err := x.CreateList(ctx, slack.Thread{
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
