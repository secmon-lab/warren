package model

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AlertID string

func (id AlertID) String() string { return string(id) }

type AlertStatus string

const (
	AlertStatusNew     AlertStatus = "new"
	AlertStatusMerged  AlertStatus = "merged"
	AlertStatusPending AlertStatus = "pending"
	AlertStatusClosed  AlertStatus = "closed"
)

type Alert struct {
	ID         AlertID     `json:"id"`
	Schema     string      `json:"schema"`
	Title      string      `json:"title"`
	Status     AlertStatus `json:"status"`
	ParentID   AlertID     `json:"parent_id"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
	Data       any         `json:"data"`
	Attributes []Attribute `json:"attributes"`

	SlackChannel   string `json:"slack_channel"`
	SlackMessageID string `json:"slack_message_id"`
}

func NewAlert(ctx context.Context, schema string, data any, p PolicyAlert) Alert {
	return Alert{
		ID:         AlertID(uuid.New().String()),
		Schema:     schema,
		Title:      p.Title,
		Status:     AlertStatusNew,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Data:       data,
		Attributes: p.Attrs,
	}
}

type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Link  string `json:"link"`
}
