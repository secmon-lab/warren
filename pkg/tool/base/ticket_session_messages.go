package base

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

// getTicketSessionMessages handles the warren_get_ticket_session_messages
// tool call. It returns messages from every Session attached to the
// current ticket, optionally filtered by source and/or type.
//
// This is the chat-session-redesign replacement for
// warren_get_ticket_comments. It returns a strictly richer payload
// (author information, AI output types, turn grouping) while covering
// the same ground on the Slack/user-comment side.
func (x *Warren) getTicketSessionMessages(ctx context.Context, args map[string]any) (map[string]any, error) {
	limitVal, err := getArg[int64](args, "limit")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get limit",
			goerr.TV(errutil.ParameterKey, "limit"),
			goerr.T(errutil.TagValidation))
	}
	offsetVal, err := getArg[int64](args, "offset")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get offset",
			goerr.TV(errutil.ParameterKey, "offset"),
			goerr.T(errutil.TagValidation))
	}
	sourceStr, err := getArg[string](args, "source")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get source",
			goerr.TV(errutil.ParameterKey, "source"),
			goerr.T(errutil.TagValidation))
	}
	typeStr, err := getArg[string](args, "type")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get type",
			goerr.TV(errutil.ParameterKey, "type"),
			goerr.T(errutil.TagValidation))
	}

	limit := int(limitVal)
	offset := int(offsetVal)
	if limit <= 0 {
		limit = DefaultCommentsLimit
	}
	if offset < 0 {
		offset = 0
	}

	var sourceFilter *session.SessionSource
	if sourceStr != "" {
		s := session.SessionSource(sourceStr)
		if !s.Valid() {
			return nil, goerr.New("invalid source value",
				goerr.V("source", sourceStr),
				goerr.T(errutil.TagValidation))
		}
		sourceFilter = &s
	}
	var typeFilter *session.MessageType
	if typeStr != "" {
		mt := session.MessageType(typeStr)
		if !mt.Valid() {
			return nil, goerr.New("invalid type value",
				goerr.V("type", typeStr),
				goerr.T(errutil.TagValidation))
		}
		typeFilter = &mt
	}

	messages, err := x.repo.GetTicketSessionMessages(ctx, x.ticketID, sourceFilter, typeFilter, limit, offset)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get ticket session messages",
			goerr.TV(errutil.TicketIDKey, x.ticketID))
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
		if m.TicketID != nil {
			entry["ticket_id"] = string(*m.TicketID)
		}
		if m.Author != nil {
			author := map[string]any{
				"user_id":      string(m.Author.UserID),
				"display_name": m.Author.DisplayName,
			}
			if m.Author.SlackUserID != nil {
				author["slack_user_id"] = *m.Author.SlackUserID
			}
			if m.Author.Email != nil {
				author["email"] = *m.Author.Email
			}
			entry["author"] = author
		}
		results = append(results, entry)
	}

	return map[string]any{
		"ticket_id": string(x.ticketID),
		"messages":  results,
		"limit":     limit,
		"offset":    offset,
	}, nil
}
