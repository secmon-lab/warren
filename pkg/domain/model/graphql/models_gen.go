// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package graphql

import (
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
)

type ActivitiesResponse struct {
	Activities []*Activity `json:"activities"`
	TotalCount int         `json:"totalCount"`
}

type Activity struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	UserID    *string        `json:"userID,omitempty"`
	AlertID   *string        `json:"alertID,omitempty"`
	TicketID  *string        `json:"ticketID,omitempty"`
	CommentID *string        `json:"commentID,omitempty"`
	CreatedAt string         `json:"createdAt"`
	Metadata  *string        `json:"metadata,omitempty"`
	User      *User          `json:"user,omitempty"`
	Alert     *alert.Alert   `json:"alert,omitempty"`
	Ticket    *ticket.Ticket `json:"ticket,omitempty"`
}

type AlertAttribute struct {
	Key   string  `json:"key"`
	Value string  `json:"value"`
	Link  *string `json:"link,omitempty"`
	Auto  bool    `json:"auto"`
}

type AlertCluster struct {
	ID          string         `json:"id"`
	CenterAlert *alert.Alert   `json:"centerAlert"`
	Alerts      []*alert.Alert `json:"alerts"`
	Size        int            `json:"size"`
	Keywords    []string       `json:"keywords,omitempty"`
	CreatedAt   string         `json:"createdAt"`
}

type AlertsConnection struct {
	Alerts     []*alert.Alert `json:"alerts"`
	TotalCount int            `json:"totalCount"`
}

type AlertsResponse struct {
	Alerts     []*alert.Alert `json:"alerts"`
	TotalCount int            `json:"totalCount"`
}

type ClusteringSummary struct {
	Clusters    []*AlertCluster   `json:"clusters"`
	NoiseAlerts []*alert.Alert    `json:"noiseAlerts"`
	Parameters  *DBSCANParameters `json:"parameters"`
	ComputedAt  string            `json:"computedAt"`
	TotalCount  int               `json:"totalCount"`
}

type CommentsResponse struct {
	Comments   []*ticket.Comment `json:"comments"`
	TotalCount int               `json:"totalCount"`
}

type DBSCANParameters struct {
	Eps        float64 `json:"eps"`
	MinSamples int     `json:"minSamples"`
}

type DashboardStats struct {
	OpenTicketsCount   int              `json:"openTicketsCount"`
	UnboundAlertsCount int              `json:"unboundAlertsCount"`
	OpenTickets        []*ticket.Ticket `json:"openTickets"`
	UnboundAlerts      []*alert.Alert   `json:"unboundAlerts"`
}

type Mutation struct {
}

type Query struct {
}

type TicketsResponse struct {
	Tickets    []*ticket.Ticket `json:"tickets"`
	TotalCount int              `json:"totalCount"`
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
