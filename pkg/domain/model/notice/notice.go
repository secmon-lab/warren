package notice

import (
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// Notice represents a simplified alert notification stored in database
type Notice struct {
	ID        types.NoticeID `json:"id"`
	Alert     alert.Alert    `json:"alert"`
	CreatedAt time.Time      `json:"created_at"`
	Escalated bool           `json:"escalated"`
	SlackTS   string         `json:"slack_ts"` // Slack message timestamp for interaction
}

// Notices is a slice of Notice pointers
type Notices []*Notice

// NewNoticeID generates a new notice ID
func NewNoticeID() types.NoticeID {
	return types.NewNoticeID()
}
