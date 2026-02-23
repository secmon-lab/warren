package refine

import (
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
)

// GroupStatus represents the status of a refine group
type GroupStatus string

const (
	GroupStatusPending  GroupStatus = "pending"
	GroupStatusAccepted GroupStatus = "accepted"
	GroupStatusExpired  GroupStatus = "expired"
)

func (s GroupStatus) String() string {
	return string(s)
}

// Group represents a consolidation candidate group of unbound alerts
type Group struct {
	ID             types.RefineGroupID `json:"id" firestore:"ID"`
	PrimaryAlertID types.AlertID       `json:"primary_alert_id" firestore:"PrimaryAlertID"`
	AlertIDs       []types.AlertID     `json:"alert_ids" firestore:"AlertIDs"`
	Reason         string              `json:"reason" firestore:"Reason"`
	CreatedAt      time.Time           `json:"created_at" firestore:"CreatedAt"`
	Status         GroupStatus         `json:"status" firestore:"Status"`
}
