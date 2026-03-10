package legacy

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/usecase/chat"
)

// CollectThreadComments delegates to the shared chat package implementation.
func CollectThreadComments(ctx context.Context, repo interfaces.Repository, ticketID types.TicketID, currentSession *session.Session) []ticket.Comment {
	return chat.CollectThreadComments(ctx, repo, ticketID, currentSession)
}
