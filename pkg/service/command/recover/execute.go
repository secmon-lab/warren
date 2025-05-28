package recover

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/service/command/core"
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
		msg.Trace(ctx, "✅ No alerts missing metadata")
		return nil
	}

	msg.Trace(ctx, "🔨Fixing alerts missing metadata: %d", len(alerts))

	for i, alert := range alerts {
		if i%32 == 0 {
			msg.Trace(ctx, "Fixing alert metadata %d/%d: %s", i+1, len(alerts), alert.ID)
		}

		if err := alert.FillMetadata(ctx, clients.LLM()); err != nil {
			return err
		}

		if err := clients.Repo().PutAlert(ctx, *alert); err != nil {
			return err
		}
	}

	msg.Trace(ctx, "✅ Fixed %d alerts missing metadata", len(alerts))
	return nil
}
