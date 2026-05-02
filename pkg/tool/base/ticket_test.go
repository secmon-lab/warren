package base_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/tool/base"
	"github.com/secmon-lab/warren/pkg/utils/dryrun"
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
	warren := base.New(repo, testTicket.ID)

	t.Run("successful finding update", func(t *testing.T) {
		args := map[string]any{
			"summary":        "Malicious activity detected in network logs",
			"severity":       "high",
			"reason":         "Multiple failed login attempts followed by successful login from unusual location",
			"recommendation": "Review user account and consider temporary suspension",
		}

		result, err := warren.Run(ctx, "warren_update_finding", args)
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

		_, err := warren.Run(ctx, "warren_update_finding", args)
		gt.Error(t, err)
	})

	t.Run("missing required field", func(t *testing.T) {
		args := map[string]any{
			"summary":  "Test summary",
			"severity": "low",
			"reason":   "Test reason",
			// missing recommendation
		}

		_, err := warren.Run(ctx, "warren_update_finding", args)
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
	warren := base.New(repo, testTicket.ID, base.WithSlackUpdate(slackUpdateFunc))

	t.Run("slack update callback called", func(t *testing.T) {
		args := map[string]any{
			"summary":        "Test finding with Slack update",
			"severity":       "medium",
			"reason":         "Suspicious but not critical activity",
			"recommendation": "Monitor closely",
		}

		result, err := warren.Run(ctx, "warren_update_finding", args)
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
	warren := base.New(repo, types.NewTicketID())

	specs, err := warren.Specs(ctx)
	gt.NoError(t, err)

	// Verify we have the expected number of specs
	expectedCommands := []string{
		"warren_get_alerts",
		"warren_find_nearest_ticket",
		"warren_search_tickets_by_words",
		"warren_update_finding",
		"warren_get_ticket_session_messages",
		"warren_search_session_messages",
	}
	gt.Value(t, len(specs)).Equal(len(expectedCommands))

	var updateFindingSpec *gollem.ToolSpec
	for i, spec := range specs {
		if spec.Name == "warren_update_finding" {
			updateFindingSpec = &specs[i]
		}
	}

	gt.NotNil(t, updateFindingSpec)
	gt.Value(t, updateFindingSpec.Name).Equal("warren_update_finding")

	description := updateFindingSpec.Description
	if description == "" {
		t.Error("Description should not be empty")
	}

	requiredParams := []string{"summary", "severity", "reason", "recommendation"}
	for _, param := range requiredParams {
		gt.NotNil(t, updateFindingSpec.Parameters[param])
		gt.True(t, updateFindingSpec.Parameters[param].Required)
	}
}

func TestWarren_UpdateFindingDryRun(t *testing.T) {
	runTest := func(tc struct {
		name              string
		isDryRun          bool
		expectRepoUpdate  bool
		expectSlackUpdate bool
	}) func(t *testing.T) {
		return func(t *testing.T) {
			// Setup mock repository
			mockRepo := &mock.RepositoryMock{}

			ticketID := types.TicketID("test-ticket-id")

			// Create test ticket with Slack thread to enable Slack updates
			testTicket := &ticket.Ticket{
				ID: ticketID,
				Metadata: ticket.Metadata{
					Title:       "Test Ticket",
					Description: "Test Description",
					Summary:     "Test Summary",
				},
				Status: types.TicketStatusOpen,
				SlackThread: &slack.Thread{
					TeamID:    "T1234567890",
					ChannelID: "C1234567890",
					ThreadID:  "1234567890.123456",
				},
			}

			// Setup mock expectations
			mockRepo.GetTicketFunc = func(ctx context.Context, id types.TicketID) (*ticket.Ticket, error) {
				return testTicket, nil
			}

			var repoUpdateCalled bool
			var slackUpdateCalled bool

			mockRepo.PutTicketFunc = func(ctx context.Context, t ticket.Ticket) error {
				repoUpdateCalled = true
				return nil
			}

			slackUpdateFunc := func(ctx context.Context, ticket *ticket.Ticket) error {
				slackUpdateCalled = true
				return nil
			}

			// Create Warren tool with slack update callback
			warren := base.New(mockRepo, ticketID, base.WithSlackUpdate(slackUpdateFunc))

			// Create context with dry-run setting
			ctx := context.Background()
			if tc.isDryRun {
				ctx = dryrun.With(ctx, true)
			}

			// Test update_finding tool call
			args := map[string]any{
				"summary":        "Test finding summary",
				"severity":       "high",
				"reason":         "Test reason for finding",
				"recommendation": "Test recommendation",
			}

			_, err := warren.Run(ctx, "warren_update_finding", args)
			gt.NoError(t, err)

			// Verify expectations
			gt.Equal(t, tc.expectRepoUpdate, repoUpdateCalled)
			gt.Equal(t, tc.expectSlackUpdate, slackUpdateCalled)
		}
	}

	t.Run("dry-run enabled", runTest(struct {
		name              string
		isDryRun          bool
		expectRepoUpdate  bool
		expectSlackUpdate bool
	}{
		name:              "dry-run enabled",
		isDryRun:          true,
		expectRepoUpdate:  false, // Should not update repository
		expectSlackUpdate: false, // Should not update Slack
	}))

	t.Run("dry-run disabled", runTest(struct {
		name              string
		isDryRun          bool
		expectRepoUpdate  bool
		expectSlackUpdate bool
	}{
		name:              "dry-run disabled",
		isDryRun:          false,
		expectRepoUpdate:  true, // Should update repository
		expectSlackUpdate: true, // Should update Slack
	}))
}

func TestWarren_SearchTicketsByWords(t *testing.T) {
	repo := repository.NewMemory()
	ctx := context.Background()

	// Create test embeddings with proper dimensions (256)
	embedding1 := make(firestore.Vector32, 256)
	embedding2 := make(firestore.Vector32, 256)
	queryEmbedding := make([]float64, 256)

	// Fill with test values - make first embedding more similar to query
	for i := 0; i < 256; i++ {
		embedding1[i] = 0.1 + float32(i)*0.001
		embedding2[i] = 0.5 + float32(i)*0.001
		queryEmbedding[i] = 0.15 + float64(i)*0.001 // Similar to embedding1
	}

	// Create a test ticket with embedding
	now := time.Now()
	testTicket := ticket.Ticket{
		ID: types.NewTicketID(),
		Metadata: ticket.Metadata{
			Title:       "Test security incident",
			Description: "A suspicious malware detection alert",
		},
		Embedding: embedding1,
		CreatedAt: now.Add(-time.Hour), // Created 1 hour ago
		UpdatedAt: now.Add(-time.Hour),
	}
	gt.NoError(t, repo.PutTicket(ctx, testTicket))

	// Create another ticket for comparison
	testTicket2 := ticket.Ticket{
		ID: types.NewTicketID(),
		Metadata: ticket.Metadata{
			Title:       "Network anomaly",
			Description: "Unusual network traffic detected",
		},
		Embedding: embedding2,
		CreatedAt: now.Add(-2 * time.Hour), // Created 2 hours ago
		UpdatedAt: now.Add(-2 * time.Hour),
	}
	gt.NoError(t, repo.PutTicket(ctx, testTicket2))

	// Create mock LLM client that returns a test embedding
	mockLLM := &mock.LLMClientMock{}
	mockLLM.GenerateEmbeddingFunc = func(ctx context.Context, dimension int, inputs []string) ([][]float64, error) {
		// Return embedding similar to first ticket
		return [][]float64{queryEmbedding}, nil
	}

	warren := base.New(repo, testTicket.ID, base.WithLLMClient(mockLLM))

	t.Run("search with query", func(t *testing.T) {
		// Use actual JSON unmarshalling to simulate real usage
		jsonData := `{
			"query": "malware security incident",
			"limit": 5,
			"duration": 30
		}`
		var args map[string]any
		gt.NoError(t, json.Unmarshal([]byte(jsonData), &args))

		result, err := warren.Run(ctx, "warren_search_tickets_by_words", args)
		gt.NoError(t, err)

		// Check response structure
		tickets, ok := result["tickets"].([]any)
		gt.Value(t, ok).Equal(true)
		gt.Value(t, len(tickets) > 0).Equal(true)

		query, ok := result["query"].(string)
		gt.Value(t, ok).Equal(true)
		gt.Value(t, query).Equal("malware security incident")

		count, ok := result["count"].(int)
		gt.Value(t, ok).Equal(true)
		gt.Value(t, count).Equal(len(tickets))
	})

	t.Run("missing query parameter", func(t *testing.T) {
		// Use actual JSON unmarshalling - missing query field
		jsonData := `{
			"limit": 5
		}`
		var args map[string]any
		gt.NoError(t, json.Unmarshal([]byte(jsonData), &args))

		_, err := warren.Run(ctx, "warren_search_tickets_by_words", args)
		gt.Error(t, err)
	})

	t.Run("no LLM client configured", func(t *testing.T) {
		warrenWithoutLLM := base.New(repo, testTicket.ID)
		jsonData := `{
			"query": "test query"
		}`
		var args map[string]any
		gt.NoError(t, json.Unmarshal([]byte(jsonData), &args))

		_, err := warrenWithoutLLM.Run(ctx, "warren_search_tickets_by_words", args)
		gt.Error(t, err)
	})
}
