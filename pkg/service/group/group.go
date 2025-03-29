package group

import (
	"context"
	"strconv"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/prompt"
	"github.com/secmon-lab/warren/pkg/service/llm"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

func Run(ctx context.Context, repo interfaces.Repository, th *slack_svc.ThreadService, llmClient interfaces.LLMInquiry, user slack.User, alerts alert.Alerts, remaining string) error {
	ctx = msg.NewTrace(ctx, "🤖 Grouping alerts...")

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

	clusters := alert.ClusterAlerts(ctx, alerts, threshold, topN)

	var lists []alert.List
	threadModel := slack.Thread{
		ChannelID: th.ChannelID(),
		ThreadID:  th.ThreadID(),
	}
	for _, cluster := range clusters {
		list := alert.NewList(ctx, threadModel, &user, cluster)

		p, err := prompt.BuildMetaListPrompt(ctx, list)
		if err != nil {
			return goerr.Wrap(err, "failed to build meta list prompt")
		}

		msg.Trace(ctx, "📝 Generating meta data for list: %s", list.ID)
		resp, err := llm.Ask[prompt.MetaListPromptResult](ctx, llmClient, p)
		if err != nil {
			msg.Trace(ctx, "💥 failed meta data generation, skip")
			errs.Handle(ctx, err)
		} else {
			list.Title = resp.Title
			list.Description = resp.Description
		}

		if err := repo.PutAlertList(ctx, list); err != nil {
			return goerr.Wrap(err, "failed to put alert list")
		}
		lists = append(lists, list)
	}

	if err := th.PostAlertClusters(ctx, lists); err != nil {
		return goerr.Wrap(err, "failed to post alert clusters")
	}

	return nil
}
