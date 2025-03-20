package alert

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

func NewAlertID() types.AlertID {
	return types.AlertID(uuid.New().String())
}

type Status string

const (
	StatusNew          Status = "new"
	StatusAcknowledged Status = "acked"
	StatusBlocked      Status = "blocked"
	StatusResolved     Status = "resolved"
)

var statusLabels = map[Status]string{
	StatusNew:          "🆕 New",
	StatusAcknowledged: "👀 Acknowledged",
	StatusBlocked:      "🚫 Blocked",
	StatusResolved:     "✅️ Resolved",
}

func (s Status) String() string {
	return string(s)
}

func (s Status) Label() string {
	return statusLabels[s]
}

func (s Status) Validate() error {
	switch s {
	case StatusNew, StatusAcknowledged, StatusBlocked, StatusResolved:
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
	ID          types.AlertID   `json:"id"`
	Schema      string          `json:"schema"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      Status          `json:"status"`
	ParentID    types.AlertID   `json:"parent_id"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ResolvedAt  *time.Time      `json:"resolved_at"`
	Data        any             `json:"data"`
	Attributes  []Attribute     `json:"attributes"`
	Conclusion  AlertConclusion `json:"conclusion"`
	Reason      string          `json:"reason"`
	Finding     *AlertFinding   `json:"finding"`

	Assignee    *slack.User   `json:"assignee"`
	SlackThread *slack.Thread `json:"slack_thread"`

	Embedding []float32 `json:"-"`
}

type Metadata struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Data        any         `json:"data"`
	Attrs       []Attribute `json:"attrs"`
}

func NewAlert(ctx context.Context, schema string, metadata Metadata) Alert {
	return Alert{
		ID:          NewAlertID(),
		Schema:      schema,
		Title:       metadata.Title,
		Description: metadata.Description,
		Status:      StatusNew,
		CreatedAt:   clock.Now(ctx),
		UpdatedAt:   clock.Now(ctx),
		Data:        metadata.Data,
		Attributes:  metadata.Attrs,
	}
}

type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Link  string `json:"link"`
	Auto  bool   `json:"auto"`
}

type AlertComment struct {
	AlertID   types.AlertID `json:"alert_id"`
	Timestamp string        `json:"timestamp"`
	Comment   string        `json:"comment"`
	User      slack.User    `json:"user"`
}
