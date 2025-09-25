package action

import (
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
)

const (
	PublishTypeDiscard = "discard" // Discard the alert
	PublishTypeNotice  = "notice"  // Send as notice
	PublishTypeAlert   = "alert"   // Send as full alert (default)
)

// PolicyResult represents the result of action policy evaluation
type PolicyResult struct {
	Channel []string          `json:"channel,omitempty"` // Slack notification channels
	Publish string            `json:"publish,omitempty"` // Publication type
	Attr    map[string]string `json:"attr,omitempty"`    // Additional attributes
}

// QueryInput represents the input for action policy evaluation
type QueryInput struct {
	GenAI any          `json:"genai"` // LLM response result
	Alert *alert.Alert `json:"alert"` // Original alert data with filled metadata
}
