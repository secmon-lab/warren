package base

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/dryrun"
	"github.com/secmon-lab/warren/pkg/utils/embedding"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

func (x *Warren) findNearestTicket(ctx context.Context, in findNearestTicketInput) (map[string]any, error) {
	limit := in.Limit
	duration := in.Duration

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

	now := clock.Now(ctx)
	nearestTickets, err := x.repo.FindNearestTicketsWithSpan(ctx, currentTicket.Embedding, now.AddDate(0, 0, -int(duration)), now, int(limit)+1)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to find nearest tickets",
			goerr.TV(errutil.TicketIDKey, x.ticketID))
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

func (x *Warren) searchTicketsByWords(ctx context.Context, in searchTicketsByWordsInput) (map[string]any, error) {
	query := in.Query
	if query == "" {
		return nil, goerr.New("query is required",
			goerr.TV(errutil.ParameterKey, "query"),
			goerr.T(errutil.TagValidation))
	}

	// Check if LLM client is available
	if x.llmClient == nil {
		return nil, goerr.New("LLM client is not configured for word-based search",
			goerr.T(errutil.TagInternal))
	}

	limit := in.Limit
	if limit <= 0 {
		limit = DefaultSearchTicketsLimit
	}

	duration := in.Duration
	if duration <= 0 {
		duration = DefaultSearchTicketsDuration
	}

	// Generate embedding from the search query
	queryEmbedding, err := embedding.Generate(ctx, x.llmClient, query)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate embedding for query",
			goerr.TV(errutil.QueryKey, query),
			goerr.T(errutil.TagLLMError))
	}

	now := clock.Now(ctx)
	nearestTickets, err := x.repo.FindNearestTicketsWithSpan(ctx, queryEmbedding, now.AddDate(0, 0, -int(duration)), now, int(limit))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to find nearest tickets",
			goerr.TV(errutil.TicketIDKey, x.ticketID))
	}

	// Convert tickets to result format
	var results []any
	for _, t := range nearestTickets {
		results = append(results, t)
	}

	return map[string]any{
		"tickets": results,
		"query":   query,
		"count":   len(results),
	}, nil
}

func (x *Warren) updateFinding(ctx context.Context, in updateFindingInput) (map[string]any, error) {
	summary := in.Summary
	if summary == "" {
		return nil, goerr.New("summary is required",
			goerr.TV(errutil.ParameterKey, "summary"),
			goerr.T(errutil.TagValidation))
	}

	severityStr := in.Severity
	if severityStr == "" {
		return nil, goerr.New("severity is required",
			goerr.TV(errutil.ParameterKey, "severity"),
			goerr.T(errutil.TagValidation))
	}

	reason := in.Reason
	if reason == "" {
		return nil, goerr.New("reason is required",
			goerr.TV(errutil.ParameterKey, "reason"),
			goerr.T(errutil.TagValidation))
	}

	recommendation := in.Recommendation
	if recommendation == "" {
		return nil, goerr.New("recommendation is required",
			goerr.TV(errutil.ParameterKey, "recommendation"),
			goerr.T(errutil.TagValidation))
	}

	// Validate severity
	severity := types.AlertSeverity(severityStr)
	if err := severity.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid severity value",
			goerr.TV(errutil.SeverityKey, string(severity)),
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

	// Check if dry-run mode is enabled
	isDryRun := dryrun.IsDryRun(ctx)

	// Update finding
	updatedTicket := *currentTicket
	updatedTicket.Finding = &ticket.Finding{
		Severity:       severity,
		Summary:        summary,
		Reason:         reason,
		Recommendation: recommendation,
	}
	updatedTicket.UpdatedAt = clock.Now(ctx)

	// Save to database (skip if dry-run)
	if !isDryRun {
		if err := x.repo.PutTicket(ctx, updatedTicket); err != nil {
			return nil, goerr.Wrap(err, "failed to update ticket in database")
		}
	}

	// Update Slack message if callback is provided and ticket has Slack thread (skip if dry-run)
	slackUpdated := false
	if !isDryRun && x.slackUpdate != nil && currentTicket.HasSlackThread() {
		if err := x.slackUpdate(ctx, &updatedTicket); err != nil {
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
				"updated_at":     updatedTicket.UpdatedAt.Format(time.RFC3339),
				"dry_run":        isDryRun,
			}, nil
		}
		slackUpdated = true
	}

	message := "Finding updated successfully"
	if isDryRun {
		message = "Finding update validated (dry-run mode)"
	}

	response := map[string]any{
		"success":        true,
		"message":        message,
		"severity":       string(severity),
		"summary":        summary,
		"reason":         reason,
		"recommendation": recommendation,
		"updated_at":     updatedTicket.UpdatedAt.Format(time.RFC3339),
		"slack_updated":  slackUpdated,
		"dry_run":        isDryRun,
	}

	if !isDryRun && !slackUpdated && currentTicket.HasSlackThread() {
		response["slack_update_required"] = true
		response["message"] = "Finding updated successfully. Slack message update may be needed."
	}

	return response, nil
}
