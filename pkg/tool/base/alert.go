package base

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

func (x *Warren) getAlerts(ctx context.Context, args map[string]any) (map[string]any, error) {
	limit, err := getArg[int64](args, "limit")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get limit",
			goerr.TV(errutil.ParameterKey, "limit"),
			goerr.T(errutil.TagValidation))
	}

	offset, err := getArg[int64](args, "offset")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get offset",
			goerr.TV(errutil.ParameterKey, "offset"),
			goerr.T(errutil.TagValidation))
	}

	// Get current ticket
	currentTicket, err := x.repo.GetTicket(ctx, x.ticketID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get current ticket",
			goerr.TV(errutil.TicketIDKey, x.ticketID))
	}
	if currentTicket == nil {
		return nil, goerr.New("ticket not found",
			goerr.TV(errutil.TicketIDKey, x.ticketID),
			goerr.T(errutil.TagNotFound))
	}

	alerts, err := x.repo.BatchGetAlerts(ctx, currentTicket.AlertIDs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alerts",
			goerr.TV(errutil.TicketIDKey, x.ticketID))
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
			return nil, goerr.Wrap(err, "failed to marshal alert",
				goerr.TV(errutil.AlertIDKey, alert.ID),
				goerr.T(errutil.TagInternal))
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
