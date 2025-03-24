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

func TestSessionStartPrompt(t *testing.T) {
	tests := []struct {
		name   string
		alerts alert.Alerts
	}{
		{
			name: "single alert",
			alerts: alert.Alerts{
				ptr.Ref(alert.New(context.Background(), "aws.guardduty", alert.Metadata{
					Data: map[string]any{"Findings": map[string]any{"Severity": 7}},
				})),
			},
		},
		{
			name:   "no alerts",
			alerts: alert.Alerts{},
		},
		{
			name: "multiple alerts",
			alerts: alert.Alerts{
				ptr.Ref(alert.New(context.Background(), "aws.guardduty", alert.Metadata{
					Data: map[string]any{"Findings": map[string]any{"Severity": 7}},
				})),
				ptr.Ref(alert.New(context.Background(), "aws.guardduty", alert.Metadata{
					Data: map[string]any{"Findings": map[string]any{"Severity": 5}},
				})),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			d, err := prompt.BuildSessionStartPrompt(ctx, tt.alerts)
			gt.NoError(t, err)
			t.Log(d)
		})
	}
}

func TestSessionNextPrompt(t *testing.T) {
	tests := []struct {
		name   string
		result *action.Result
	}{
		{
			name: "single action",
			result: &action.Result{
				Message: "test",
				Type:    action.ResultTypeText,
				Rows:    []string{"test"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			d, err := prompt.BuildSessionNextPrompt(ctx, tt.result)
			gt.NoError(t, err)
			t.Log(d)
		})
	}
}
