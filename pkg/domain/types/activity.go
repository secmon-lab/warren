package types

import (
	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
)

type ActivityID string

func (x ActivityID) String() string {
	return string(x)
}

func NewActivityID() ActivityID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return ActivityID(id.String())
}

func (x ActivityID) Validate() error {
	if x == EmptyActivityID {
		return goerr.New("empty activity ID")
	}
	if _, err := uuid.Parse(string(x)); err != nil {
		return goerr.Wrap(err, "invalid activity ID format", goerr.V("id", x))
	}
	return nil
}

const (
	EmptyActivityID ActivityID = ""
)

type ActivityType string

const (
	ActivityTypeTicketCreated       ActivityType = "ticket_created"
	ActivityTypeTicketUpdated       ActivityType = "ticket_updated"
	ActivityTypeTicketStatusChanged ActivityType = "ticket_status_changed"
	ActivityTypeCommentAdded        ActivityType = "comment_added"
	ActivityTypeAlertBound          ActivityType = "alert_bound"
	ActivityTypeAlertsBulkBound     ActivityType = "alerts_bulk_bound"
)

var activityTypeLabels = map[ActivityType]string{
	ActivityTypeTicketCreated:       "Ticket Created",
	ActivityTypeTicketUpdated:       "Ticket Updated",
	ActivityTypeTicketStatusChanged: "Status Changed",
	ActivityTypeCommentAdded:        "Comment Added",
	ActivityTypeAlertBound:          "Alert Bound",
	ActivityTypeAlertsBulkBound:     "Alerts Bulk Bound",
}

func (t ActivityType) String() string {
	return string(t)
}

func (t ActivityType) Label() string {
	return activityTypeLabels[t]
}

func (t ActivityType) Validate() error {
	switch t {
	case ActivityTypeTicketCreated, ActivityTypeTicketUpdated, ActivityTypeTicketStatusChanged, ActivityTypeCommentAdded, ActivityTypeAlertBound, ActivityTypeAlertsBulkBound:
		return nil
	}
	return goerr.New("invalid activity type", goerr.V("type", t))
}
