package session

import (
	"log/slog"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
)

type History struct {
	ID        types.HistoryID `firestore:"id" json:"id"`
	SessionID types.SessionID `firestore:"session_id" json:"session_id"`
	CreatedAt time.Time       `firestore:"created_at" json:"created_at"`
}

func (x *History) LogValue() slog.Value {
	if x == nil {
		return slog.AnyValue(nil)
	}

	return slog.GroupValue(
		slog.String("id", x.ID.String()),
		slog.Any("created_at", x.CreatedAt),
	)
}
