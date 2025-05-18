package list

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
)

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
		var filtered alert.Alerts
		for _, a := range alerts {
			raw, err := json.Marshal(a.Data)
			if err != nil {
				return nil, goerr.Wrap(err, "grep: failed to marshal alert data")
			}
			if strings.Contains(string(raw), pattern) {
				filtered = append(filtered, a)
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
