package recover

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/service/command/core"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

func Run(ctx context.Context, clients *core.Clients, slackMsg *slack.Message, input string) (any, error) {
	if err := fixAlertsWithoutMetadata(ctx, clients, slackMsg, input); err != nil {
		return nil, err
	}

	return nil, nil
}

func fixAlertsWithoutMetadata(ctx context.Context, clients *core.Clients, slackMsg *slack.Message, input string) error {
	alerts, err := clients.Repo().GetAlertWithoutEmbedding(ctx)
	if err != nil {
		return err
	}
	if len(alerts) == 0 {
		_ = msg.Trace(ctx, "✅ No alerts missing metadata")
		return nil
	}

	msg.Notify(ctx, "🔨Fixing alerts missing metadata: %d", len(alerts))

	ts := clock.Now(ctx)
	notifyPeriod := time.Minute * 2
	for i, alert := range alerts {
		if clock.Since(ctx, ts) > notifyPeriod {
			ctx = msg.Trace(ctx, "⌛ Fixing alert metadata %d/%d", i+1, len(alerts))
			ts = clock.Now(ctx)
		}

		if err := alert.FillMetadata(ctx, clients.LLM()); err != nil {
			return err
		}

		if err := clients.Repo().PutAlert(ctx, *alert); err != nil {
			return err
		}
	}

	_ = msg.Trace(ctx, "✅ Fixed %d alerts missing metadata", len(alerts))
	return nil
}
