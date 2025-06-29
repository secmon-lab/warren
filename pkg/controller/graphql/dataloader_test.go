package graphql

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

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
	loaders := NewDataLoaders(repo, slackClient)
	ctx := context.Background()

	// Load single ticket
	ticket, err := GetTicketWithLoaders(ctx, loaders, "ticket1")
	gt.NoError(t, err)
	gt.Equal(t, ticket.ID, types.TicketID("ticket1"))
	gt.Equal(t, ticket.Title, "Test Ticket 1")
}

func TestTicketLoader_BatchLoad(t *testing.T) {
	repo, slackClient := setupTestData()
	loaders := NewDataLoaders(repo, slackClient)
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
	loaders := NewDataLoaders(repo, slackClient)
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
	loaders := NewDataLoaders(repo, slackClient)
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

	loaders := NewDataLoaders(repo, slackClient)
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

	// Create middleware
	middleware := DataLoaderMiddleware(repo, slackClient)

	// Create a simple handler that uses the loaders
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to use the loaders from context
		loaders := DataLoadersFor(r.Context())
		gt.NotNil(t, loaders)

		// Load a ticket to verify middleware works
		ticket, err := GetTicket(r.Context(), "ticket1")
		gt.NoError(t, err)
		gt.Equal(t, ticket.ID, types.TicketID("ticket1"))

		w.WriteHeader(http.StatusOK)
	}))

	// Test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	gt.Equal(t, w.Code, http.StatusOK)
}

func TestCaching(t *testing.T) {
	repo, slackClient := setupTestData()
	loaders := NewDataLoaders(repo, slackClient)
	ctx := context.Background()

	// Load the same ticket twice
	ticket1, err1 := GetTicketWithLoaders(ctx, loaders, "ticket1")
	ticket2, err2 := GetTicketWithLoaders(ctx, loaders, "ticket1")

	gt.NoError(t, err1)
	gt.NoError(t, err2)
	gt.Equal(t, ticket1.ID, ticket2.ID)
	gt.Equal(t, ticket1.Title, ticket2.Title)
}

func TestDataLoaderConcurrency(t *testing.T) {
	repo, slackClient := setupTestData()
	loaders := NewDataLoaders(repo, slackClient)
	ctx := context.Background()

	// Simulate high concurrency
	numGoroutines := 100
	var wg sync.WaitGroup
	results := make([]*ticket.Ticket, numGoroutines)
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			// All load the same ticket to test caching under concurrency
			ticket, err := GetTicketWithLoaders(ctx, loaders, "ticket1")
			results[index] = ticket
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// Verify all results are correct
	for i := 0; i < numGoroutines; i++ {
		gt.NoError(t, errors[i])
		gt.NotNil(t, results[i])
		gt.Equal(t, results[i].ID, types.TicketID("ticket1"))
		gt.Equal(t, results[i].Title, "Test Ticket 1")
	}
}

func TestDataLoaderUseBatchMethods(t *testing.T) {
	repo, slackClient := setupTestData()
	loaders := NewDataLoaders(repo, slackClient)
	ctx := context.Background()

	// Reset counters before test
	repo.ResetCallCounts()

	// Load multiple tickets concurrently to trigger DataLoader batching
	ticketIDs := []types.TicketID{"ticket1", "ticket2", "ticket3"}
	var wg sync.WaitGroup
	for _, id := range ticketIDs {
		wg.Add(1)
		go func(ticketID types.TicketID) {
			defer wg.Done()
			_, err := GetTicketWithLoaders(ctx, loaders, ticketID)
			gt.NoError(t, err)
		}(id)
	}
	wg.Wait()

	// Load multiple alerts concurrently to trigger DataLoader batching
	alertIDs := []types.AlertID{"alert1", "alert2", "alert3"}
	for _, id := range alertIDs {
		wg.Add(1)
		go func(alertID types.AlertID) {
			defer wg.Done()
			_, err := GetAlertWithLoaders(ctx, loaders, alertID)
			gt.NoError(t, err)
		}(id)
	}
	wg.Wait()

	// Check call counts
	counts := repo.GetAllCallCounts()

	// Verify that batch methods were called instead of individual methods
	gt.Number(t, counts["BatchGetTickets"]).Greater(0)
	gt.Number(t, counts["BatchGetAlerts"]).Greater(0)

	// Verify that individual methods were NOT called (N+1 problem avoided)
	gt.Number(t, counts["GetTicket"]).Equal(0)
	gt.Number(t, counts["GetAlert"]).Equal(0)

	t.Logf("Call counts: %+v", counts)
}

// Helper functions for testing with loaders
func GetTicketWithLoaders(ctx context.Context, loaders *DataLoaders, ticketID types.TicketID) (*ticket.Ticket, error) {
	thunk := loaders.TicketLoader.Load(ctx, ticketID)
	return thunk()
}

func GetAlertWithLoaders(ctx context.Context, loaders *DataLoaders, alertID types.AlertID) (*alert.Alert, error) {
	thunk := loaders.AlertLoader.Load(ctx, alertID)
	return thunk()
}

func GetUserWithLoaders(ctx context.Context, loaders *DataLoaders, userID string) (*graphql1.User, error) {
	thunk := loaders.UserLoader.Load(ctx, userID)
	return thunk()
}
