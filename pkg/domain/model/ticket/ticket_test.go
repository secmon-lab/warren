package ticket_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func TestTicket_FillMetadata(t *testing.T) {
	repo := repository.NewMemory()
	llmClient := test.NewGeminiClient(t)
	ctx := t.Context()

	alerts := alert.Alerts{
		{
			ID: types.NewAlertID(),
			Metadata: alert.Metadata{
				Title:       "Test Alert",
				Description: "Test Description",
			},
			Data: map[string]any{
				"indicator": "192.168.1.1",
				"asset":     "192.168.1.1",
				"context":   "Test Context",
				"correlation": map[string]any{
					"indicator": "192.168.1.1",
					"asset":     "192.168.1.1",
				},
			},
		},
		{
			ID: types.NewAlertID(),
			Metadata: alert.Metadata{
				Title:       "Test Alert",
				Description: "Test Description",
			},
			Data: map[string]any{
				"indicator": "192.168.1.1",
				"asset":     "192.168.1.1",
				"context":   "Test Context",
				"correlation": map[string]any{
					"indicator": "192.168.1.1",
					"asset":     "192.168.1.1",
				},
			},
		},
	}
	for _, alert := range alerts {
		gt.NoError(t, repo.PutAlert(ctx, *alert))
	}

	ticketData := ticket.New(ctx, []types.AlertID{alerts[0].ID, alerts[1].ID}, &slack.Thread{})

	if err := ticketData.FillMetadata(ctx, llmClient, repo); err != nil {
		t.Fatalf("failed to fill metadata: %v", err)
	}

	gt.NotEqual(t, ticketData.Metadata.Title, "")
	gt.NotEqual(t, ticketData.Metadata.Description, "")
	gt.NotEqual(t, ticketData.Metadata.Summary, "")
}
