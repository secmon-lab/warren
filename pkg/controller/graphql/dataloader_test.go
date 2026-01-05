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
	_ = repo.PutTicket(ctx, ticket.Ticket{
		ID:       "ticket1",
		Metadata: ticket.Metadata{Title: "Test Ticket 1"},
	})
	_ = repo.PutTicket(ctx, ticket.Ticket{
		ID:       "ticket2",
		Metadata: ticket.Metadata{Title: "Test Ticket 2"},
	})
	_ = repo.PutTicket(ctx, ticket.Ticket{
		ID:       "ticket3",
		Metadata: ticket.Metadata{Title: "Test Ticket 3"},
	})

	// Setup test alerts
	_ = repo.PutAlert(ctx, alert.Alert{ID: "alert1"})
	_ = repo.PutAlert(ctx, alert.Alert{ID: "alert2"})
	_ = repo.PutAlert(ctx, alert.Alert{ID: "alert3"})

	slackClient := &mock.SlackClientMock{
		GetUserInfoFunc: func(userID string) (*slack_api.User, error) {
			return &slack_api.User{
				ID:   userID,
				Name: getUserName(userID),
			}, nil
		},
		GetUsersInfoFunc: func(users ...string) (*[]slack_api.User, error) {
			result := make([]slack_api.User, len(users))
			for i, userID := range users {
				result[i] = slack_api.User{
					ID:   userID,
					Name: getUserName(userID),
				}
			}
			return &result, nil
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
	t.Run("Repository errors are propagated", func(t *testing.T) {
		repo, slackClient := setupTestData()
		loaders := NewDataLoaders(repo, slackClient)
		ctx := context.Background()

		// Test ticket not found error
		_, err := GetTicketWithLoaders(ctx, loaders, "nonexistent")
		gt.Error(t, err)

		// Test alert not found error
		_, err = GetAlertWithLoaders(ctx, loaders, "nonexistent")
		gt.Error(t, err)
	})

	t.Run("Slack API errors fallback to ID", func(t *testing.T) {
		repo, slackClient := setupTestData()

		// Set slack client to simulate error for batch method
		slackClient.GetUsersInfoFunc = func(users ...string) (*[]slack_api.User, error) {
			return nil, errors.New("simulated slack batch error")
		}

		loaders := NewDataLoaders(repo, slackClient)
		ctx := context.Background()

		// Test user error handling - Slack API errors should fallback to ID to prevent query failures
		user, err := GetUserWithLoaders(ctx, loaders, "user1")
		gt.NoError(t, err) // Should not error, falls back to ID
		gt.Equal(t, user.ID, "user1")
		gt.Equal(t, user.Name, "user1") // Fallback to ID
	})

	t.Run("Nil SlackClient falls back without error", func(t *testing.T) {
		repo, _ := setupTestData()
		loaders := NewDataLoaders(repo, nil) // nil SlackClient
		ctx := context.Background()

		// Test user loading with nil SlackClient - should fallback to ID without error
		user, err := GetUserWithLoaders(ctx, loaders, "user1")
		gt.NoError(t, err) // Should not error, falls back to ID
		gt.Equal(t, user.ID, "user1")
		gt.Equal(t, user.Name, "user1") // Fallback to ID
	})

	t.Run("User not found in Slack response falls back without error", func(t *testing.T) {
		repo, slackClient := setupTestData()

		// Set slack client to return empty result (user not found)
		slackClient.GetUsersInfoFunc = func(users ...string) (*[]slack_api.User, error) {
			return &[]slack_api.User{}, nil // Empty response
		}

		loaders := NewDataLoaders(repo, slackClient)
		ctx := context.Background()

		// Test user not found in Slack response - should fallback to ID without error
		user, err := GetUserWithLoaders(ctx, loaders, "user1")
		gt.NoError(t, err) // Should not error, falls back to ID
		gt.Equal(t, user.ID, "user1")
		gt.Equal(t, user.Name, "user1") // Fallback to ID
	})
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

func TestUserLoaderUseBatchMethod(t *testing.T) {
	repo, slackClient := setupTestData()
	loaders := NewDataLoaders(repo, slackClient)
	ctx := context.Background()

	// Load multiple users concurrently to trigger DataLoader batching
	userIDs := []string{"user1", "user2", "user3"}
	var wg sync.WaitGroup
	for _, id := range userIDs {
		wg.Add(1)
		go func(userID string) {
			defer wg.Done()
			user, err := GetUserWithLoaders(ctx, loaders, userID)
			gt.NoError(t, err)
			gt.Equal(t, user.ID, userID)
			gt.Equal(t, user.Name, getUserName(userID))
		}(id)
	}
	wg.Wait()

	// Verify that GetUsersInfo was called instead of individual GetUserInfo calls
	getUsersInfoCalls := slackClient.GetUsersInfoCalls()
	getUserInfoCalls := slackClient.GetUserInfoCalls()

	// Should have at least one call to GetUsersInfo
	gt.Number(t, len(getUsersInfoCalls)).Greater(0)

	// Should have no calls to individual GetUserInfo (N+1 problem avoided)
	gt.Number(t, len(getUserInfoCalls)).Equal(0)

	t.Logf("GetUsersInfo calls: %d, GetUserInfo calls: %d", len(getUsersInfoCalls), len(getUserInfoCalls))

	// Verify the GetUsersInfo call contains all expected user IDs
	if len(getUsersInfoCalls) > 0 {
		calledUsers := getUsersInfoCalls[0].Users
		gt.Number(t, len(calledUsers)).Equal(3)

		// Check that all user IDs are present in the batch call
		userIDMap := make(map[string]bool)
		for _, userID := range calledUsers {
			userIDMap[userID] = true
		}

		for _, expectedID := range userIDs {
			gt.True(t, userIDMap[expectedID])
		}
	}
}

func TestSystemUserLoader(t *testing.T) {
	t.Run("System user does not call Slack API", func(t *testing.T) {
		repo, slackClient := setupTestData()
		loaders := NewDataLoaders(repo, slackClient)
		ctx := context.Background()

		// Load system user
		user, err := GetUserWithLoaders(ctx, loaders, string(types.SystemUserID))
		gt.NoError(t, err)
		gt.Equal(t, user.ID, string(types.SystemUserID))
		gt.Equal(t, user.Name, "System")

		// Verify Slack API was NOT called
		getUsersInfoCalls := slackClient.GetUsersInfoCalls()
		gt.Number(t, len(getUsersInfoCalls)).Equal(0)
	})

	t.Run("System user mixed with regular users", func(t *testing.T) {
		repo, slackClient := setupTestData()
		loaders := NewDataLoaders(repo, slackClient)
		ctx := context.Background()

		// Load system user and regular users concurrently
		userIDs := []string{string(types.SystemUserID), "user1", "user2"}
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
		}

		// Verify system user has correct name
		gt.Equal(t, results[0].Name, "System")
		gt.Equal(t, results[1].Name, "User One")
		gt.Equal(t, results[2].Name, "User Two")

		// Verify Slack API was called only for non-system users
		getUsersInfoCalls := slackClient.GetUsersInfoCalls()
		gt.Number(t, len(getUsersInfoCalls)).Greater(0)

		// The batch call should only contain non-system users
		if len(getUsersInfoCalls) > 0 {
			calledUsers := getUsersInfoCalls[0].Users
			for _, userID := range calledUsers {
				gt.NotEqual(t, userID, string(types.SystemUserID))
			}
		}
	})

	t.Run("Multiple system users only", func(t *testing.T) {
		repo, slackClient := setupTestData()
		loaders := NewDataLoaders(repo, slackClient)
		ctx := context.Background()

		// Load multiple system users
		userIDs := []string{string(types.SystemUserID), string(types.SystemUserID)}
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
			gt.Equal(t, results[i].ID, string(types.SystemUserID))
			gt.Equal(t, results[i].Name, "System")
		}

		// Verify Slack API was NOT called at all
		getUsersInfoCalls := slackClient.GetUsersInfoCalls()
		gt.Number(t, len(getUsersInfoCalls)).Equal(0)
	})
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
