package alert

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

type List struct {
	ID          types.AlertListID `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	AlertIDs    []types.AlertID   `json:"alert_ids"`
	SlackThread *slack.Thread     `json:"slack_thread"`
	CreatedAt   time.Time         `json:"created_at"`
	CreatedBy   *slack.User       `json:"created_by"`

	Alerts Alerts `firestore:"-"`
}

func NewList(ctx context.Context, thread slack.Thread, createdBy *slack.User, alerts Alerts) List {
	list := List{
		ID:          types.NewAlertListID(),
		SlackThread: &thread,
		CreatedAt:   clock.Now(ctx),
		CreatedBy:   createdBy,
	}
	for _, alert := range alerts {
		list.AlertIDs = append(list.AlertIDs, alert.ID)
		list.Alerts = append(list.Alerts, alert)
	}

	return list
}
