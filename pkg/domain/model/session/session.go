package session

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

type Session struct {
	ID         types.SessionID   `json:"id"`
	CreatedAt  time.Time         `json:"created_at"`
	CreatedBy  slack.User        `json:"created_by"`
	Thread     slack.Thread      `json:"thread"`
	RootListID types.AlertListID `json:"root_list_id"`
}

func New(ctx context.Context, createdBy slack.User, thread slack.Thread, rootListID types.AlertListID) *Session {
	return &Session{
		ID:         types.NewSessionID(),
		CreatedAt:  clock.Now(ctx),
		CreatedBy:  createdBy,
		Thread:     thread,
		RootListID: rootListID,
	}
}
