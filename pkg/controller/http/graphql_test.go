package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/activity"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	slack_service "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/user"
	slack_api "github.com/slack-go/slack"
)

type graphqlRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type graphqlResponse struct {
	Data   interface{} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

func TestGraphQLQueries(t *testing.T) {
	// Test cases
	t.Run("query tickets", func(t *testing.T) {
		repo := repository.NewMemory()
		ticketID := types.NewTicketID()
		alertID := types.NewAlertID()
		ticketObj := &ticket.Ticket{
			ID:        ticketID,
			Status:    types.TicketStatusOpen,
			AlertIDs:  []types.AlertID{alertID},
			CreatedAt: time.Now(),
		}
		alertObj := &alert.Alert{
			ID:       alertID,
			TicketID: ticketID,
			Metadata: alert.Metadata{
				Title: "Test Alert",
			},
			CreatedAt: time.Now(),
		}
		gt.NoError(t, repo.PutTicket(context.Background(), *ticketObj))
		gt.NoError(t, repo.PutAlert(context.Background(), *alertObj))
		server := httptest.NewServer(graphqlHandler(repo, nil, nil))
		defer server.Close()

		req := graphqlRequest{
			Query: `
				query {
					tickets {
						tickets {
							id
							status
							createdAt
						}
						totalCount
					}
				}
			`,
		}

		resp, err := sendGraphQLRequest(server.URL, req)
		gt.NoError(t, err)
		gt.Array(t, resp.Errors).Length(0)

		data, ok := resp.Data.(map[string]interface{})
		gt.True(t, ok)

		ticketsResponse, ok := data["tickets"].(map[string]interface{})
		gt.True(t, ok)

		tickets, ok := ticketsResponse["tickets"].([]interface{})
		gt.True(t, ok)
		gt.Array(t, tickets).Length(1)

		ticket, ok := tickets[0].(map[string]interface{})
		gt.True(t, ok)
		gt.Value(t, ticket["id"]).Equal(ticketID.String())
		gt.Value(t, ticket["status"]).Equal(types.TicketStatusOpen.String())
	})

	t.Run("query ticket by id", func(t *testing.T) {
		repo := repository.NewMemory()
		ticketID := types.NewTicketID()
		alertID := types.NewAlertID()
		ticketObj := &ticket.Ticket{
			ID:        ticketID,
			Status:    types.TicketStatusOpen,
			AlertIDs:  []types.AlertID{alertID},
			CreatedAt: time.Now(),
		}
		alertObj := &alert.Alert{
			ID:       alertID,
			TicketID: ticketID,
			Metadata: alert.Metadata{
				Title: "Test Alert",
			},
			CreatedAt: time.Now(),
		}
		gt.NoError(t, repo.PutTicket(context.Background(), *ticketObj))
		gt.NoError(t, repo.PutAlert(context.Background(), *alertObj))
		server := httptest.NewServer(graphqlHandler(repo, nil, nil))
		defer server.Close()

		req := graphqlRequest{
			Query: `
				query($id: ID!) {
					ticket(id: $id) {
						id
						status
						alerts {
							id
							title
						}
					}
				}
			`,
			Variables: map[string]interface{}{
				"id": ticketID.String(),
			},
		}

		resp, err := sendGraphQLRequest(server.URL, req)
		gt.NoError(t, err)
		gt.Array(t, resp.Errors).Length(0)

		data, ok := resp.Data.(map[string]interface{})
		gt.True(t, ok)

		ticket, ok := data["ticket"].(map[string]interface{})
		gt.True(t, ok)
		gt.Value(t, ticket["id"]).Equal(ticketID.String())
		gt.Value(t, ticket["status"]).Equal(types.TicketStatusOpen.String())

		alerts, ok := ticket["alerts"].([]interface{})
		gt.True(t, ok)
		gt.Array(t, alerts).Length(1)

		alert, ok := alerts[0].(map[string]interface{})
		gt.True(t, ok)
		gt.Value(t, alert["id"]).Equal(alertID.String())
		gt.Value(t, alert["title"]).Equal("Test Alert")
	})

	t.Run("query alerts", func(t *testing.T) {
		repo := repository.NewMemory()
		// Add an alert not associated with any ticket
		alertNoTicket := &alert.Alert{
			ID:       types.NewAlertID(),
			TicketID: types.EmptyTicketID,
			Metadata: alert.Metadata{
				Title: "Orphan Alert",
			},
			CreatedAt: time.Now(),
		}
		gt.NoError(t, repo.PutAlert(context.Background(), *alertNoTicket))
		server := httptest.NewServer(graphqlHandler(repo, nil, nil))
		defer server.Close()

		req := graphqlRequest{
			Query: `
				query {
					alerts {
						alerts {
							id
							title
						}
						totalCount
					}
				}
			`,
		}

		resp, err := sendGraphQLRequest(server.URL, req)
		gt.NoError(t, err)
		gt.Array(t, resp.Errors).Length(0)

		data, ok := resp.Data.(map[string]interface{})
		gt.True(t, ok)

		alertsResponse, ok := data["alerts"].(map[string]interface{})
		gt.True(t, ok)

		totalCount, ok := alertsResponse["totalCount"].(float64)
		gt.True(t, ok)
		gt.Value(t, totalCount).Equal(float64(1))

		alerts, ok := alertsResponse["alerts"].([]interface{})
		gt.True(t, ok)
		gt.Array(t, alerts).Length(1)

		alert, ok := alerts[0].(map[string]interface{})
		gt.True(t, ok)
		gt.Value(t, alert["id"]).Equal(alertNoTicket.ID.String())
		gt.Value(t, alert["title"]).Equal("Orphan Alert")
	})

	t.Run("query ticket with comments", func(t *testing.T) {
		repo := repository.NewMemory()
		ticketID := types.NewTicketID()
		ticketObj := &ticket.Ticket{
			ID:        ticketID,
			Status:    types.TicketStatusOpen,
			CreatedAt: time.Now(),
		}
		gt.NoError(t, repo.PutTicket(context.Background(), *ticketObj))

		// Add a comment with Slack markdown
		comment := ticket.Comment{
			ID:       types.NewCommentID(),
			TicketID: ticketID,
			Comment:  "This is a *bold* comment with <@U123456> mention",
			User: &slack.User{
				ID:   "U123456",
				Name: "Test User",
			},
			CreatedAt: time.Now(),
		}
		gt.NoError(t, repo.PutTicketComment(context.Background(), comment))

		server := httptest.NewServer(graphqlHandler(repo, nil, nil))
		defer server.Close()

		req := graphqlRequest{
			Query: `
				query($id: ID!) {
					ticket(id: $id) {
						id
						comments {
							id
							content
							user {
								id
								name
							}
						}
					}
				}
			`,
			Variables: map[string]interface{}{
				"id": ticketID.String(),
			},
		}

		resp, err := sendGraphQLRequest(server.URL, req)
		gt.NoError(t, err)
		gt.Array(t, resp.Errors).Length(0)

		data, ok := resp.Data.(map[string]interface{})
		gt.True(t, ok)

		ticket, ok := data["ticket"].(map[string]interface{})
		gt.True(t, ok)

		comments, ok := ticket["comments"].([]interface{})
		gt.True(t, ok)
		gt.Array(t, comments).Length(1)

		commentData, ok := comments[0].(map[string]interface{})
		gt.True(t, ok)
		// Since slack service is nil, content should be returned as-is
		gt.Value(t, commentData["content"]).Equal("This is a *bold* comment with <@U123456> mention")
	})
}

func sendGraphQLRequest(url string, req graphqlRequest) (*graphqlResponse, error) {
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url+"/graphql", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var graphqlResp graphqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		return nil, err
	}

	return &graphqlResp, nil
}

func setupGraphQLTestData(t *testing.T) *repository.Memory {
	repo := repository.NewMemory()
	ctx := context.Background()

	// Use agent context to prevent automatic activity creation when creating tickets
	agentCtx := user.WithAgent(ctx)

	// Add test activities
	activityID1 := types.NewActivityID()
	activityID2 := types.NewActivityID()
	activityID3 := types.NewActivityID()
	activityID4 := types.NewActivityID()
	activityID5 := types.NewActivityID()

	// Add test tickets
	ticketID1 := types.TicketID("ticket1")
	ticketID2 := types.TicketID("ticket2")
	ticketID3 := types.TicketID("ticket3")

	ticket1 := ticket.Ticket{
		ID:        ticketID1,
		Status:    types.TicketStatusOpen,
		CreatedAt: time.Now(),
		Metadata: ticket.Metadata{
			Title: "Test Ticket 1",
		},
	}
	ticket2 := ticket.Ticket{
		ID:        ticketID2,
		Status:    types.TicketStatusPending,
		CreatedAt: time.Now(),
		Metadata: ticket.Metadata{
			Title: "Test Ticket 2",
		},
	}
	ticket3 := ticket.Ticket{
		ID:        ticketID3,
		Status:    types.TicketStatusResolved,
		CreatedAt: time.Now(),
		Metadata: ticket.Metadata{
			Title: "Test Ticket 3",
		},
	}

	// Use agent context to prevent automatic activity creation
	gt.NoError(t, repo.PutTicket(agentCtx, ticket1))
	gt.NoError(t, repo.PutTicket(agentCtx, ticket2))
	gt.NoError(t, repo.PutTicket(agentCtx, ticket3))

	// Add test alerts
	alertID1 := types.AlertID("alert1")
	alertID2 := types.AlertID("alert2")

	alert1 := alert.Alert{
		ID:        alertID1,
		TicketID:  ticketID1,
		CreatedAt: time.Now(),
		Metadata: alert.Metadata{
			Title: "Test Alert 1",
		},
	}
	alert2 := alert.Alert{
		ID:        alertID2,
		TicketID:  ticketID2,
		CreatedAt: time.Now(),
		Metadata: alert.Metadata{
			Title: "Test Alert 2",
		},
	}

	gt.NoError(t, repo.PutAlert(ctx, alert1))
	gt.NoError(t, repo.PutAlert(ctx, alert2))

	// Add test activities
	activity1 := &activity.Activity{
		ID:        activityID1,
		Type:      types.ActivityTypeTicketCreated,
		UserID:    "user1",
		TicketID:  ticketID1,
		CreatedAt: time.Now().Add(-5 * time.Minute),
	}
	activity2 := &activity.Activity{
		ID:        activityID2,
		Type:      types.ActivityTypeTicketCreated,
		UserID:    "user2",
		TicketID:  ticketID2,
		CreatedAt: time.Now().Add(-4 * time.Minute),
	}
	activity3 := &activity.Activity{
		ID:        activityID3,
		Type:      types.ActivityTypeTicketCreated,
		UserID:    "user1",
		TicketID:  ticketID3,
		CreatedAt: time.Now().Add(-3 * time.Minute),
	}
	activity4 := &activity.Activity{
		ID:        activityID4,
		Type:      types.ActivityTypeAlertBound,
		UserID:    "user2",
		AlertID:   alertID1,
		CreatedAt: time.Now().Add(-2 * time.Minute),
	}
	activity5 := &activity.Activity{
		ID:        activityID5,
		Type:      types.ActivityTypeAlertBound,
		UserID:    "user1",
		AlertID:   alertID2,
		CreatedAt: time.Now().Add(-1 * time.Minute),
	}

	gt.NoError(t, repo.PutActivity(ctx, activity1))
	gt.NoError(t, repo.PutActivity(ctx, activity2))
	gt.NoError(t, repo.PutActivity(ctx, activity3))
	gt.NoError(t, repo.PutActivity(ctx, activity4))
	gt.NoError(t, repo.PutActivity(ctx, activity5))

	return repo
}

func setupMockSlackService() (*slack_service.Service, error) {
	mockClient := &mock.SlackClientMock{
		GetUserInfoFunc: func(userID string) (*slack_api.User, error) {
			userNames := map[string]string{
				"user1": "User One",
				"user2": "User Two",
			}
			name := userNames[userID]
			if name == "" {
				name = userID
			}
			return &slack_api.User{
				ID:   userID,
				Name: name,
			}, nil
		},
		GetUsersInfoFunc: func(users ...string) (*[]slack_api.User, error) {
			userNames := map[string]string{
				"user1": "User One",
				"user2": "User Two",
			}

			result := make([]slack_api.User, len(users))
			for i, userID := range users {
				name := userNames[userID]
				if name == "" {
					name = userID
				}
				result[i] = slack_api.User{
					ID:   userID,
					Name: name,
				}
			}
			return &result, nil
		},
		AuthTestFunc: func() (*slack_api.AuthTestResponse, error) {
			return &slack_api.AuthTestResponse{
				UserID:       "bot-user",
				TeamID:       "test-team",
				Team:         "test-team",
				EnterpriseID: "",
				BotID:        "bot-id",
			}, nil
		},
		GetTeamInfoFunc: func() (*slack_api.TeamInfo, error) {
			return &slack_api.TeamInfo{
				Domain: "test-workspace",
			}, nil
		},
	}

	return slack_service.New(mockClient, "test-channel")
}

func TestDataLoaderIntegration(t *testing.T) {
	t.Run("ActivityFeed_DataLoader_Batching", func(t *testing.T) {
		// Setup repository using existing test data setup
		repo := setupGraphQLTestData(t)

		// Setup mock Slack service
		slackService, err := setupMockSlackService()
		gt.NoError(t, err)

		// Create HTTP server with DataLoader middleware
		server := httptest.NewServer(graphqlHandler(repo, slackService, nil))
		defer server.Close()

		// Send GraphQL query for activity feed
		req := graphqlRequest{
			Query: `
				query {
					activities(offset: 0, limit: 5) {
						activities {
							id
							type
							createdAt
							user {
								id
								name
							}
							ticket {
								id
								title
							}
							alert {
								id  
								title
							}
						}
						totalCount
					}
				}
			`,
		}

		resp, err := sendGraphQLRequest(server.URL, req)
		gt.NoError(t, err)
		gt.Array(t, resp.Errors).Length(0)

		// Verify response structure and data correctness
		data, ok := resp.Data.(map[string]interface{})
		gt.True(t, ok)

		activitiesResponse, ok := data["activities"].(map[string]interface{})
		gt.True(t, ok)

		activities, ok := activitiesResponse["activities"].([]interface{})
		gt.True(t, ok)
		gt.Array(t, activities).Length(5)

		// Verify each activity has the expected nested data with correct values
		for _, activityInterface := range activities {
			activity, ok := activityInterface.(map[string]interface{})
			gt.True(t, ok)

			// Verify required fields are present
			gt.NotNil(t, activity["id"])
			gt.NotNil(t, activity["type"])
			gt.NotNil(t, activity["createdAt"])

			// Check that nested user data is loaded correctly
			if user, exists := activity["user"]; exists && user != nil {
				userMap := user.(map[string]interface{})
				gt.NotNil(t, userMap["id"])
				gt.NotNil(t, userMap["name"])

				// Verify user names are correctly mapped
				userID := userMap["id"].(string)
				userName := userMap["name"].(string)
				if userID == "user1" {
					gt.Equal(t, userName, "User One")
				} else if userID == "user2" {
					gt.Equal(t, userName, "User Two")
				}
			}

			// Check that nested ticket data is loaded correctly (for ticket activities)
			if ticket, exists := activity["ticket"]; exists && ticket != nil {
				ticketMap := ticket.(map[string]interface{})
				gt.NotNil(t, ticketMap["id"])
				gt.NotNil(t, ticketMap["title"])

				// Verify ticket titles match expected values
				ticketID := ticketMap["id"].(string)
				ticketTitle := ticketMap["title"].(string)
				switch ticketID {
				case "ticket1":
					gt.Equal(t, ticketTitle, "Test Ticket 1")
				case "ticket2":
					gt.Equal(t, ticketTitle, "Test Ticket 2")
				case "ticket3":
					gt.Equal(t, ticketTitle, "Test Ticket 3")
				}
			}

			// Check that nested alert data is loaded correctly (for alert activities)
			if alert, exists := activity["alert"]; exists && alert != nil {
				alertMap := alert.(map[string]interface{})
				gt.NotNil(t, alertMap["id"])
				gt.NotNil(t, alertMap["title"])

				// Verify alert titles match expected values
				alertID := alertMap["id"].(string)
				alertTitle := alertMap["title"].(string)
				switch alertID {
				case "alert1":
					gt.Equal(t, alertTitle, "Test Alert 1")
				case "alert2":
					gt.Equal(t, alertTitle, "Test Alert 2")
				}
			}
		}

		// Verify total count
		totalCount, ok := activitiesResponse["totalCount"].(float64)
		gt.True(t, ok)
		gt.Equal(t, int(totalCount), 5)
	})

	t.Run("Ticket_Alert_Cross_Reference", func(t *testing.T) {
		// Test that DataLoader correctly resolves ticket->alerts and alert->ticket references
		repo := repository.NewMemory()
		ctx := context.Background()

		// Setup mock Slack service
		slackService, err := setupMockSlackService()
		gt.NoError(t, err)

		// Add ticket with multiple alerts
		ticketID := types.TicketID("ticket1")
		alert1ID := types.AlertID("alert1")
		alert2ID := types.AlertID("alert2")

		ticketObj := ticket.Ticket{
			ID:        ticketID,
			AlertIDs:  []types.AlertID{alert1ID, alert2ID},
			Status:    types.TicketStatusOpen,
			CreatedAt: time.Now(),
			Metadata: ticket.Metadata{
				Title: "Multi-Alert Ticket",
			},
		}
		gt.NoError(t, repo.PutTicket(ctx, ticketObj))

		alert1 := alert.Alert{
			ID:        alert1ID,
			TicketID:  ticketID,
			CreatedAt: time.Now(),
			Metadata: alert.Metadata{
				Title: "First Alert",
			},
		}
		alert2 := alert.Alert{
			ID:        alert2ID,
			TicketID:  ticketID,
			CreatedAt: time.Now(),
			Metadata: alert.Metadata{
				Title: "Second Alert",
			},
		}
		gt.NoError(t, repo.PutAlert(ctx, alert1))
		gt.NoError(t, repo.PutAlert(ctx, alert2))

		server := httptest.NewServer(graphqlHandler(repo, slackService, nil))
		defer server.Close()

		// Query ticket with its alerts
		req := graphqlRequest{
			Query: `
				query($id: ID!) {
					ticket(id: $id) {
						id
						title
						alerts {
							id
							title
							ticket {
								id
								title
							}
						}
					}
				}
			`,
			Variables: map[string]interface{}{
				"id": ticketID.String(),
			},
		}

		resp, err := sendGraphQLRequest(server.URL, req)
		gt.NoError(t, err)
		gt.Array(t, resp.Errors).Length(0)

		// Verify response data correctness
		data, ok := resp.Data.(map[string]interface{})
		gt.True(t, ok)

		ticket, ok := data["ticket"].(map[string]interface{})
		gt.True(t, ok)
		gt.Value(t, ticket["id"]).Equal(ticketID.String())
		gt.Value(t, ticket["title"]).Equal("Multi-Alert Ticket")

		alerts, ok := ticket["alerts"].([]interface{})
		gt.True(t, ok)
		gt.Array(t, alerts).Length(2)

		// Verify nested alert->ticket references work correctly
		for _, alertInterface := range alerts {
			alertMap := alertInterface.(map[string]interface{})
			nestedTicket := alertMap["ticket"].(map[string]interface{})
			gt.Value(t, nestedTicket["id"]).Equal(ticketID.String())
			gt.Value(t, nestedTicket["title"]).Equal("Multi-Alert Ticket")
		}
	})

	t.Run("DataLoader_Batch_Method_Verification", func(t *testing.T) {
		// Test to verify that DataLoader uses batch methods instead of individual gets
		repo := setupGraphQLTestData(t)

		// Reset counters before test
		repo.ResetCallCounts()

		// Setup mock Slack service
		slackService, err := setupMockSlackService()
		gt.NoError(t, err)

		server := httptest.NewServer(graphqlHandler(repo, slackService, nil))
		defer server.Close()

		// Query activities to trigger DataLoader usage
		req := graphqlRequest{
			Query: `
				query {
					activities(offset: 0, limit: 3) {
						activities {
							id
							type
							ticket {
								id
								title
							}
							alert {
								id
								title
							}
						}
					}
				}
			`,
		}

		resp, err := sendGraphQLRequest(server.URL, req)
		gt.NoError(t, err)
		gt.Array(t, resp.Errors).Length(0)

		// Verify that batch methods were used
		counts := repo.GetAllCallCounts()
		t.Logf("Repository method call counts: %+v", counts)

		// Verify batch methods were called
		batchTicketCalls := counts["BatchGetTickets"]
		batchAlertCalls := counts["BatchGetAlerts"]
		gt.Number(t, batchTicketCalls).GreaterOrEqual(1)
		gt.Number(t, batchAlertCalls).GreaterOrEqual(1)

		// Verify individual methods were NOT called (N+1 problem avoided)
		individualTicketCalls := counts["GetTicket"]
		individualAlertCalls := counts["GetAlert"]
		t.Logf("Individual method calls - GetTicket: %d, GetAlert: %d", individualTicketCalls, individualAlertCalls)
		t.Logf("Batch method calls - BatchGetTickets: %d, BatchGetAlerts: %d", batchTicketCalls, batchAlertCalls)
		gt.Number(t, individualTicketCalls).Equal(0)
		gt.Number(t, individualAlertCalls).Equal(0)
	})

	t.Run("Slack_API_Error_Propagation", func(t *testing.T) {
		// Test that Slack API errors are properly propagated to GraphQL responses
		repo := setupGraphQLTestData(t)

		// Setup mock Slack service with error
		mockClient := &mock.SlackClientMock{
			GetUsersInfoFunc: func(users ...string) (*[]slack_api.User, error) {
				return nil, errors.New("slack API rate limit exceeded")
			},
			AuthTestFunc: func() (*slack_api.AuthTestResponse, error) {
				return &slack_api.AuthTestResponse{
					UserID: "bot-user",
					TeamID: "test-team",
				}, nil
			},
			GetTeamInfoFunc: func() (*slack_api.TeamInfo, error) {
				return &slack_api.TeamInfo{
					Domain: "test-workspace",
				}, nil
			},
		}
		slackService, err := slack_service.New(mockClient, "test-channel")
		gt.NoError(t, err)

		server := httptest.NewServer(graphqlHandler(repo, slackService, nil))
		defer server.Close()

		// Query activities to trigger user loading which will cause Slack API error
		req := graphqlRequest{
			Query: `
				query {
					activities(offset: 0, limit: 1) {
						activities {
							id
							type
							user {
								id
								name
							}
						}
					}
				}
			`,
		}

		resp, err := sendGraphQLRequest(server.URL, req)
		gt.NoError(t, err)

		// Verify that GraphQL errors are returned
		gt.Array(t, resp.Errors).Length(1)
		gt.True(t, strings.Contains(resp.Errors[0].Message, "failed to fetch user info from Slack"))
		gt.True(t, strings.Contains(resp.Errors[0].Message, "slack API rate limit exceeded"))
	})
}
