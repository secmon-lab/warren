package base_test

import (
	"context"
	"testing"
	"time"

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
		"warren_update_finding",
		"warren_get_ticket_comments",
	}
	gt.Value(t, len(specs)).Equal(len(expectedCommands))

	// Find the update_finding command spec
	var updateFindingSpec *gollem.ToolSpec
	var getCommentsSpec *gollem.ToolSpec
	for i, spec := range specs {
		if spec.Name == "warren_update_finding" {
			updateFindingSpec = &specs[i]
		}
		if spec.Name == "warren_get_ticket_comments" {
			getCommentsSpec = &specs[i]
		}
	}

	gt.NotNil(t, updateFindingSpec)
	gt.Value(t, updateFindingSpec.Name).Equal("warren_update_finding")

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

	// Test get_ticket_comments spec
	gt.NotNil(t, getCommentsSpec)
	gt.Value(t, getCommentsSpec.Name).Equal("warren_get_ticket_comments")

	// Check it has the expected parameters
	gt.NotNil(t, getCommentsSpec.Parameters["limit"])
	gt.NotNil(t, getCommentsSpec.Parameters["offset"])

	// Check that limit and offset are not required (should be optional)
	gt.Value(t, len(getCommentsSpec.Required)).Equal(0)
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

func TestWarren_GetTicketComments(t *testing.T) {
	repo := repository.NewMemory()
	ctx := context.Background()

	// Create a test ticket
	testTicket := ticket.Ticket{
		ID:     types.NewTicketID(),
		Status: types.TicketStatusOpen,
		Metadata: ticket.Metadata{
			Title:             "Test Ticket",
			Description:       "Test Description",
			Summary:           "Test Summary",
			TitleSource:       types.SourceHuman,
			DescriptionSource: types.SourceHuman,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save ticket to repository
	gt.NoError(t, repo.PutTicket(ctx, testTicket))

	// Create test comments with different timestamps to ensure proper ordering
	user1 := &slack.User{ID: "U12345", Name: "Test User 1"}
	user2 := &slack.User{ID: "U67890", Name: "Test User 2"}

	comment1 := testTicket.NewComment(ctx, "First comment", user1, "1234567890.123456")
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	comment2 := testTicket.NewComment(ctx, "Second comment", user2, "1234567890.123457")
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	comment3 := testTicket.NewComment(ctx, "Third comment", user1, "")

	// Save comments to repository
	gt.NoError(t, repo.PutTicketComment(ctx, comment1))
	gt.NoError(t, repo.PutTicketComment(ctx, comment2))
	gt.NoError(t, repo.PutTicketComment(ctx, comment3))

	// Create Warren tool
	warren := base.New(repo, testTicket.ID)

	t.Run("get all comments with default pagination", func(t *testing.T) {
		args := map[string]any{}

		result, err := warren.Run(ctx, "warren_get_ticket_comments", args)
		gt.NoError(t, err)

		// Verify result structure
		comments, ok := result["comments"].([]map[string]any)
		gt.Value(t, ok).Equal(true)
		gt.Value(t, len(comments)).Equal(3)
		gt.Value(t, result["total_count"].(int)).Equal(3)
		gt.Value(t, result["offset"].(int)).Equal(0)
		gt.Value(t, result["limit"].(int)).Equal(50)
		gt.Value(t, result["has_more"].(bool)).Equal(false)

		// Verify comment content - should be in reverse chronological order (newest first)
		gt.Value(t, comments[0]["comment"].(string)).Equal("Third comment")
		gt.Value(t, comments[0]["user"].(map[string]any)["name"].(string)).Equal("Test User 1")
		_, hasSlackID := comments[0]["slack_message_id"]
		gt.Value(t, hasSlackID).Equal(false) // Should not have slack_message_id when empty

		gt.Value(t, comments[1]["comment"].(string)).Equal("Second comment")
		gt.Value(t, comments[1]["user"].(map[string]any)["name"].(string)).Equal("Test User 2")
		gt.Value(t, comments[1]["slack_message_id"].(string)).Equal("1234567890.123457")

		gt.Value(t, comments[2]["comment"].(string)).Equal("First comment")
		gt.Value(t, comments[2]["user"].(map[string]any)["name"].(string)).Equal("Test User 1")
		gt.Value(t, comments[2]["slack_message_id"].(string)).Equal("1234567890.123456")
	})

	t.Run("get comments with custom pagination", func(t *testing.T) {
		args := map[string]any{
			"limit":  float64(2), // JSON numbers are float64
			"offset": float64(1),
		}

		result, err := warren.Run(ctx, "warren_get_ticket_comments", args)
		gt.NoError(t, err)

		// Verify pagination
		comments, ok := result["comments"].([]map[string]any)
		gt.Value(t, ok).Equal(true)
		gt.Value(t, len(comments)).Equal(2)
		gt.Value(t, result["total_count"].(int)).Equal(3)
		gt.Value(t, result["offset"].(int)).Equal(1)
		gt.Value(t, result["limit"].(int)).Equal(2)
		gt.Value(t, result["has_more"].(bool)).Equal(false)

		// Should skip first comment (Third) and get second (Second) and third (First)
		gt.Value(t, comments[0]["comment"].(string)).Equal("Second comment")
		gt.Value(t, comments[1]["comment"].(string)).Equal("First comment")
	})

	t.Run("get comments with limit smaller than total", func(t *testing.T) {
		args := map[string]any{
			"limit":  float64(2),
			"offset": float64(0),
		}

		result, err := warren.Run(ctx, "warren_get_ticket_comments", args)
		gt.NoError(t, err)

		// Verify has_more is true
		comments, ok := result["comments"].([]map[string]any)
		gt.Value(t, ok).Equal(true)
		gt.Value(t, len(comments)).Equal(2)
		gt.Value(t, result["total_count"].(int)).Equal(3)
		gt.Value(t, result["has_more"].(bool)).Equal(true)
	})

	t.Run("ticket with no comments", func(t *testing.T) {
		// Create another ticket with no comments
		emptyTicket := ticket.Ticket{
			ID:     types.NewTicketID(),
			Status: types.TicketStatusOpen,
			Metadata: ticket.Metadata{
				Title:             "Empty Ticket",
				TitleSource:       types.SourceHuman,
				DescriptionSource: types.SourceHuman,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		gt.NoError(t, repo.PutTicket(ctx, emptyTicket))

		warren := base.New(repo, emptyTicket.ID)
		args := map[string]any{}

		result, err := warren.Run(ctx, "warren_get_ticket_comments", args)
		gt.NoError(t, err)

		// Should return empty comments array
		comments, ok := result["comments"].([]map[string]any)
		gt.Value(t, ok).Equal(true)
		gt.Value(t, len(comments)).Equal(0)
		gt.Value(t, result["total_count"].(int)).Equal(0)
		gt.Value(t, result["has_more"].(bool)).Equal(false)
	})
}
