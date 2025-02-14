package model

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

type AlertID string

func (id AlertID) String() string { return string(id) }

type AlertStatus string

const (
	AlertStatusNew          AlertStatus = "new"
	AlertStatusAcknowledged AlertStatus = "acked"
	AlertStatusMerged       AlertStatus = "merged"
	AlertStatusClosed       AlertStatus = "closed"
)

func (s AlertStatus) Validate() error {
	switch s {
	case AlertStatusNew, AlertStatusAcknowledged, AlertStatusMerged, AlertStatusClosed:
		return nil
	}
	return goerr.New("invalid alert status", goerr.V("status", s))
}

// AlertSeverity is the severity of the alert. This is set by the AI.
type AlertSeverity string

const (
	AlertSeverityUnknown  AlertSeverity = "unknown"
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityCritical AlertSeverity = "critical"
)

func (s AlertSeverity) Validate() error {
	switch s {
	case AlertSeverityUnknown, AlertSeverityLow, AlertSeverityMedium, AlertSeverityHigh, AlertSeverityCritical:
		return nil
	}
	return goerr.New("invalid alert severity", goerr.V("severity", s))
}

// AlertConclusion is the conclusion of the alert. This is set by the user.
type AlertConclusion string

const (
	AlertConclusionUnknown       AlertConclusion = "unknown"
	AlertConclusionUnaffected    AlertConclusion = "unaffected"
	AlertConclusionFalsePositive AlertConclusion = "false_positive"
	AlertConclusionTruePositive  AlertConclusion = "true_positive"
)

func (r AlertConclusion) Validate() error {
	switch r {
	case AlertConclusionUnknown, AlertConclusionUnaffected, AlertConclusionFalsePositive, AlertConclusionTruePositive:
		return nil
	}
	return goerr.New("invalid alert result", goerr.V("result", r))
}

// AlertFinding is the conclusion of the alert. This is set by the AI.
type AlertFinding struct {
	Severity       AlertSeverity `json:"severity"`
	Summary        string        `json:"summary"`
	Reason         string        `json:"reason"`
	Recommendation string        `json:"recommendation"`
}

type Alert struct {
	ID          AlertID         `json:"id"`
	Schema      string          `json:"schema"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      AlertStatus     `json:"status"`
	ParentID    AlertID         `json:"parent_id"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ClosedAt    *time.Time      `json:"closed_at"`
	Data        any             `json:"data"`
	Attributes  []Attribute     `json:"attributes"`
	Conclusion  AlertConclusion `json:"conclusion"`
	Finding     *AlertFinding   `json:"finding"`

	Assignee    *SlackUser   `json:"assignee"`
	SlackThread *SlackThread `json:"slack_thread"`
}

type SlackThread struct {
	ChannelID string `json:"channel_id"`
	ThreadID  string `json:"thread_id"`
}

type SlackUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func NewAlert(ctx context.Context, schema string, p PolicyAlert) Alert {
	return Alert{
		ID:         AlertID(uuid.New().String()),
		Schema:     schema,
		Title:      p.Title,
		Status:     AlertStatusNew,
		CreatedAt:  clock.Now(ctx),
		UpdatedAt:  clock.Now(ctx),
		Data:       p.Data,
		Attributes: p.Attrs,
	}
}

type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Link  string `json:"link"`
}
