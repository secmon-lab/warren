package source

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

type Source func(ctx context.Context, repo interfaces.Repository) (alert.Alerts, error)

func Thread(slackThread slack.Thread) Source {
	return func(ctx context.Context, repo interfaces.Repository) (alert.Alerts, error) {
		msg.Trace(ctx, "🤖 Getting alerts from slack thread...")

		alertList, err := repo.GetAlertListByThread(ctx, slackThread)
		if err != nil {
			return nil, err
		}
		if alertList == nil {
			return nil, nil
		}
		alerts, err := repo.BatchGetAlerts(ctx, alertList.AlertIDs)
		if err != nil {
			return nil, err
		}
		return alerts, nil
	}
}

func RootAlertList(slackThread slack.Thread) Source {
	return func(ctx context.Context, repo interfaces.Repository) (alert.Alerts, error) {
		msg.Trace(ctx, "🤖 Getting root alerts from alert list")
		logging.From(ctx).Info("Getting root alerts from alert list")

		return nil, nil
	}
}

func LatestAlertList(slackThread slack.Thread) Source {
	return func(ctx context.Context, repo interfaces.Repository) (alert.Alerts, error) {
		msg.Trace(ctx, "🤖 Getting latest alerts from slack thread")
		logging.From(ctx).Info("Getting latest alerts from slack thread", "slack_thread", slackThread)

		alertList, err := repo.GetLatestAlertListInThread(ctx, slackThread)
		if err != nil {
			return nil, err
		}
		if alertList == nil {
			return nil, goerr.New("no alert list found in this thread", goerr.V("slack_thread", slackThread))
		}
		alerts, err := repo.BatchGetAlerts(ctx, alertList.AlertIDs)
		if err != nil {
			return nil, err
		}
		return alerts, nil
	}
}

func AlertListID(alertListID types.AlertListID) Source {
	return func(ctx context.Context, repo interfaces.Repository) (alert.Alerts, error) {
		msg.Trace(ctx, "🤖 Getting alerts from alert list: %s", alertListID)

		alertList, err := repo.GetAlertList(ctx, alertListID)
		if err != nil {
			return nil, err
		}
		if alertList == nil {
			return nil, nil
		}
		alerts, err := repo.BatchGetAlerts(ctx, alertList.AlertIDs)
		if err != nil {
			return nil, err
		}
		return alerts, nil
	}
}

func Span(begin, end time.Time) Source {
	return func(ctx context.Context, repo interfaces.Repository) (alert.Alerts, error) {
		msg.Trace(ctx, "🤖 Getting alerts from span: %s to %s", begin.Format("2006-01-02 15:04"), end.Format("2006-01-02 15:04"))

		alerts, err := repo.GetAlertsBySpan(ctx, begin, end)
		if err != nil {
			return nil, err
		}
		return alerts, nil
	}
}

func Static(alerts alert.Alerts) Source {
	return func(ctx context.Context, repo interfaces.Repository) (alert.Alerts, error) {
		return alerts, nil
	}
}

func Status(status ...types.AlertStatus) Source {
	return func(ctx context.Context, repo interfaces.Repository) (alert.Alerts, error) {
		msg.Trace(ctx, "🤖 Getting alerts with status: %s", status)

		alerts, err := repo.GetAlertsByStatus(ctx, status...)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get alerts without status")
		}

		return alerts, nil
	}
}

func Unresolved() Source {
	return func(ctx context.Context, repo interfaces.Repository) (alert.Alerts, error) {
		msg.Trace(ctx, "🤖 Getting unresolved alerts...")

		alerts, err := repo.GetAlertsWithoutStatus(ctx, types.AlertStatusResolved)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get alerts without resolved status")
		}

		return alerts, nil
	}
}
