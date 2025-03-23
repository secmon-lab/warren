package prompt_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/prompt"
	"github.com/secmon-lab/warren/pkg/utils/ptr"
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
	actions := []action.ActionSpec{
		{Name: "action1", Description: "action1 description", Args: []action.ArgumentSpec{{Name: "arg1", Description: "arg1 description", Type: action.ArgumentTypeString, Required: true}}},
	}
	d, err := prompt.BuildActionPrompt(context.Background(), actions)
	gt.NoError(t, err)

	t.Log(d)
	gt.S(t, d).Contains("## `action1`")
	gt.S(t, d).Contains("- `arg1` (string, required): arg1 description")
}

func TestIgnorePolicyPrompt(t *testing.T) {
	contents := policy.Contents{
		"test.rego": `package alert.aws.guardduty

alert contains {} # Detected as an alert`,
	}
	alerts := alert.Alerts{
		{
			Schema: "aws.guardduty",
			Data:   map[string]any{"Findings": map[string]any{"Severity": 7}},
		},
	}

	d, err := prompt.BuildIgnorePolicyPrompt(context.Background(), contents, alerts, "test")
	gt.NoError(t, err)

	gt.S(t, d).Contains("# Rules")
	gt.S(t, d).Contains("## Policy")
	gt.S(t, d).Contains("## Alerts")
	gt.S(t, d).Contains("# Output")
}

func TestTestDataReadmePrompt(t *testing.T) {
	ctx := context.Background()
	alerts := alert.Alerts{
		ptr.Ref(alert.New(ctx, "aws.guardduty", alert.Metadata{
			Data: map[string]any{"Findings": map[string]any{"Severity": 7}},
		})),
		ptr.Ref(alert.New(ctx, "aws.guardduty", alert.Metadata{
			Data: map[string]any{"Findings": map[string]any{"Severity": 7}},
		})),
	}

	d, err := prompt.BuildTestDataReadmePrompt(ctx, "ignore", alerts)
	gt.NoError(t, err)

	t.Log(d)
}

func TestFilterQueryPrompt(t *testing.T) {
	ctx := context.Background()
	alerts := []alert.Alert{
		alert.New(ctx, "aws.guardduty", alert.Metadata{
			Data: map[string]any{"Findings": map[string]any{"Severity": 7}},
		}),
		alert.New(ctx, "aws.guardduty", alert.Metadata{
			Data: map[string]any{"Findings": map[string]any{"Severity": 7}},
		}),
	}

	d, err := prompt.BuildFilterQueryPrompt(ctx, "test", alerts)
	gt.NoError(t, err)

	t.Log(d)
}

func TestMetaListPrompt(t *testing.T) {
	ctx := context.Background()
	alerts := []alert.Alert{
		alert.New(ctx, "aws.guardduty", alert.Metadata{
			Data: map[string]any{"Findings": map[string]any{"Severity": 7}},
		}),
	}

	alertList := alert.NewList(ctx, slack.Thread{
		ChannelID: "C0123456789",
		ThreadID:  "T0123456789",
	}, &slack.User{
		ID:   "U0123456789",
		Name: "John Doe",
	}, alerts)

	d, err := prompt.BuildMetaListPrompt(ctx, alertList)
	gt.NoError(t, err)

	t.Log(d)
}
