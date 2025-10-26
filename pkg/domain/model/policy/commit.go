package policy

import (
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// CommitPolicyInput represents the input for commit policy evaluation
type CommitPolicyInput struct {
	Alert  *alert.Alert  `json:"alert"`
	Enrich EnrichResults `json:"enrich"`
}

// CommitPolicyResult represents the result of commit policy evaluation
type CommitPolicyResult struct {
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Attr        []alert.Attribute `json:"attr,omitempty"`
	Publish     types.PublishType `json:"publish,omitempty"`
}

// ApplyTo applies the commit policy result to an alert
func (r *CommitPolicyResult) ApplyTo(a *alert.Alert) {
	if r.Title != "" {
		a.Title = r.Title
		a.TitleSource = types.SourcePolicy
	}

	if r.Description != "" {
		a.Description = r.Description
		a.DescriptionSource = types.SourcePolicy
	}

	if r.Channel != "" {
		a.Channel = r.Channel
	}

	if len(r.Attr) > 0 {
		// Mark all attributes from commit policy as auto-generated
		for i := range r.Attr {
			r.Attr[i].Auto = true
		}
		a.Attributes = append(a.Attributes, r.Attr...)
	}
}
