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

const (
	EmptyAlertListID AlertListID = ""
)

type AlertSchema string

func (x AlertSchema) String() string {
	return string(x)
}

type AlertStatus string

const (
	AlertStatusUnknown      AlertStatus = "unknown"
	AlertStatusNew          AlertStatus = "new"
	AlertStatusAcknowledged AlertStatus = "acked"
	AlertStatusBlocked      AlertStatus = "blocked"
	AlertStatusResolved     AlertStatus = "resolved"
)

var alertStatusLabels = map[AlertStatus]string{
	AlertStatusUnknown:      "❓️ Unknown",
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
