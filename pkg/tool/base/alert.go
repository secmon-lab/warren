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

	// Get current ticket
	currentTicket, err := x.repo.GetTicket(ctx, x.ticketID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get current ticket")
	}
	if currentTicket == nil {
		return nil, goerr.New("ticket not found", goerr.V("ticket_id", x.ticketID))
	}

	alerts, err := x.repo.BatchGetAlerts(ctx, currentTicket.AlertIDs)
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

	rows := make([]any, 0, len(alerts))
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
