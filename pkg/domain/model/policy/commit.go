package policy

import (
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// CommitPolicyAlert represents alert data for commit policy input (matches doc/policy.md format)
type CommitPolicyAlert struct {
	ID       types.AlertID     `json:"id"`
	Schema   types.AlertSchema `json:"schema"`
	Metadata alert.Metadata    `json:"metadata"`
	Data     any               `json:"data"`
}

// CommitPolicyInput represents the input for commit policy evaluation
type CommitPolicyInput struct {
	Alert  CommitPolicyAlert `json:"alert"`
	Enrich EnrichResults     `json:"enrich"`
}

// NewCommitPolicyInput creates a CommitPolicyInput from an Alert
func NewCommitPolicyInput(a *alert.Alert, enrichResults EnrichResults) CommitPolicyInput {
	return CommitPolicyInput{
		Alert: CommitPolicyAlert{
			ID:     a.ID,
			Schema: a.Schema,
			Metadata: alert.Metadata{
				Title:             a.Title,
				Description:       a.Description,
				Attributes:        a.Attributes,
				TitleSource:       a.TitleSource,
				DescriptionSource: a.DescriptionSource,
				Tags:              a.Tags,
				Channel:           a.Channel,
			},
			Data: a.Data,
		},
		Enrich: enrichResults,
	}
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
