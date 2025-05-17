package alert

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

// Alert represents an event of a potential security incident. This model is designed to be immutable. An Alert can be linked to at most one ticket.
type Alert struct {
	ID        types.AlertID     `json:"id"`
	TicketID  types.TicketID    `json:"ticket_id"`
	Schema    types.AlertSchema `json:"schema"`
	CreatedAt time.Time         `json:"created_at"`

	SlackThread *slack.Thread `json:"slack_thread"`

	Embedding []float32 `json:"-"`

	Metadata
}

type Alerts []*Alert

type QueryOutput struct {
	Alert []Metadata `json:"alert"`
}

type Metadata struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Data        any         `json:"data"`
	Attrs       []Attribute `json:"attrs"`
}

func New(ctx context.Context, schema types.AlertSchema, metadata Metadata) Alert {
	return Alert{
		ID:        types.NewAlertID(),
		Schema:    schema,
		CreatedAt: clock.Now(ctx),
		Metadata:  metadata,
	}
}

type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Link  string `json:"link"`
	Auto  bool   `json:"auto"`
}
