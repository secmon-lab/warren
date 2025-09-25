package types

import (
	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
)

type AlertID string

func (x AlertID) String() string {
	return string(x)
}

func NewAlertID() AlertID {
	return AlertID(uuid.New().String())
}

type AlertListID string

func (x AlertListID) String() string {
	return string(x)
}

func NewAlertListID() AlertListID {
	return AlertListID(uuid.New().String())
}

func (x AlertListID) Validate() error {
	if x == EmptyAlertListID {
		return goerr.New("empty alert list ID")
	}
	if _, err := uuid.Parse(string(x)); err != nil {
		return goerr.Wrap(err, "invalid alert list ID format", goerr.V("id", x))
	}
	return nil
}

const (
	EmptyAlertListID AlertListID = ""
)

type AlertSchema string

func (x AlertSchema) String() string {
	return string(x)
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
	AlertConclusionEscalated     AlertConclusion = "escalated"
)

func (r AlertConclusion) Validate() error {
	switch r {
	case AlertConclusionIntended, AlertConclusionUnaffected, AlertConclusionFalsePositive, AlertConclusionTruePositive, AlertConclusionEscalated:
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
	AlertConclusionEscalated:     "⬆️ Escalated",
}

func (r AlertConclusion) Label() string {
	return alertConclusionLabels[r]
}

// NoticeID represents a unique identifier for notice
type NoticeID string

func (x NoticeID) String() string {
	return string(x)
}

func NewNoticeID() NoticeID {
	return NoticeID(uuid.New().String())
}

const (
	EmptyNoticeID NoticeID = ""
)
