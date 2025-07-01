package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestCreateManualTicket(t *testing.T) {
	type testCase struct {
		title       string
		description string
		user        *slack.User
		expectError bool
	}

	runTest := func(tc testCase) func(t *testing.T) {
		return func(t *testing.T) {
			ctx := context.Background()
			repo := repository.NewMemory()

			// Create LLM client mock for embedding generation
			llmMock := &mock.LLMClientMock{
				GenerateEmbeddingFunc: func(ctx context.Context, dimension int, input []string) ([][]float64, error) {
					// Return mock embedding vector with correct dimension
					embedding := make([]float64, dimension)
					for i := range embedding {
						embedding[i] = 0.1 + float64(i)*0.01 // Generate some test values
					}
					return [][]float64{embedding}, nil
				},
			}

			uc := New(WithRepository(repo), WithLLMClient(llmMock))

			ticket, err := uc.CreateManualTicket(ctx, tc.title, tc.description, tc.user)

			if tc.expectError {
				gt.Error(t, err)
				return
			}

			gt.NoError(t, err)
			gt.Value(t, ticket).NotNil()
			gt.Value(t, ticket.Metadata.Title).Equal(tc.title)
			gt.Value(t, ticket.Metadata.Description).Equal(tc.description)
			gt.Value(t, ticket.Assignee).Equal(tc.user)
			gt.Array(t, ticket.AlertIDs).Length(0)         // Should be empty array
			gt.Value(t, ticket.Metadata.Summary).Equal("") // Should be empty per requirements

			// Verify embedding was generated for manual tickets (title + description only)
			if tc.title != "" {
				gt.Array(t, ticket.Embedding).Length(256) // Mock returns 256-dimensional vector
			}

			// Verify ticket was saved to repository
			savedTicket, err := repo.GetTicket(ctx, ticket.ID)
			gt.NoError(t, err)
			gt.Value(t, savedTicket.ID).Equal(ticket.ID)
			gt.Value(t, savedTicket.Metadata.Title).Equal(tc.title)

			// Verify LLM was called for embedding generation
			gt.Array(t, llmMock.GenerateEmbeddingCalls()).Length(1)
		}
	}

	t.Run("valid title and description", runTest(testCase{
		title:       "Test Title",
		description: "Test Description",
		user:        &slack.User{ID: "user1", Name: "Test User"},
		expectError: false,
	}))

	t.Run("valid title only", runTest(testCase{
		title:       "Test Title",
		description: "",
		user:        &slack.User{ID: "user1", Name: "Test User"},
		expectError: false,
	}))

	t.Run("empty title", runTest(testCase{
		title:       "",
		description: "Test Description",
		user:        &slack.User{ID: "user1", Name: "Test User"},
		expectError: true,
	}))

	t.Run("nil user", runTest(testCase{
		title:       "Test Title",
		description: "Test Description",
		user:        nil,
		expectError: false,
	}))
}

func TestUpdateTicketStatus(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	// Create LLM client mock
	llmMock := &mock.LLMClientMock{
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, input []string) ([][]float64, error) {
			embedding := make([]float64, dimension)
			for i := range embedding {
				embedding[i] = 0.1 + float64(i)*0.01
			}
			return [][]float64{embedding}, nil
		},
	}

	// Create use case without Slack service to test core functionality
	uc := New(WithRepository(repo), WithLLMClient(llmMock))

	// Create a test ticket first
	user := &slack.User{ID: "user1", Name: "Test User"}
	ticket, err := uc.CreateManualTicket(ctx, "Test Title", "Test Description", user)
	gt.NoError(t, err)
	gt.Value(t, ticket).NotNil()

	// Test updating ticket status
	updatedTicket, err := uc.UpdateTicketStatus(ctx, ticket.ID, types.TicketStatus("resolved"))
	gt.NoError(t, err)
	gt.Value(t, updatedTicket).NotNil()
	gt.Value(t, updatedTicket.Status).Equal(types.TicketStatus("resolved"))
}

func TestUpdateTicketConclusion(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	// Create LLM client mock
	llmMock := &mock.LLMClientMock{
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, input []string) ([][]float64, error) {
			embedding := make([]float64, dimension)
			for i := range embedding {
				embedding[i] = 0.1 + float64(i)*0.01
			}
			return [][]float64{embedding}, nil
		},
	}

	// Create use case without Slack service to test core functionality
	uc := New(WithRepository(repo), WithLLMClient(llmMock))

	// Create a test ticket first and set it to resolved
	user := &slack.User{ID: "user1", Name: "Test User"}
	ticket, err := uc.CreateManualTicket(ctx, "Test Title", "Test Description", user)
	gt.NoError(t, err)
	gt.Value(t, ticket).NotNil()

	// Set ticket to resolved status
	ticket.Status = types.TicketStatus("resolved")
	err = repo.PutTicket(ctx, *ticket)
	gt.NoError(t, err)

	// Test updating ticket conclusion
	updatedTicket, err := uc.UpdateTicketConclusion(ctx, ticket.ID, types.AlertConclusion("true_positive"), "Test reason")
	gt.NoError(t, err)
	gt.Value(t, updatedTicket).NotNil()
	gt.Value(t, updatedTicket.Conclusion).Equal(types.AlertConclusion("true_positive"))
	gt.Value(t, updatedTicket.Reason).Equal("Test reason")
}

func TestUpdateTicketSlackIntegration(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	// Create LLM client mock
	llmMock := &mock.LLMClientMock{
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, input []string) ([][]float64, error) {
			embedding := make([]float64, dimension)
			for i := range embedding {
				embedding[i] = 0.1 + float64(i)*0.01
			}
			return [][]float64{embedding}, nil
		},
	}

	// Create use case without Slack service to test core functionality
	uc := New(WithRepository(repo), WithLLMClient(llmMock))

	// Create a test ticket
	user := &slack.User{ID: "user1", Name: "Test User"}
	ticket, err := uc.CreateManualTicket(ctx, "Test Title", "Test Description", user)
	gt.NoError(t, err)
	gt.Value(t, ticket).NotNil()

	// Test updating ticket title and description
	updatedTicket, err := uc.UpdateTicket(ctx, ticket.ID, "Updated Title", "Updated Description", user)
	gt.NoError(t, err)
	gt.Value(t, updatedTicket).NotNil()
	gt.Value(t, updatedTicket.Metadata.Title).Equal("Updated Title")
	gt.Value(t, updatedTicket.Metadata.Description).Equal("Updated Description")

	// Test updating ticket status
	updatedTicket, err = uc.UpdateTicketStatus(ctx, ticket.ID, types.TicketStatus("resolved"))
	gt.NoError(t, err)
	gt.Value(t, updatedTicket).NotNil()
	gt.Value(t, updatedTicket.Status).Equal(types.TicketStatus("resolved"))

	// Test updating ticket conclusion (requires resolved status)
	updatedTicket, err = uc.UpdateTicketConclusion(ctx, ticket.ID, types.AlertConclusion("true_positive"), "Test conclusion reason")
	gt.NoError(t, err)
	gt.Value(t, updatedTicket).NotNil()
	gt.Value(t, updatedTicket.Conclusion).Equal(types.AlertConclusion("true_positive"))
	gt.Value(t, updatedTicket.Reason).Equal("Test conclusion reason")

	// Test updating multiple tickets status
	// Create another ticket for batch testing
	ticket2, err := uc.CreateManualTicket(ctx, "Test Title 2", "Test Description 2", user)
	gt.NoError(t, err)

	tickets, err := uc.UpdateMultipleTicketsStatus(ctx, []types.TicketID{ticket.ID, ticket2.ID}, types.TicketStatus("open"))
	gt.NoError(t, err)
	gt.Array(t, tickets).Length(2)
	gt.Value(t, tickets[0].Status).Equal(types.TicketStatus("open"))
	gt.Value(t, tickets[1].Status).Equal(types.TicketStatus("open"))
}

func TestCreateTicketFromAlerts(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	// Create LLM client mock
	llmMock := &mock.LLMClientMock{
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, input []string) ([][]float64, error) {
			embedding := make([]float64, dimension)
			for i := range embedding {
				embedding[i] = 0.1 + float64(i)*0.01
			}
			return [][]float64{embedding}, nil
		},
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{`{"title": "Test Title", "description": "Test Description", "summary": "Test Summary"}`},
					}, nil
				},
			}, nil
		},
	}

	uc := New(WithRepository(repo), WithLLMClient(llmMock))

	t.Run("successful ticket creation from unbound alerts", func(t *testing.T) {
		// Create test alerts
		alert1 := &alert.Alert{
			ID:       types.NewAlertID(),
			TicketID: types.EmptyTicketID, // Unbound
			Metadata: alert.Metadata{
				Title: "Test Alert 1",
			},
			Embedding: make([]float32, 256),
		}
		alert2 := &alert.Alert{
			ID:       types.NewAlertID(),
			TicketID: types.EmptyTicketID, // Unbound
			Metadata: alert.Metadata{
				Title: "Test Alert 2",
			},
			Embedding: make([]float32, 256),
		}

		// Store alerts in repository
		err := repo.PutAlert(ctx, *alert1)
		gt.NoError(t, err)
		err = repo.PutAlert(ctx, *alert2)
		gt.NoError(t, err)

		user := &slack.User{ID: "user1", Name: "Test User"}
		alertIDs := []types.AlertID{alert1.ID, alert2.ID}

		// Create ticket from alerts
		ticket, err := uc.CreateTicketFromAlerts(ctx, alertIDs, user, nil)

		gt.NoError(t, err)
		gt.Value(t, ticket).NotNil()
		gt.Array(t, ticket.AlertIDs).Length(2)
		gt.Value(t, ticket.Assignee).Equal(user)

		// Verify alerts are now bound to the ticket
		updatedAlert1, err := repo.GetAlert(ctx, alert1.ID)
		gt.NoError(t, err)
		gt.Value(t, updatedAlert1.TicketID).Equal(ticket.ID)

		updatedAlert2, err := repo.GetAlert(ctx, alert2.ID)
		gt.NoError(t, err)
		gt.Value(t, updatedAlert2.TicketID).Equal(ticket.ID)
	})

	t.Run("error when alert is already bound to another ticket", func(t *testing.T) {
		// Create first ticket
		existingTicket, err := uc.CreateManualTicket(ctx, "Existing Ticket", "Description", &slack.User{ID: "user1"})
		gt.NoError(t, err)

		// Create test alert already bound to existing ticket
		boundAlert := &alert.Alert{
			ID:       types.NewAlertID(),
			TicketID: existingTicket.ID, // Already bound
			Metadata: alert.Metadata{
				Title: "Already Bound Alert",
			},
			Embedding: make([]float32, 256),
		}

		// Create unbound alert
		unboundAlert := &alert.Alert{
			ID:       types.NewAlertID(),
			TicketID: types.EmptyTicketID, // Unbound
			Metadata: alert.Metadata{
				Title: "Unbound Alert",
			},
			Embedding: make([]float32, 256),
		}

		// Store alerts in repository
		err = repo.PutAlert(ctx, *boundAlert)
		gt.NoError(t, err)
		err = repo.PutAlert(ctx, *unboundAlert)
		gt.NoError(t, err)

		user := &slack.User{ID: "user2", Name: "Test User 2"}
		alertIDs := []types.AlertID{boundAlert.ID, unboundAlert.ID}

		// Attempt to create ticket from alerts (should fail)
		ticket, err := uc.CreateTicketFromAlerts(ctx, alertIDs, user, nil)

		gt.Error(t, err)
		gt.Value(t, ticket).Nil()

		// Verify error message contains expected text
		errorMsg := err.Error()
		gt.Value(t, errorMsg).NotEqual("")
		// Check that error mentions the alert is already bound
		if !strings.Contains(errorMsg, "alert is already bound to a ticket") {
			t.Errorf("Expected error to contain 'alert is already bound to a ticket', got: %s", errorMsg)
		}
		// Note: goerr.V variables may not appear in the error text, so we just verify the main message

		// Verify no changes were made to alerts
		checkAlert, err := repo.GetAlert(ctx, boundAlert.ID)
		gt.NoError(t, err)
		gt.Value(t, checkAlert.TicketID).Equal(existingTicket.ID) // Still bound to original ticket

		checkAlert, err = repo.GetAlert(ctx, unboundAlert.ID)
		gt.NoError(t, err)
		gt.Value(t, checkAlert.TicketID).Equal(types.EmptyTicketID) // Still unbound
	})

	t.Run("error when no alerts provided", func(t *testing.T) {
		user := &slack.User{ID: "user1", Name: "Test User"}
		alertIDs := []types.AlertID{} // Empty

		ticket, err := uc.CreateTicketFromAlerts(ctx, alertIDs, user, nil)

		gt.Error(t, err)
		gt.Value(t, ticket).Nil()

		// Verify error message contains expected text
		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "no alerts provided") {
			t.Errorf("Expected error to contain 'no alerts provided', got: %s", errorMsg)
		}
	})

	t.Run("error when alerts not found", func(t *testing.T) {
		user := &slack.User{ID: "user1", Name: "Test User"}
		nonExistentID := types.NewAlertID()
		alertIDs := []types.AlertID{nonExistentID}

		ticket, err := uc.CreateTicketFromAlerts(ctx, alertIDs, user, nil)

		gt.Error(t, err)
		gt.Value(t, ticket).Nil()

		// Verify error message contains expected text
		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "some alerts not found") {
			t.Errorf("Expected error to contain 'some alerts not found', got: %s", errorMsg)
		}
	})
}
