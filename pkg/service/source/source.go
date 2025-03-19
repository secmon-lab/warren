package source

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/thread"
)

type Source func(ctx context.Context, repo interfaces.Repository) ([]model.Alert, error)

func Thread(slackThread model.SlackThread) Source {
	return func(ctx context.Context, repo interfaces.Repository) ([]model.Alert, error) {
		thread.Reply(ctx, "🤖 Getting alerts from slack thread...")

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

func LatestAlertList(slackThread model.SlackThread) Source {
	return func(ctx context.Context, repo interfaces.Repository) ([]model.Alert, error) {
		thread.Reply(ctx, "🤖 Getting latest alerts from slack thread")
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

func AlertListID(alertListID model.AlertListID) Source {
	return func(ctx context.Context, repo interfaces.Repository) ([]model.Alert, error) {
		thread.Reply(ctx, "🤖 Getting alerts from alert list: "+alertListID.String())

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
	return func(ctx context.Context, repo interfaces.Repository) ([]model.Alert, error) {
		thread.Reply(ctx, "🤖 Getting alerts from span: "+begin.Format("2006-01-02 15:04")+" to "+end.Format("2006-01-02 15:04"))

		alerts, err := repo.GetAlertsBySpan(ctx, begin, end)
		if err != nil {
			return nil, err
		}
		return alerts, nil
	}
}

func Alert(alert *model.Alert) Source {
	return func(ctx context.Context, repo interfaces.Repository) ([]model.Alert, error) {
		thread.Reply(ctx, "🤖 Getting alerts from alert: "+alert.ID.String())

		alerts, err := repo.GetAlertsByParentID(ctx, alert.ID)
		if err != nil {
			return nil, err
		}

		alerts = append(alerts, *alert)
		return alerts, nil
	}
}

func Static(alerts []model.Alert) Source {
	return func(ctx context.Context, repo interfaces.Repository) ([]model.Alert, error) {
		return alerts, nil
	}
}

func Unresolved() Source {
	return func(ctx context.Context, repo interfaces.Repository) ([]model.Alert, error) {
		thread.Reply(ctx, "🤖 Getting unresolved alerts...")

		alerts, err := repo.GetAlertsWithoutStatus(ctx, model.AlertStatusResolved)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get alerts without resolved status")
		}

		return alerts, nil
	}
}
