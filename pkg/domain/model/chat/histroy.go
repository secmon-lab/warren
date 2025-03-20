package chat

import (
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type History struct {
	ID        types.HistoryID  `json:"id"`
	Thread    slack.Thread     `json:"thread"`
	CreatedBy slack.User       `json:"created_by"`
	CreatedAt time.Time        `json:"created_at"`
	Contents  []*genai.Content `json:"contents"`
}
