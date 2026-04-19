package base

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

// searchSessionMessages handles the warren_search_session_messages tool
// call. It delegates to Repository.SearchSessionMessages which performs
// a case-insensitive substring match across every SessionMessage bound
// to the current ticket and returns the top `limit` results.
//
// This is the chat-session-redesign replacement for the generic
// "past message search" surface called out in the spec
// (pkg/tool/past_message_search). Since the backing query is already
// ticket-scoped through the Repository method, we expose it as another
// function on the ticket-scoped warren tool rather than a separate
// package.
func (x *Warren) searchSessionMessages(ctx context.Context, args map[string]any) (map[string]any, error) {
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
	limitVal, err := getArg[int64](args, "limit")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get limit",
			goerr.TV(errutil.ParameterKey, "limit"),
			goerr.T(errutil.TagValidation))
	}
	limit := int(limitVal)
	if limit <= 0 {
		limit = DefaultCommentsLimit
	}

	messages, err := x.repo.SearchSessionMessages(ctx, x.ticketID, query, limit)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to search session messages",
			goerr.TV(errutil.TicketIDKey, x.ticketID),
			goerr.V("query", query))
	}

	results := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		entry := map[string]any{
			"id":         string(m.ID),
			"session_id": string(m.SessionID),
			"type":       string(m.Type),
			"content":    m.Content,
			"created_at": m.CreatedAt.Format(time.RFC3339),
		}
		if m.TurnID != nil {
			entry["turn_id"] = string(*m.TurnID)
		}
		if m.Author != nil {
			author := map[string]any{
				"user_id":      string(m.Author.UserID),
				"display_name": m.Author.DisplayName,
			}
			if m.Author.SlackUserID != nil {
				author["slack_user_id"] = *m.Author.SlackUserID
			}
			entry["author"] = author
		}
		results = append(results, entry)
	}

	return map[string]any{
		"ticket_id": string(x.ticketID),
		"query":     query,
		"messages":  results,
		"limit":     limit,
	}, nil
}
