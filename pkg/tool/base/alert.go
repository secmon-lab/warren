package base

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
)

func (x *Warren) getAlerts(ctx context.Context, args map[string]any) (map[string]any, error) {
	limit, err := getArg[int64](args, "limit")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get limit")
	}

	offset, err := getArg[int64](args, "offset")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get offset")
	}

	alerts, err := x.repo.BatchGetAlerts(ctx, x.ticket.AlertIDs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alerts")
	}

	// Apply pagination
	if offset > 0 {
		if offset >= int64(len(alerts)) {
			alerts = alert.Alerts{}
		} else {
			alerts = alerts[offset:]
		}
	}

	if limit > 0 && limit < int64(len(alerts)) {
		alerts = alerts[:limit]
	}

	rows := make([]string, 0, len(alerts))
	for _, alert := range alerts {
		raw, err := json.Marshal(alert)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to marshal alert")
		}
		rows = append(rows, string(raw))
	}

	return map[string]any{
		"alerts": rows,
		"count":  len(alerts),
		"offset": offset,
		"limit":  limit,
	}, nil
}
