package loaders

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	graphql1 "github.com/secmon-lab/warren/pkg/domain/model/graphql"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	slack_api "github.com/slack-go/slack"
)

func getUserName(userID string) string {
	userNames := map[string]string{
		"user1": "User One",
		"user2": "User Two",
		"user3": "User Three",
	}

	if name, exists := userNames[userID]; exists {
		return name
	}
	return userID // fallback to userID
}

func setupTestData() (*repository.Memory, *mock.SlackClientMock) {
	repo := repository.NewMemory()

	// Setup test tickets
	ctx := context.Background()
	repo.PutTicket(ctx, ticket.Ticket{
		ID:       "ticket1",
		Metadata: ticket.Metadata{Title: "Test Ticket 1"},
	})
	repo.PutTicket(ctx, ticket.Ticket{
		ID:       "ticket2",
		Metadata: ticket.Metadata{Title: "Test Ticket 2"},
	})
	repo.PutTicket(ctx, ticket.Ticket{
		ID:       "ticket3",
		Metadata: ticket.Metadata{Title: "Test Ticket 3"},
	})

	// Setup test alerts
	repo.PutAlert(ctx, alert.Alert{ID: "alert1"})
	repo.PutAlert(ctx, alert.Alert{ID: "alert2"})
	repo.PutAlert(ctx, alert.Alert{ID: "alert3"})

	slackClient := &mock.SlackClientMock{
		GetUserInfoFunc: func(userID string) (*slack_api.User, error) {
			return &slack_api.User{
				ID:   userID,
				Name: getUserName(userID),
			}, nil
		},
	}

	return repo, slackClient
}

func TestTicketLoader_SingleLoad(t *testing.T) {
	repo, slackClient := setupTestData()
	loaders := NewLoaders(repo, slackClient)
	ctx := context.Background()

	// Load single ticket
	ticket, err := GetTicketWithLoaders(ctx, loaders, "ticket1")
	gt.NoError(t, err)
	gt.Equal(t, ticket.ID, types.TicketID("ticket1"))
	gt.Equal(t, ticket.Title, "Test Ticket 1")
}

func TestTicketLoader_BatchLoad(t *testing.T) {
	repo, slackClient := setupTestData()
	loaders := NewLoaders(repo, slackClient)
	ctx := context.Background()

	// Load multiple tickets concurrently to test DataLoader batching
	ticketIDs := []types.TicketID{"ticket1", "ticket2", "ticket3"}
	results := make([]*ticket.Ticket, len(ticketIDs))
	errors := make([]error, len(ticketIDs))

	var wg sync.WaitGroup
	for i, id := range ticketIDs {
		wg.Add(1)
		go func(index int, ticketID types.TicketID) {
			defer wg.Done()
			t, err := GetTicketWithLoaders(ctx, loaders, ticketID)
			results[index] = t
			errors[index] = err
		}(i, id)
	}
	wg.Wait()

	// Verify all loads returned correct data
	for i, err := range errors {
		gt.NoError(t, err)
		gt.NotNil(t, results[i])
		gt.Equal(t, results[i].ID, ticketIDs[i])
		expectedTitles := []string{"Test Ticket 1", "Test Ticket 2", "Test Ticket 3"}
		gt.Equal(t, results[i].Title, expectedTitles[i])
	}
}

func TestAlertLoader_BatchLoad(t *testing.T) {
	repo, slackClient := setupTestData()
	loaders := NewLoaders(repo, slackClient)
	ctx := context.Background()

	// Load multiple alerts concurrently
	alertIDs := []types.AlertID{"alert1", "alert2", "alert3"}
	results := make([]*alert.Alert, len(alertIDs))
	errors := make([]error, len(alertIDs))

	var wg sync.WaitGroup
	for i, id := range alertIDs {
		wg.Add(1)
		go func(index int, alertID types.AlertID) {
			defer wg.Done()
			a, err := GetAlertWithLoaders(ctx, loaders, alertID)
			results[index] = a
			errors[index] = err
		}(i, id)
	}
	wg.Wait()

	// Verify all loads returned correct data
	for i, err := range errors {
		gt.NoError(t, err)
		gt.NotNil(t, results[i])
		gt.Equal(t, results[i].ID, alertIDs[i])
	}
}

func TestUserLoader_BatchLoad(t *testing.T) {
	repo, slackClient := setupTestData()
	loaders := NewLoaders(repo, slackClient)
	ctx := context.Background()

	// Load multiple users concurrently
	userIDs := []string{"user1", "user2", "user3"}
	results := make([]*graphql1.User, len(userIDs))
	errors := make([]error, len(userIDs))

	var wg sync.WaitGroup
	for i, id := range userIDs {
		wg.Add(1)
		go func(index int, userID string) {
			defer wg.Done()
			u, err := GetUserWithLoaders(ctx, loaders, userID)
			results[index] = u
			errors[index] = err
		}(i, id)
	}
	wg.Wait()

	// Verify all loads returned correct data
	for i, err := range errors {
		gt.NoError(t, err)
		gt.NotNil(t, results[i])
		gt.Equal(t, results[i].ID, userIDs[i])
		expectedNames := []string{"User One", "User Two", "User Three"}
		gt.Equal(t, results[i].Name, expectedNames[i])
	}
}

func TestErrorHandling(t *testing.T) {
	repo, slackClient := setupTestData()

	// Set slack client to simulate error
	slackClient.GetUserInfoFunc = func(userID string) (*slack_api.User, error) {
		return nil, errors.New("simulated slack error")
	}

	loaders := NewLoaders(repo, slackClient)
	ctx := context.Background()

	// Test ticket not found error
	_, err := GetTicketWithLoaders(ctx, loaders, "nonexistent")
	gt.Error(t, err)

	// Test alert not found error
	_, err = GetAlertWithLoaders(ctx, loaders, "nonexistent")
	gt.Error(t, err)

	// Test user error handling (user loader doesn't propagate slack errors, it falls back)
	user, err := GetUserWithLoaders(ctx, loaders, "user1")
	gt.NoError(t, err) // Should not error, falls back to ID
	gt.Equal(t, user.ID, "user1")
	gt.Equal(t, user.Name, "user1") // Fallback to ID
}

func TestMiddleware(t *testing.T) {
	repo, slackClient := setupTestData()

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get loaders from context
		loaders := For(r.Context())
		gt.NotNil(t, loaders)

		// Test that we can load data
		ticket, err := GetTicketWithLoaders(r.Context(), loaders, "ticket1")
		gt.NoError(t, err)
		gt.Equal(t, ticket.ID, types.TicketID("ticket1"))
		gt.Equal(t, ticket.Title, "Test Ticket 1")

		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	middleware := Middleware(repo, slackClient)
	wrappedHandler := middleware(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute
	wrappedHandler.ServeHTTP(w, req)

	// Verify response
	gt.Equal(t, w.Code, http.StatusOK)
}

func TestCaching(t *testing.T) {
	repo, slackClient := setupTestData()
	loaders := NewLoaders(repo, slackClient)
	ctx := context.Background()

	// Load the same ticket multiple times
	ticket1, err := GetTicketWithLoaders(ctx, loaders, "ticket1")
	gt.NoError(t, err)

	ticket2, err := GetTicketWithLoaders(ctx, loaders, "ticket1")
	gt.NoError(t, err)

	// Should return the same data due to caching
	gt.Equal(t, ticket1.ID, ticket2.ID)
	gt.Equal(t, ticket1.Title, ticket2.Title)
}

func TestDataLoaderConcurrency(t *testing.T) {
	repo, slackClient := setupTestData()
	loaders := NewLoaders(repo, slackClient)
	ctx := context.Background()

	start := time.Now()

	// Start multiple concurrent loads to test batching efficiency
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ticketID := types.TicketID("ticket1")
			if index > 4 {
				ticketID = types.TicketID("ticket2")
			}
			ticket, err := GetTicketWithLoaders(ctx, loaders, ticketID)
			gt.NoError(t, err)
			gt.NotNil(t, ticket)

			// Verify correct data is returned
			if ticketID == "ticket1" {
				gt.Equal(t, ticket.Title, "Test Ticket 1")
			} else {
				gt.Equal(t, ticket.Title, "Test Ticket 2")
			}
		}(i)
	}
	wg.Wait()

	elapsed := time.Since(start)

	// Should complete quickly due to DataLoader batching
	gt.True(t, elapsed < 100*time.Millisecond)
}

// Helper functions for testing
func GetTicketWithLoaders(ctx context.Context, loaders *Loaders, ticketID types.TicketID) (*ticket.Ticket, error) {
	thunk := loaders.TicketLoader.Load(ctx, ticketID)
	return thunk()
}

func GetAlertWithLoaders(ctx context.Context, loaders *Loaders, alertID types.AlertID) (*alert.Alert, error) {
	thunk := loaders.AlertLoader.Load(ctx, alertID)
	return thunk()
}

func GetUserWithLoaders(ctx context.Context, loaders *Loaders, userID string) (*graphql1.User, error) {
	thunk := loaders.UserLoader.Load(ctx, userID)
	return thunk()
}
