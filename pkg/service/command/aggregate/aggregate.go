package aggregate

import (
	"context"
	_ "embed"
	"strconv"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/service/command/core"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

//go:embed help.md
var aggregateHelp string

func showAggregateHelp(ctx context.Context) {
	msg.Notify(ctx, "%s", aggregateHelp)
}

func parseArgs(ctx context.Context, args []string) (threshold float64, topN int, retErr error) {
	threshold = 0.99
	topN = 5

	defer func() {
		if retErr != nil {
			showAggregateHelp(ctx)
		}
	}()

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "th", "threshold":
			if i+1 >= len(args) {
				retErr = goerr.New("threshold value is required")
				return
			}
			v, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				retErr = goerr.Wrap(err, "failed to parse threshold")
				return
			}
			if v < 0.00 || v > 1.00 {
				retErr = goerr.New("invalid threshold range (0.00-1.00)")
				return
			}
			threshold = v
			i++ // skip next argument as it's the value

		case "top":
			if i+1 >= len(args) {
				retErr = goerr.New("top value is required")
				return
			}
			v, err := strconv.Atoi(args[i+1])
			if err != nil {
				retErr = goerr.Wrap(err, "failed to parse top value")
				return
			}
			if v < 1 {
				retErr = goerr.New("top value must be greater than 0")
				return
			}
			topN = v
			i++ // skip next argument as it's the value

		default:
			retErr = goerr.New("unknown argument: " + args[i])
			return
		}
	}

	return threshold, topN, nil
}

// Aggregate runs the aggregate command with the given input.
func Create(ctx context.Context, clients *core.Clients, slackMsg *slack.Message, remaining string) (any, error) {
	alertList, err := clients.Repo().GetLatestAlertListInThread(ctx, slackMsg.Thread())
	if err != nil {
		msg.Notify(ctx, "🤔 No alert list found in this thread. Please create one first.")
		return nil, goerr.Wrap(err, "failed to get latest alert list in thread")
	}

	if remaining == "help" || remaining == "h" {
		showAggregateHelp(ctx)
		return nil, nil
	}

	args := strings.Fields(remaining)

	threshold, topN, err := parseArgs(ctx, args)
	if err != nil {
		return nil, err
	}

	alerts, err := alertList.GetAlerts(ctx, clients.Repo())
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alerts")
	}

	ctx = msg.Trace(ctx, "Aggregating %d alerts (threshold: %f, topN: %d)", len(alerts), threshold, topN)

	clusters := alert.ClusterAlerts(ctx, alerts, threshold, topN)

	for _, cluster := range clusters {
		newList, err := clients.CreateList(ctx, *alertList.SlackThread, slackMsg.User(), cluster)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to create alert list")
		}

		slackMessageID, err := clients.Thread().PostAlertList(ctx, newList)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to post alert list")
		}

		// Save SlackMessageID to the alert list
		newList.SlackMessageID = slackMessageID
		if err := clients.Repo().PutAlertList(ctx, newList); err != nil {
			return nil, goerr.Wrap(err, "failed to update alert list")
		}
	}

	return nil, nil
}
