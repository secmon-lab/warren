package list

import (
	"context"
	"encoding/json"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/prompt"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/utils/thread"
)

type actionMapping map[string]func([]string) action

func newActionMapping(x *Service) actionMapping {
	return actionMapping{
		"grep":   x.actionGrep,
		"sort":   x.actionSort,
		"limit":  x.actionLimit,
		"offset": x.actionOffset,
		"query":  x.actionQuery,
		"user":   x.actionUser,
		"status": x.actionStatus,
	}
}

func (x actionMapping) findMatchedAction(name string) (func([]string) action, error) {
	var matchedFn func([]string) action
	var longestMatch string
	for n, fn := range x {
		if strings.HasPrefix(n, name) {
			if len(n) > len(longestMatch) {
				matchedFn = fn
				longestMatch = n
			}
		}
	}
	if matchedFn == nil {
		return nil, goerr.New("unknown command: "+name, goerr.V("command", name))
	}
	return matchedFn, nil
}

type action func(ctx context.Context, alerts []alert.Alert) ([]alert.Alert, error)

func (x *Service) actionGrep(args []string) action {
	return func(ctx context.Context, alerts []alert.Alert) ([]alert.Alert, error) {
		filtered := []alert.Alert{}

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
	var comp func(a, b alert.Alert) bool

	if len(args) != 1 {
		return func(ctx context.Context, alerts []alert.Alert) ([]alert.Alert, error) {
			return nil, goerr.New("invalid sort key. (valid keys: created_at, updated_at)", goerr.V("key", args))
		}
	}

	key := args[0]
	switch key {
	case "created_at":
		comp = func(a, b alert.Alert) bool {
			return a.CreatedAt.Before(b.CreatedAt)
		}
	case "updated_at":
		comp = func(a, b alert.Alert) bool {
			return a.UpdatedAt.Before(b.UpdatedAt)
		}
	default:
		return func(ctx context.Context, alerts []alert.Alert) ([]alert.Alert, error) {
			return nil, goerr.New("invalid sort key. (valid keys: created_at, updated_at)", goerr.V("key", key))
		}
	}

	return func(ctx context.Context, alerts []alert.Alert) ([]alert.Alert, error) {
		sort.Slice(alerts, func(i, j int) bool {
			return comp(alerts[i], alerts[j])
		})
		return alerts, nil
	}
}

func (x *Service) actionLimit(args []string) action {
	if len(args) != 1 {
		return func(ctx context.Context, alerts []alert.Alert) ([]alert.Alert, error) {
			return nil, goerr.New("argument required. (valid arguments: limit [number])", goerr.V("args", args))
		}
	}

	limit, err := strconv.Atoi(args[0])
	if err != nil {
		return func(ctx context.Context, alerts []alert.Alert) ([]alert.Alert, error) {
			return nil, goerr.Wrap(err, "failed to convert limit to int", goerr.V("limit", args[0]))
		}
	}

	return func(ctx context.Context, alerts []alert.Alert) ([]alert.Alert, error) {
		if limit > len(alerts) {
			return alerts, nil
		}
		return alerts[:limit], nil
	}
}

func (x *Service) actionOffset(args []string) action {
	if len(args) != 1 {
		return func(ctx context.Context, alerts []alert.Alert) ([]alert.Alert, error) {
			return nil, goerr.New("argument required. (valid arguments: offset [number])", goerr.V("args", args))
		}
	}

	offset, err := strconv.Atoi(args[0])
	if err != nil {
		return func(ctx context.Context, alerts []alert.Alert) ([]alert.Alert, error) {
			return nil, goerr.Wrap(err, "failed to convert offset to int", goerr.V("offset", args[0]))
		}
	}

	return func(ctx context.Context, alerts []alert.Alert) ([]alert.Alert, error) {
		if offset > len(alerts) {
			return []alert.Alert{}, nil
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
	return func(ctx context.Context, alerts []alert.Alert) ([]alert.Alert, error) {
		if len(args) != 1 {
			return nil, goerr.New("argument required. (valid arguments: user [user_id])")
		}

		userID, err := extractUserID(args[0])
		if err != nil {
			return nil, err
		}

		filtered := []alert.Alert{}
		for _, alert := range alerts {
			if alert.Assignee.ID == userID {
				filtered = append(filtered, alert)
			}
		}
		return filtered, nil
	}
}

func (x *Service) actionStatus(args []string) action {
	return func(ctx context.Context, alerts []alert.Alert) ([]alert.Alert, error) {
		if len(args) == 0 {
			return nil, goerr.New("argument required. (valid arguments: status [status])")
		}

		statusList := []alert.Status{}

		for _, arg := range args {
			switch arg {
			case "new":
				statusList = append(statusList, alert.StatusNew)
			case "ack", "acked", "acknowledged":
				statusList = append(statusList, alert.StatusAcknowledged)
			case "blocked":
				statusList = append(statusList, alert.StatusBlocked)
			case "resolved":
				statusList = append(statusList, alert.StatusResolved)
			default:
				return nil, goerr.New("invalid status, status must be one of new, ack, blocked, resolved", goerr.V("status", arg))
			}
		}

		filtered := []alert.Alert{}
		for _, alert := range alerts {
			if slices.Contains(statusList, alert.Status) {
				filtered = append(filtered, alert)
			}
		}

		return filtered, nil
	}
}

func (x *Service) actionQuery(args []string) action {
	return func(ctx context.Context, alerts []alert.Alert) ([]alert.Alert, error) {
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
				if goerr.HasTag(err, errs.TagInvalidLLMResponse) {
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
