package prompt_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/prompt"
	"github.com/secmon-lab/warren/pkg/utils/ptr"
)

func TestSessionStartPrompt(t *testing.T) {
	tests := []struct {
		name   string
		alerts alert.Alerts
	}{
		{
			name: "single alert",
			alerts: alert.Alerts{
				ptr.Ref(alert.New(context.Background(), "aws.guardduty", map[string]any{"Findings": map[string]any{"Severity": 7}}, alert.Metadata{})),
			},
		},
		{
			name:   "no alerts",
			alerts: alert.Alerts{},
		},
		{
			name: "multiple alerts",
			alerts: alert.Alerts{
				ptr.Ref(alert.New(context.Background(), "aws.guardduty", map[string]any{"Findings": map[string]any{"Severity": 7}}, alert.Metadata{})),
				ptr.Ref(alert.New(context.Background(), "aws.guardduty", map[string]any{"Findings": map[string]any{"Severity": 5}}, alert.Metadata{})),
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
