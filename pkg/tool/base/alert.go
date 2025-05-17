package base

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
)

func (x *Base) getAlerts(ctx context.Context, args map[string]any) (map[string]any, error) {
	var limit, offset int64

	if limitVal, ok := args["limit"].(float64); ok {
		limit = int64(limitVal)
	}
	if offsetVal, ok := args["offset"].(float64); ok {
		offset = int64(offsetVal)
	}

	alerts, err := x.repo.BatchGetAlerts(ctx, x.alertIDs)
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

func (x *Base) searchAlerts(ctx context.Context, args map[string]any) (map[string]any, error) {
	path, err := getArg[string](args, "path")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get path")
	}

	op, err := getArg[string](args, "op")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get op")
	}

	value, err := getArg[string](args, "value")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get value")
	}

	var offset, limit float64
	if offsetVal, ok := args["offset"].(float64); ok {
		offset = offsetVal
	}
	if limitVal, ok := args["limit"].(float64); ok {
		limit = limitVal
	}
	if limit <= 0 {
		limit = 25
	}

	alerts, err := x.repo.SearchAlerts(ctx, path, op, value, int(limit))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to search alerts")
	}

	if offset > 0 {
		if int(offset) >= len(alerts) {
			alerts = nil
		} else {
			alerts = alerts[int(offset):]
		}
	}

	if limit < float64(len(alerts)) {
		alerts = alerts[:int(limit)]
	}

	alertMap := make(map[string]any)
	for _, alert := range alerts {
		raw, err := json.Marshal(alert)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to marshal alert")
		}
		alertMap[alert.ID.String()] = string(raw)
	}

	return map[string]any{
		"alerts": alertMap,
		"count":  len(alerts),
		"offset": offset,
		"limit":  limit,
	}, nil
}
