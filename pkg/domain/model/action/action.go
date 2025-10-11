package action

import (
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// PolicyResult represents the result of action policy evaluation
type PolicyResult struct {
	Channel     string            `json:"channel,omitempty"`     // Slack notification channel
	Publish     types.PublishType `json:"publish,omitempty"`     // Publication type
	Attr        map[string]string `json:"attr,omitempty"`        // Additional attributes
	Title       string            `json:"title,omitempty"`       // Override alert title
	Description string            `json:"description,omitempty"` // Override alert description
}

// QueryInput represents the input for action policy evaluation
type QueryInput struct {
	GenAI any          `json:"genai"` // LLM response result
	Alert *alert.Alert `json:"alert"` // Original alert data with filled metadata
}
