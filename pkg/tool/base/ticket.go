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

func (x *Warren) findNearestTicket(ctx context.Context, args map[string]any) (map[string]any, error) {
	limit, err := getArg[int64](args, "limit")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get limit",
			goerr.TV(errutil.ParameterKey, "limit"),
			goerr.T(errutil.TagValidation))
	}

	duration, err := getArg[int64](args, "duration")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get duration",
			goerr.TV(errutil.ParameterKey, "duration"),
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

func (x *Warren) searchTicketsByWords(ctx context.Context, args map[string]any) (map[string]any, error) {
	query, err := getArg[string](args, "query")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get query",
			goerr.TV(errutil.ParameterKey, "query"),
			goerr.T(errutil.TagValidation))
	}
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

	limit, err := getArg[int64](args, "limit")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get limit",
			goerr.TV(errutil.ParameterKey, "limit"),
			goerr.T(errutil.TagValidation))
	}
	if limit <= 0 {
		limit = DefaultSearchTicketsLimit
	}

	duration, err := getArg[int64](args, "duration")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get duration",
			goerr.TV(errutil.ParameterKey, "duration"),
			goerr.T(errutil.TagValidation))
	}
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

func (x *Warren) updateFinding(ctx context.Context, args map[string]any) (map[string]any, error) {
	summary, err := getArg[string](args, "summary")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get summary",
			goerr.TV(errutil.ParameterKey, "summary"),
			goerr.T(errutil.TagValidation))
	}
	if summary == "" {
		return nil, goerr.New("summary is required",
			goerr.TV(errutil.ParameterKey, "summary"),
			goerr.T(errutil.TagValidation))
	}

	severityStr, err := getArg[string](args, "severity")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get severity",
			goerr.TV(errutil.ParameterKey, "severity"),
			goerr.T(errutil.TagValidation))
	}
	if severityStr == "" {
		return nil, goerr.New("severity is required",
			goerr.TV(errutil.ParameterKey, "severity"),
			goerr.T(errutil.TagValidation))
	}

	reason, err := getArg[string](args, "reason")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get reason",
			goerr.TV(errutil.ParameterKey, "reason"),
			goerr.T(errutil.TagValidation))
	}
	if reason == "" {
		return nil, goerr.New("reason is required",
			goerr.TV(errutil.ParameterKey, "reason"),
			goerr.T(errutil.TagValidation))
	}

	recommendation, err := getArg[string](args, "recommendation")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get recommendation",
			goerr.TV(errutil.ParameterKey, "recommendation"),
			goerr.T(errutil.TagValidation))
	}
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

func (x *Warren) getTicketComments(ctx context.Context, args map[string]any) (map[string]any, error) {
	// Get optional pagination parameters
	limitVal, err := getArg[int64](args, "limit")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get limit",
			goerr.TV(errutil.ParameterKey, "limit"),
			goerr.T(errutil.TagValidation))
	}

	offsetVal, err := getArg[int64](args, "offset")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get offset")
	}

	// Set default values
	limit := int(limitVal)
	offset := int(offsetVal)
	if limit <= 0 {
		limit = DefaultCommentsLimit
	}
	if offset < 0 {
		offset = 0
	}

	// Get comments with pagination
	comments, err := x.repo.GetTicketCommentsPaginated(ctx, x.ticketID, offset, limit)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get ticket comments")
	}

	// Get total count
	totalCount, err := x.repo.CountTicketComments(ctx, x.ticketID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to count ticket comments")
	}

	// Convert comments to serializable format
	var results []map[string]any
	for _, comment := range comments {
		commentData := map[string]any{
			"id":         string(comment.ID),
			"ticket_id":  string(comment.TicketID),
			"created_at": comment.CreatedAt.Format(time.RFC3339),
			"comment":    comment.Comment,
			"prompted":   comment.Prompted,
		}

		// Add user information if available
		if comment.User != nil {
			commentData["user"] = map[string]any{
				"id":   comment.User.ID,
				"name": comment.User.Name,
			}
		}

		// Add Slack message ID if available
		if comment.SlackMessageID != "" {
			commentData["slack_message_id"] = comment.SlackMessageID
		}

		results = append(results, commentData)
	}

	return map[string]any{
		"comments":    results,
		"total_count": totalCount,
		"offset":      offset,
		"limit":       limit,
		"has_more":    offset+len(results) < totalCount,
	}, nil
}
