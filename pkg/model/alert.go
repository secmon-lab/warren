package model

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

type AlertID string

func NewAlertID() AlertID {
	return AlertID(uuid.New().String())
}

func (id AlertID) String() string { return string(id) }

type AlertStatus string

const (
	AlertStatusNew          AlertStatus = "new"
	AlertStatusAcknowledged AlertStatus = "acked"
	AlertStatusBlocked      AlertStatus = "blocked"
	AlertStatusResolved     AlertStatus = "resolved"
)

var alertStatusLabels = map[AlertStatus]string{
	AlertStatusNew:          "🆕 New",
	AlertStatusAcknowledged: "👀 Acknowledged",
	AlertStatusBlocked:      "🚫 Blocked",
	AlertStatusResolved:     "✅️ Resolved",
}

func (s AlertStatus) String() string {
	return string(s)
}

func (s AlertStatus) Label() string {
	return alertStatusLabels[s]
}

func (s AlertStatus) Validate() error {
	switch s {
	case AlertStatusNew, AlertStatusAcknowledged, AlertStatusBlocked, AlertStatusResolved:
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

var alertSeverityLabels = map[AlertSeverity]string{
	AlertSeverityUnknown:  "❓️ Unknown",
	AlertSeverityLow:      "🟢 Low",
	AlertSeverityMedium:   "🟡 Medium",
	AlertSeverityHigh:     "🔴 High",
	AlertSeverityCritical: "🚨 Critical",
}

func (s AlertSeverity) Label() string {
	return alertSeverityLabels[s]
}

func (s AlertSeverity) Validate() error {
	switch s {
	case AlertSeverityUnknown, AlertSeverityLow, AlertSeverityMedium, AlertSeverityHigh, AlertSeverityCritical:
		return nil
	}
	return goerr.New("invalid alert severity", goerr.V("severity", s))
}

func (s AlertSeverity) String() string {
	return string(s)
}

// AlertConclusion is the conclusion of the alert. This is set by the user.
type AlertConclusion string

const (
	AlertConclusionIntended      AlertConclusion = "intended"
	AlertConclusionUnaffected    AlertConclusion = "unaffected"
	AlertConclusionFalsePositive AlertConclusion = "false_positive"
	AlertConclusionTruePositive  AlertConclusion = "true_positive"
)

func (r AlertConclusion) Validate() error {
	switch r {
	case AlertConclusionIntended, AlertConclusionUnaffected, AlertConclusionFalsePositive, AlertConclusionTruePositive:
		return nil
	}
	return goerr.New("invalid alert result", goerr.V("result", r))
}

func (r AlertConclusion) String() string {
	return string(r)
}

var alertConclusionLabels = map[AlertConclusion]string{
	AlertConclusionIntended:      "👍 Intended",
	AlertConclusionUnaffected:    "🛡️ Unaffected",
	AlertConclusionFalsePositive: "🚫 False Positive",
	AlertConclusionTruePositive:  "🚨 True Positive",
}

func (r AlertConclusion) Label() string {
	return alertConclusionLabels[r]
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
	ResolvedAt  *time.Time      `json:"resolved_at"`
	Data        any             `json:"data"`
	Attributes  []Attribute     `json:"attributes"`
	Conclusion  AlertConclusion `json:"conclusion"`
	Reason      string          `json:"reason"`
	Finding     *AlertFinding   `json:"finding"`

	Assignee    *SlackUser   `json:"assignee"`
	SlackThread *SlackThread `json:"slack_thread"`
	Embedding   []float32    `json:"-"`
}

type SlackThread struct {
	TeamID    string `json:"team_id"`
	ChannelID string `json:"channel_id"`
	ThreadID  string `json:"thread_id"`
}

type SlackUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func NewAlert(ctx context.Context, schema string, p PolicyAlert) Alert {
	return Alert{
		ID:         NewAlertID(),
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
	Auto  bool   `json:"auto"`
}

type AlertComment struct {
	AlertID   AlertID `json:"alert_id"`
	Timestamp string  `json:"timestamp"`
	Comment   string  `json:"comment"`
	UserID    string  `json:"user_id"`
}

type AlertListID string

func NewAlertListID() AlertListID {
	return AlertListID(uuid.New().String())
}

func (id AlertListID) String() string {
	return string(id)
}

type AlertList struct {
	ID          AlertListID  `json:"id"`
	AlertIDs    []AlertID    `json:"alert_ids"`
	SlackThread *SlackThread `json:"slack_thread"`
	CreatedAt   time.Time    `json:"created_at"`
	CreatedBy   *SlackUser   `json:"created_by"`

	Alerts []Alert `firestore:"-"`
}

func NewAlertList(ctx context.Context, thread SlackThread, createdBy *SlackUser, alerts []Alert) AlertList {
	alertList := AlertList{
		ID:          NewAlertListID(),
		SlackThread: &thread,
		CreatedAt:   clock.Now(ctx),
		CreatedBy:   createdBy,
	}
	for _, alert := range alerts {
		alertList.AlertIDs = append(alertList.AlertIDs, alert.ID)
		alertList.Alerts = append(alertList.Alerts, alert)
	}

	return alertList
}
