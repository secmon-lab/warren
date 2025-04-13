package aggr

import (
	"context"
	"strconv"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/gemini"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/command/list"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

func Run(ctx context.Context, repo interfaces.Repository, slackThread *slack_svc.ThreadService, llmClient interfaces.LLMClient, user slack.User, alertIDs []types.AlertID, remaining string) error {
	ctx = msg.NewTrace(ctx, "🤖 Aggregating alerts...")

	alerts, err := repo.BatchGetAlerts(ctx, alertIDs)
	if err != nil {
		return goerr.Wrap(err, "failed to get alerts")
	}

	args := strings.Fields(remaining)

	threshold := 0.99

	if len(args) > 0 {
		v, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			return goerr.Wrap(err, "failed to parse threshold (0.00-1.00)")
		}
		if v < 0.00 || v > 1.00 {
			return goerr.Wrap(err, "invalid threshold range (0.00-1.00)")
		}
		threshold = v
	}

	topN := 5
	if len(args) > 1 {
		v, err := strconv.Atoi(args[1])
		if err != nil {
			return goerr.Wrap(err, "failed to parse topN (1-10)")
		}
		topN = v
	}

	logging.From(ctx).Debug("Starting group", "threshold", threshold, "topN", topN, "alerts", len(alerts))

	clusters := alert.ClusterAlerts(ctx, alerts, threshold, topN)

	threadModel := slack.Thread{
		ChannelID: slackThread.ChannelID(),
		ThreadID:  slackThread.ThreadID(),
	}

	var lists []alert.List
	query := llmClient.NewQuery(
		gemini.WithContentType("application/json"),
	)

	for _, cluster := range clusters {
		newList, err := list.CreateList(ctx, repo, query, threadModel, &user, cluster)
		if err != nil {
			return goerr.Wrap(err, "failed to create alert list")
		}

		lists = append(lists, *newList)
	}

	if err := slackThread.PostAlertClusters(ctx, lists); err != nil {
		return goerr.Wrap(err, "failed to post alert clusters")
	}

	return nil
}
