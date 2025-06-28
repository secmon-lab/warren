package activity

import (
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
)

type Activity struct {
	ID        types.ActivityID
	Type      types.ActivityType
	UserID    string
	AlertID   types.AlertID
	TicketID  types.TicketID
	CommentID types.CommentID
	CreatedAt time.Time
	Metadata  map[string]any
}

func New(activityType types.ActivityType) *Activity {
	return &Activity{
		ID:        types.NewActivityID(),
		Type:      activityType,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]any),
	}
}

func (a *Activity) WithUserID(userID string) *Activity {
	a.UserID = userID
	return a
}

func (a *Activity) WithTicketID(ticketID types.TicketID) *Activity {
	a.TicketID = ticketID
	return a
}

func (a *Activity) WithAlertID(alertID types.AlertID) *Activity {
	a.AlertID = alertID
	return a
}

func (a *Activity) WithCommentID(commentID types.CommentID) *Activity {
	a.CommentID = commentID
	return a
}

func (a *Activity) WithMetadata(key string, value any) *Activity {
	if a.Metadata == nil {
		a.Metadata = make(map[string]any)
	}
	a.Metadata[key] = value
	return a
}
