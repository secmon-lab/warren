package base_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/tool/base"
)

func TestWarren_UpdateFinding(t *testing.T) {
	repo := repository.NewMemory()
	ctx := context.Background()

	// Create a test ticket
	testTicket := ticket.Ticket{
		ID:     types.NewTicketID(),
		Status: types.TicketStatusOpen,
		Metadata: ticket.Metadata{
			Title:       "Test Ticket",
			Description: "Test Description",
			Summary:     "Test Summary",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save ticket to repository
	gt.NoError(t, repo.PutTicket(ctx, testTicket))

	// Create Warren tool
	warren := base.New(repo, nil, testTicket.ID)

	t.Run("successful finding update", func(t *testing.T) {
		args := map[string]any{
			"summary":        "Malicious activity detected in network logs",
			"severity":       "high",
			"reason":         "Multiple failed login attempts followed by successful login from unusual location",
			"recommendation": "Review user account and consider temporary suspension",
		}

		result, err := warren.Run(ctx, "warren.update_finding", args)
		gt.NoError(t, err)

		// Verify result structure
		gt.Value(t, result["success"]).Equal(true)
		gt.Value(t, result["severity"]).Equal("high")
		gt.Value(t, result["summary"]).Equal("Malicious activity detected in network logs")
		gt.Value(t, result["reason"]).Equal("Multiple failed login attempts followed by successful login from unusual location")
		gt.Value(t, result["recommendation"]).Equal("Review user account and consider temporary suspension")

		// Verify ticket was updated in database
		updatedTicket, err := repo.GetTicket(ctx, testTicket.ID)
		gt.NoError(t, err)
		gt.NotNil(t, updatedTicket.Finding)
		gt.Value(t, updatedTicket.Finding.Severity).Equal(types.AlertSeverityHigh)
		gt.Value(t, updatedTicket.Finding.Summary).Equal("Malicious activity detected in network logs")
		gt.Value(t, updatedTicket.Finding.Reason).Equal("Multiple failed login attempts followed by successful login from unusual location")
		gt.Value(t, updatedTicket.Finding.Recommendation).Equal("Review user account and consider temporary suspension")
	})

	t.Run("invalid severity", func(t *testing.T) {
		args := map[string]any{
			"summary":        "Test summary",
			"severity":       "invalid",
			"reason":         "Test reason",
			"recommendation": "Test recommendation",
		}

		_, err := warren.Run(ctx, "warren.update_finding", args)
		gt.Error(t, err)
	})

	t.Run("missing required field", func(t *testing.T) {
		args := map[string]any{
			"summary":  "Test summary",
			"severity": "low",
			"reason":   "Test reason",
			// missing recommendation
		}

		_, err := warren.Run(ctx, "warren.update_finding", args)
		gt.Error(t, err)
	})
}

func TestWarren_UpdateFindingWithSlackCallback(t *testing.T) {
	repo := repository.NewMemory()
	ctx := context.Background()

	// Create a test ticket with slack thread
	testTicket := ticket.Ticket{
		ID:     types.NewTicketID(),
		Status: types.TicketStatusOpen,
		Metadata: ticket.Metadata{
			Title:       "Test Ticket",
			Description: "Test Description",
			Summary:     "Test Summary",
		},
		SlackThread: &slack.Thread{
			TeamID:    "T1234567890",
			ChannelID: "C1234567890",
			ThreadID:  "1234567890.123456",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save ticket to repository
	gt.NoError(t, repo.PutTicket(ctx, testTicket))

	// Create mock Slack update function
	slackUpdateCalled := false
	slackUpdateFunc := func(ctx context.Context, ticket *ticket.Ticket) error {
		slackUpdateCalled = true
		return nil
	}

	// Create Warren tool with Slack callback
	warren := base.New(repo, nil, testTicket.ID, base.WithSlackUpdate(slackUpdateFunc))

	t.Run("slack update callback called", func(t *testing.T) {
		args := map[string]any{
			"summary":        "Test finding with Slack update",
			"severity":       "medium",
			"reason":         "Suspicious but not critical activity",
			"recommendation": "Monitor closely",
		}

		result, err := warren.Run(ctx, "warren.update_finding", args)
		gt.NoError(t, err)

		// Verify Slack update was called
		gt.Value(t, slackUpdateCalled).Equal(true)
		gt.Value(t, result["slack_updated"]).Equal(true)
	})
}

func TestWarren_Specs(t *testing.T) {
	repo := repository.NewMemory()
	ctx := context.Background()

	// Create Warren tool
	warren := base.New(repo, nil, types.NewTicketID())

	specs, err := warren.Specs(ctx)
	gt.NoError(t, err)

	// Find the update_finding command spec
	var updateFindingSpec *gollem.ToolSpec
	for i, spec := range specs {
		if spec.Name == "warren.update_finding" {
			updateFindingSpec = &specs[i]
			break
		}
	}

	gt.NotNil(t, updateFindingSpec)
	gt.Value(t, updateFindingSpec.Name).Equal("warren.update_finding")

	// Check description contains expected text
	description := updateFindingSpec.Description
	if description == "" {
		t.Error("Description should not be empty")
	}

	// Check required parameters
	requiredParams := []string{"summary", "severity", "reason", "recommendation"}
	for _, param := range requiredParams {
		gt.NotNil(t, updateFindingSpec.Parameters[param])
	}

	gt.Value(t, len(updateFindingSpec.Required)).Equal(4)
	for _, required := range requiredParams {
		found := false
		for _, req := range updateFindingSpec.Required {
			if req == required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Required parameter %s not found", required)
		}
	}
}
