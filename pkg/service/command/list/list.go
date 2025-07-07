package list

import (
	"context"
	_ "embed"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/command/core"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

//go:embed help.md
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

func buildPipeline(commands []string) (*pipeline, error) {
	actions := []actionFunc{}
	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}

		parts := strings.SplitN(command, " ", 2)
		cmdName := parts[0]
		var args string
		if len(parts) > 1 {
			args = parts[1]
		}

		init, err := FindMatchedInitFunc(cmdName)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to find matched action")
		}

		actionFunc, err := init(args)
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
type initFunc func(args string) (actionFunc, error)

var actionMapping = map[string]initFunc{
	"limit":  actionLimit,
	"offset": actionOffset,
	"grep":   actionGrep,
	"sort":   actionSort,
	"from":   actionFrom,
	"to":     actionTo,
	"after":  actionAfter,
	"since":  actionSince,
	"all":    actionAll,
	"help":   actionHelp,
	"h":      actionHelp,
}

// FindMatchedInitFunc finds the matched init function for the given command
func FindMatchedInitFunc(command string) (initFunc, error) {
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

func actionLimit(args string) (actionFunc, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return nil, goerr.New("limit: requires one argument")
	}

	limit, err := strconv.Atoi(args)
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

func actionOffset(args string) (actionFunc, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return nil, goerr.New("offset: requires one argument")
	}

	offset, err := strconv.Atoi(args)
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

func actionGrep(args string) (actionFunc, error) {
	pattern := strings.TrimSpace(args)
	if pattern == "" {
		return nil, goerr.New("grep: requires one argument")
	}

	return func(ctx context.Context, alerts alert.Alerts) (alert.Alerts, error) {
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

func actionSort(args string) (actionFunc, error) {
	field := strings.TrimSpace(args)
	if field == "" {
		return nil, goerr.New("sort: requires one argument")
	}

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

func actionFrom(args string) (actionFunc, error) {
	parts := strings.Fields(args)
	if len(parts) != 3 || parts[1] != "to" {
		return nil, goerr.New("from: requires 'from <time> to <time>' format")
	}

	return func(ctx context.Context, alerts alert.Alerts) (alert.Alerts, error) {
		fromTime, err := ParseTime(ctx, parts[0])
		if err != nil {
			return nil, goerr.Wrap(err, "from: failed to parse from time")
		}

		toTime, err := ParseTime(ctx, parts[2])
		if err != nil {
			return nil, goerr.Wrap(err, "from: failed to parse to time")
		}

		var filtered alert.Alerts
		for _, a := range alerts {
			if (a.CreatedAt.After(fromTime) || a.CreatedAt.Equal(fromTime)) &&
				(a.CreatedAt.Before(toTime) || a.CreatedAt.Equal(toTime)) {
				filtered = append(filtered, a)
			}
		}
		return filtered, nil
	}, nil
}

func actionTo(args string) (actionFunc, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return nil, goerr.New("to: requires one argument")
	}

	return func(ctx context.Context, alerts alert.Alerts) (alert.Alerts, error) {
		toTime, err := ParseTime(ctx, args)
		if err != nil {
			return nil, goerr.Wrap(err, "to: failed to parse time")
		}

		var filtered alert.Alerts
		for _, a := range alerts {
			if a.CreatedAt.Before(toTime) || a.CreatedAt.Equal(toTime) {
				filtered = append(filtered, a)
			}
		}
		return filtered, nil
	}, nil
}

func actionAfter(args string) (actionFunc, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return nil, goerr.New("after: requires one argument")
	}

	return func(ctx context.Context, alerts alert.Alerts) (alert.Alerts, error) {
		afterTime, err := ParseTime(ctx, args)
		if err != nil {
			return nil, goerr.Wrap(err, "after: failed to parse time")
		}

		var filtered alert.Alerts
		for _, a := range alerts {
			if a.CreatedAt.After(afterTime) {
				filtered = append(filtered, a)
			}
		}
		return filtered, nil
	}, nil
}

func actionSince(args string) (actionFunc, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return nil, goerr.New("since: requires one argument")
	}

	duration, err := ParseDuration(args)
	if err != nil {
		return nil, goerr.Wrap(err, "since: failed to parse duration")
	}

	return func(ctx context.Context, alerts alert.Alerts) (alert.Alerts, error) {
		sinceTime := clock.Now(ctx).Add(-duration)
		var filtered alert.Alerts
		for _, a := range alerts {
			if a.CreatedAt.After(sinceTime) || a.CreatedAt.Equal(sinceTime) {
				filtered = append(filtered, a)
			}
		}
		return filtered, nil
	}, nil
}

func actionAll(args string) (actionFunc, error) {
	args = strings.TrimSpace(args)
	if args != "" {
		return nil, goerr.New("all: takes no arguments")
	}

	return func(ctx context.Context, alerts alert.Alerts) (alert.Alerts, error) {
		return alerts, nil
	}, nil
}

func actionHelp(args string) (actionFunc, error) {
	args = strings.TrimSpace(args)
	if args != "" {
		return nil, goerr.New("help: takes no arguments")
	}

	return func(ctx context.Context, alerts alert.Alerts) (alert.Alerts, error) {
		return alert.Alerts{}, nil
	}, nil
}

// ParseTime parses a time string in either HH:MM or YYYY-MM-DD format
func ParseTime(ctx context.Context, timeStr string) (time.Time, error) {
	// Try parsing as time format (HH:MM)
	if t, err := time.Parse("15:04", timeStr); err == nil {
		now := clock.Now(ctx)
		return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location()), nil
	}

	// Try parsing as date format (YYYY-MM-DD)
	if t, err := time.Parse("2006-01-02", timeStr); err == nil {
		return t, nil
	}

	return time.Time{}, goerr.New("invalid time format", goerr.V("time", timeStr))
}

// ParseDuration parses a duration string in the format of "10m", "1h", "1d"
func ParseDuration(durationStr string) (time.Duration, error) {
	// Parse duration like "10m", "1h", "1d"
	unit := durationStr[len(durationStr)-1:]
	value, err := strconv.Atoi(durationStr[:len(durationStr)-1])
	if err != nil {
		return 0, goerr.Wrap(err, "failed to parse duration value")
	}

	switch unit {
	case "m":
		return time.Duration(value) * time.Minute, nil
	case "h":
		return time.Duration(value) * time.Hour, nil
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	default:
		return 0, goerr.New("invalid duration unit", goerr.V("unit", unit))
	}
}

func Create(ctx context.Context, clients *core.Clients, slackMsg *slack.Message, input string) (any, error) {
	th := clients.Thread()

	commands := strings.Split(input, "|")
	pipelineCommands := []string{}
	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		pipelineCommands = append(pipelineCommands, command)
	}

	for _, cmd := range pipelineCommands {
		if cmd != "" {
			parts := strings.SplitN(cmd, " ", 2)
			if len(parts) > 0 && strings.ToLower(parts[0]) == "help" {
				th.Reply(ctx, helpListMessage)
				return types.EmptyAlertListID, nil
			}
		}
	}

	var pipeline *pipeline
	var err error
	if len(pipelineCommands) > 0 {
		pipeline, err = buildPipeline(pipelineCommands)
		if err != nil {
			_ = msg.Trace(ctx, "💥 Building pipeline: %s", err)
			return types.EmptyAlertListID, err
		}
	} else {
		// Default to showing all alerts
		pipeline, err = buildPipeline([]string{"all"})
		if err != nil {
			_ = msg.Trace(ctx, "💥 Building default pipeline: %s", err)
			return types.EmptyAlertListID, err
		}
	}

	msg.Notify(ctx, "🤖 Getting and filtering alerts without ticket...")

	alerts, err := clients.Repo().GetAlertWithoutTicket(ctx, 0, 0)
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

	if len(alerts) == 0 {
		msg.Trace(ctx, "No alerts found")
		return types.EmptyAlertListID, nil
	}

	alertList, err := clients.CreateList(ctx, slack.Thread{
		ChannelID: th.ChannelID(),
		ThreadID:  th.ThreadID(),
	}, slackMsg.User(), alerts)
	if err != nil {
		msg.Trace(ctx, "💥 Create alert list: %s", err)
		return types.EmptyAlertListID, err
	}

	slackMessageID, err := th.PostAlertList(ctx, alertList)
	if err != nil {
		msg.Trace(ctx, "💥 Post alert list: %s", err)
		return types.EmptyAlertListID, err
	}

	// Save SlackMessageID to the alert list
	alertList.SlackMessageID = slackMessageID
	if err := clients.Repo().PutAlertList(ctx, alertList); err != nil {
		msg.Trace(ctx, "💥 Update alert list: %s", err)
		return types.EmptyAlertListID, err
	}

	return alertList.ID, nil
}
