package policy

import (
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// TriagePolicyAlert represents alert data for triage policy input (matches doc/policy.md format)
type TriagePolicyAlert struct {
	ID       types.AlertID     `json:"id"`
	Schema   types.AlertSchema `json:"schema"`
	Metadata alert.Metadata    `json:"metadata"`
	Data     any               `json:"data"`
}

// TriagePolicyInput represents the input for triage policy evaluation
type TriagePolicyInput struct {
	Alert  TriagePolicyAlert `json:"alert"`
	Enrich EnrichResults     `json:"enrich"`
}

// NewTriagePolicyInput creates a TriagePolicyInput from an Alert
func NewTriagePolicyInput(a *alert.Alert, enrichResults EnrichResults) TriagePolicyInput {
	return TriagePolicyInput{
		Alert: TriagePolicyAlert{
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

// TriagePolicyResult represents the result of triage policy evaluation
type TriagePolicyResult struct {
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Attr        []alert.Attribute `json:"attr,omitempty"`
	Publish     types.PublishType `json:"publish,omitempty"`
}

// ApplyTo applies the triage policy result to an alert
func (r *TriagePolicyResult) ApplyTo(a *alert.Alert) {
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
		// Mark all attributes from triage policy as auto-generated
		for i := range r.Attr {
			r.Attr[i].Auto = true
		}
		a.Attributes = append(a.Attributes, r.Attr...)
	}
}
