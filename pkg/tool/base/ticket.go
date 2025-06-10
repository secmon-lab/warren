package base

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

func (x *Warren) findNearestTicket(ctx context.Context, args map[string]any) (map[string]any, error) {
	limit, err := getArg[int64](args, "limit")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get limit")
	}

	duration, err := getArg[int64](args, "duration")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get duration")
	}

	// Get current ticket
	currentTicket, err := x.repo.GetTicket(ctx, x.ticketID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get current ticket")
	}
	if currentTicket == nil {
		return nil, goerr.New("ticket not found", goerr.V("ticket_id", x.ticketID))
	}

	now := clock.Now(ctx)
	nearestTickets, err := x.repo.FindNearestTicketsWithSpan(ctx, currentTicket.Embedding, now.Add(-time.Duration(duration)*24*time.Hour), now, int(limit)+1)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to find nearest tickets")
	}

	var results []any
	for _, t := range nearestTickets {
		if t.ID == currentTicket.ID {
			continue
		}
		results = append(results, t)
	}

	if len(results) > int(limit) {
		results = results[:limit]
	}

	return map[string]any{
		"tickets": results,
	}, nil
}

func (x *Warren) updateFinding(ctx context.Context, args map[string]any) (map[string]any, error) {
	summary, err := getArg[string](args, "summary")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get summary")
	}
	if summary == "" {
		return nil, goerr.New("summary is required")
	}

	severityStr, err := getArg[string](args, "severity")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get severity")
	}
	if severityStr == "" {
		return nil, goerr.New("severity is required")
	}

	reason, err := getArg[string](args, "reason")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get reason")
	}
	if reason == "" {
		return nil, goerr.New("reason is required")
	}

	recommendation, err := getArg[string](args, "recommendation")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get recommendation")
	}
	if recommendation == "" {
		return nil, goerr.New("recommendation is required")
	}

	// Validate severity
	severity := types.AlertSeverity(severityStr)
	if err := severity.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid severity value", goerr.V("severity", severity))
	}

	// Get current ticket
	currentTicket, err := x.repo.GetTicket(ctx, x.ticketID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get current ticket")
	}
	if currentTicket == nil {
		return nil, goerr.New("ticket not found", goerr.V("ticket_id", x.ticketID))
	}

	// Update finding
	currentTicket.Finding = &ticket.Finding{
		Severity:       severity,
		Summary:        summary,
		Reason:         reason,
		Recommendation: recommendation,
	}
	currentTicket.UpdatedAt = clock.Now(ctx)

	// Save to database
	if err := x.repo.PutTicket(ctx, *currentTicket); err != nil {
		return nil, goerr.Wrap(err, "failed to update ticket in database")
	}

	// Update Slack message if callback is provided and ticket has Slack thread
	slackUpdated := false
	if x.slackUpdate != nil && currentTicket.SlackThread != nil {
		if err := x.slackUpdate(ctx, currentTicket); err != nil {
			// Don't fail the entire operation if Slack update fails
			// Just log the error and continue
			return map[string]any{
				"success":        true,
				"message":        "Finding updated successfully, but Slack update failed",
				"slack_error":    err.Error(),
				"severity":       string(severity),
				"summary":        summary,
				"reason":         reason,
				"recommendation": recommendation,
				"updated_at":     currentTicket.UpdatedAt.Format(time.RFC3339),
			}, nil
		}
		slackUpdated = true
	}

	response := map[string]any{
		"success":        true,
		"message":        "Finding updated successfully",
		"severity":       string(severity),
		"summary":        summary,
		"reason":         reason,
		"recommendation": recommendation,
		"updated_at":     currentTicket.UpdatedAt.Format(time.RFC3339),
		"slack_updated":  slackUpdated,
	}

	if !slackUpdated && currentTicket.SlackThread != nil {
		response["slack_update_required"] = true
		response["message"] = "Finding updated successfully. Slack message update may be needed."
	}

	return response, nil
}
