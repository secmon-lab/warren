package prompt_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/prompt"
)

func TestPrompt(t *testing.T) {
	alert := map[string]any{
		"id":      123,
		"message": "test",
	}
	d, err := prompt.BuildInitPrompt(context.Background(), alert, 3)
	gt.NoError(t, err)

	t.Log(d)
	gt.S(t, d).
		Contains("# Alert").
		Contains(`"id": 123`).
		Contains(`"message": "test"`)
}

func TestActionPrompt(t *testing.T) {
	actions := []model.ActionSpec{
		{Name: "action1", Description: "action1 description", Args: []model.ArgumentSpec{{Name: "arg1", Description: "arg1 description", Type: model.ArgumentTypeString, Required: true}}},
	}
	d, err := prompt.BuildActionPrompt(context.Background(), actions)
	gt.NoError(t, err)

	t.Log(d)
	gt.S(t, d).Contains("## `action1`")
	gt.S(t, d).Contains("- `arg1` (string, required): arg1 description")
}

func TestIgnorePolicyPrompt(t *testing.T) {
	policy := map[string]string{
		"test.rego": `package alert.aws.guardduty

alert contains {} # Detected as an alert`,
	}
	alerts := []model.Alert{
		{
			Schema: "aws.guardduty",
			Data:   map[string]any{"Findings": map[string]any{"Severity": 7}},
		},
	}

	policyData := model.PolicyData{
		Data: policy,
	}
	d, err := prompt.BuildIgnorePolicyPrompt(context.Background(), policyData, alerts, "test")
	gt.NoError(t, err)

	gt.S(t, d).Contains("# Rules")
	gt.S(t, d).Contains("## Policy")
	gt.S(t, d).Contains("## Alerts")
	gt.S(t, d).Contains("# Output")
}

func TestMakeGroupPrompt(t *testing.T) {
	alerts := []model.Alert{
		{ID: model.NewAlertID()},
		{ID: model.NewAlertID()},
	}

	d, err := prompt.BuildMakeGroupPrompt(context.Background(), alerts, 10)
	gt.NoError(t, err)

	t.Log(d)
	gt.S(t, d).Contains("# Input")
	gt.S(t, d).Contains("# Output")
}
