package base

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func (x *Base) getAlerts(ctx context.Context, args map[string]any) (*action.Result, error) {
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
			alerts = nil
		} else {
			alerts = alerts[offset:]
		}
	}

	if limit > 0 && limit < int64(len(alerts)) {
		alerts = alerts[:limit]
	}

	var rows []string
	for _, alert := range alerts {
		raw, err := json.Marshal(alert)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to marshal alert")
		}
		rows = append(rows, string(raw))
	}

	return &action.Result{
		Name: "base.alerts",
		Data: map[string]any{
			"alerts": rows,
		},
	}, nil
}

func (x *Base) searchAlerts(ctx context.Context, args map[string]any) (*action.Result, error) {
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

	offset, err := getArg[float64](args, "offset")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get offset")
	}

	limit, err := getArg[float64](args, "limit")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get limit")
	}
	if limit <= 0 {
		limit = 25
	}

	alerts, err := x.repo.SearchAlerts(ctx, path, op, value)
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

	return &action.Result{
		Name: "base.alerts",
		Data: alertMap,
	}, nil
}

func (x *Base) resolveAlert(ctx context.Context, args map[string]any) (*action.Result, error) {
	strAlertID, err := getArg[string](args, "alert_id")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alert_id")
	}
	alertID := types.AlertID(strAlertID)

	strConclusion, err := getArg[string](args, "conclusion")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get conclusion")
	}
	conclusion := types.AlertConclusion(strConclusion)

	reason, err := getArg[string](args, "reason")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get reason")
	}

	if err := conclusion.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid conclusion", goerr.V("conclusion", strConclusion))
	}

	alert, err := x.repo.GetAlert(ctx, types.AlertID(alertID))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alert")
	}
	if alert == nil {
		return nil, goerr.New("alert not found", goerr.V("alert_id", alertID))
	}

	alert.Conclusion = types.AlertConclusion(conclusion)
	alert.Reason = reason

	if err := x.repo.PutAlert(ctx, *alert); err != nil {
		return nil, goerr.Wrap(err, "failed to put alert")
	}

	return nil, nil
}
