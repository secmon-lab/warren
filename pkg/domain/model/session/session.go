package session

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/ptr"
)

type Session struct {
	ID        types.SessionID `json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	CreatedBy *slack.User     `json:"created_by"`
	Thread    *slack.Thread   `json:"thread"`
	AlertIDs  []types.AlertID `json:"alert_ids"`
}

func New(ctx context.Context, msg *slack.Message, alertIDs []types.AlertID) *Session {
	ssn := &Session{
		ID:        types.NewSessionID(),
		CreatedAt: clock.Now(ctx),
		AlertIDs:  alertIDs,
	}

	if msg != nil {
		ssn.CreatedBy = ptr.Ref(msg.User())
		ssn.Thread = ptr.Ref(msg.Thread())
	}

	return ssn
}
