package prompt_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/prompt"
	"github.com/secmon-lab/warren/pkg/utils/ptr"
)

func TestMetaListPrompt(t *testing.T) {
	ctx := context.Background()
	alerts := alert.Alerts{
		ptr.Ref(alert.New(ctx, "aws.guardduty", alert.Metadata{
			Data: map[string]any{"Findings": map[string]any{"Severity": 7}},
		})),
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
			d, err := prompt.BuildSessionInitPrompt(ctx, tt.alerts)
			gt.NoError(t, err)
			t.Log(d)
		})
	}
}
