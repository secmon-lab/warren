package alert

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

// AlertFinding is the conclusion of the alert. This is set by the AI.
type AlertFinding struct {
	Severity       types.AlertSeverity `json:"severity"`
	Summary        string              `json:"summary"`
	Reason         string              `json:"reason"`
	Recommendation string              `json:"recommendation"`
}

type Alert struct {
	ID          types.AlertID         `json:"id"`
	Schema      string                `json:"schema"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Status      types.AlertStatus     `json:"status"`
	ParentID    types.AlertID         `json:"parent_id"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
	ResolvedAt  *time.Time            `json:"resolved_at"`
	Data        any                   `json:"data"`
	Attributes  []Attribute           `json:"attributes"`
	Conclusion  types.AlertConclusion `json:"conclusion"`
	Reason      string                `json:"reason"`
	Finding     *AlertFinding         `json:"finding"`

	Assignee    *slack.User   `json:"assignee"`
	SlackThread *slack.Thread `json:"slack_thread"`

	Embedding []float32 `json:"-"`
}

type QueryOutput struct {
	Alert []Metadata `json:"alert"`
}

type Metadata struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Data        any         `json:"data"`
	Attrs       []Attribute `json:"attrs"`
}

func NewAlert(ctx context.Context, schema string, metadata Metadata) Alert {
	return Alert{
		ID:          types.NewAlertID(),
		Schema:      schema,
		Title:       metadata.Title,
		Description: metadata.Description,
		Status:      types.AlertStatusNew,
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
