package session

import (
	"context"
	"log/slog"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
)

type History struct {
	ID        types.HistoryID `json:"id"`
	SessionID types.SessionID `json:"session_id"`
	CreatedAt time.Time       `json:"created_at"`
}

func NewHistory(ctx context.Context, ssnID types.SessionID) *History {
	return &History{
		ID:        types.NewHistoryID(),
		SessionID: ssnID,
		CreatedAt: time.Now().UTC(),
	}
}

func (x *History) LogValue() slog.Value {
	if x == nil {
		return slog.AnyValue(nil)
	}

	return slog.GroupValue(
		slog.String("id", x.ID.String()),
		slog.String("session_id", x.SessionID.String()),
		slog.Any("created_at", x.CreatedAt),
	)
}
