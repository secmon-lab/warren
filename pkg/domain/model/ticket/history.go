package ticket

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

type History struct {
	ID        types.HistoryID `json:"id"`
	TicketID  types.TicketID  `json:"ticket_id"`
	CreatedAt time.Time       `json:"created_at"`
}

func NewHistory(ctx context.Context, ticketID types.TicketID) History {
	return History{
		ID:        types.NewHistoryID(),
		TicketID:  ticketID,
		CreatedAt: clock.Now(ctx),
	}
}
