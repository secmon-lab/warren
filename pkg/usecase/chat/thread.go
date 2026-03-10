package chat

import (
	"context"
	"sort"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

// CollectThreadComments retrieves thread comments posted between sessions.
func CollectThreadComments(ctx context.Context, repo interfaces.Repository, ticketID types.TicketID, currentSession *session.Session) []ticket.Comment {
	logger := logging.From(ctx)

	const maxThreadComments = 50

	sessions, err := repo.GetSessionsByTicket(ctx, ticketID)
	if err != nil {
		logger.Warn("failed to get sessions for thread comments", "error", err, "ticket_id", ticketID)
		return nil
	}

	var prevSessionCreatedAt time.Time
	for _, s := range sessions {
		if s.ID == currentSession.ID {
			continue
		}
		if s.Status != types.SessionStatusCompleted {
			continue
		}
		if s.CreatedAt.Before(currentSession.CreatedAt) && s.CreatedAt.After(prevSessionCreatedAt) {
			prevSessionCreatedAt = s.CreatedAt
		}
	}

	logger.Debug("CollectThreadComments",
		"ticket_id", ticketID,
		"total_sessions", len(sessions),
		"prev_session_created_at", prevSessionCreatedAt,
		"current_session_created_at", currentSession.CreatedAt,
	)

	comments, err := repo.GetTicketComments(ctx, ticketID)
	if err != nil {
		logger.Warn("failed to get ticket comments for thread context", "error", err, "ticket_id", ticketID)
		return nil
	}

	var filtered []ticket.Comment
	for _, co := range comments {
		if co.CreatedAt.After(prevSessionCreatedAt) && co.CreatedAt.Before(currentSession.CreatedAt) {
			filtered = append(filtered, co)
		}
	}

	logger.Debug("CollectThreadComments filtered",
		"total_comments", len(comments),
		"filtered_count", len(filtered),
	)

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})

	if len(filtered) > maxThreadComments {
		filtered = filtered[len(filtered)-maxThreadComments:]
	}

	return filtered
}
