package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
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
		_ = repo.PutTicket(context.Background(), *ticketObj)
		_ = repo.PutAlert(context.Background(), *alertObj)
		server := httptest.NewServer(graphqlHandler(repo))
		defer server.Close()

		req := graphqlRequest{
			Query: `
				query {
					tickets {
						id
						status
						createdAt
					}
				}
			`,
		}

		resp, err := sendGraphQLRequest(server.URL, req)
		gt.NoError(t, err)
		gt.Array(t, resp.Errors).Length(0)

		data, ok := resp.Data.(map[string]interface{})
		gt.True(t, ok)

		tickets, ok := data["tickets"].([]interface{})
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
		_ = repo.PutTicket(context.Background(), *ticketObj)
		_ = repo.PutAlert(context.Background(), *alertObj)
		server := httptest.NewServer(graphqlHandler(repo))
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
		_ = repo.PutAlert(context.Background(), *alertNoTicket)
		server := httptest.NewServer(graphqlHandler(repo))
		defer server.Close()

		req := graphqlRequest{
			Query: `
				query {
					alerts {
						id
						title
					}
				}
			`,
		}

		resp, err := sendGraphQLRequest(server.URL, req)
		gt.NoError(t, err)
		gt.Array(t, resp.Errors).Length(0)

		data, ok := resp.Data.(map[string]interface{})
		gt.True(t, ok)

		alerts, ok := data["alerts"].([]interface{})
		gt.True(t, ok)
		gt.Array(t, alerts).Length(1)

		alert, ok := alerts[0].(map[string]interface{})
		gt.True(t, ok)
		gt.Value(t, alert["id"]).Equal(alertNoTicket.ID.String())
		gt.Value(t, alert["title"]).Equal("Orphan Alert")
	})

	t.Run("get tickets by status", func(t *testing.T) {
		repo := repository.NewMemory()
		alert1 := &alert.Alert{
			ID: types.NewAlertID(),
			Metadata: alert.Metadata{
				Title:       "Test Alert 1",
				Description: "Test Description 1",
			},
		}
		alert2 := &alert.Alert{
			ID: types.NewAlertID(),
			Metadata: alert.Metadata{
				Title:       "Test Alert 2",
				Description: "Test Description 2",
			},
		}
		alert3 := &alert.Alert{
			ID: types.NewAlertID(),
			Metadata: alert.Metadata{
				Title:       "Test Alert 3",
				Description: "Test Description 3",
			},
		}
		ticket1 := &ticket.Ticket{
			ID:         types.NewTicketID(),
			Status:     types.TicketStatusOpen,
			AlertIDs:   []types.AlertID{alert1.ID, alert2.ID},
			Conclusion: types.AlertConclusionTruePositive,
			Metadata: ticket.Metadata{
				Title:       "Test Ticket 1",
				Description: "Test Description 1",
				Summary:     "Test Summary 1",
			},
		}
		ticket2 := &ticket.Ticket{
			ID:         types.NewTicketID(),
			Status:     types.TicketStatusPending,
			AlertIDs:   []types.AlertID{alert3.ID},
			Conclusion: types.AlertConclusionFalsePositive,
			Metadata: ticket.Metadata{
				Title:       "Test Ticket 2",
				Description: "Test Description 2",
				Summary:     "Test Summary 2",
			},
		}
		_ = repo.PutTicket(context.Background(), *ticket1)
		_ = repo.PutTicket(context.Background(), *ticket2)
		_ = repo.PutAlert(context.Background(), *alert1)
		_ = repo.PutAlert(context.Background(), *alert2)
		_ = repo.PutAlert(context.Background(), *alert3)
		server := httptest.NewServer(graphqlHandler(repo))
		defer server.Close()

		req := graphqlRequest{
			Query: `
				query {
					tickets(statuses: ["open"]) {
						id
						status
						alerts {
							id
							title
						}
					}
				}
			`,
		}

		resp, err := sendGraphQLRequest(server.URL, req)
		gt.NoError(t, err)
		gt.Array(t, resp.Errors).Length(0)

		data, ok := resp.Data.(map[string]interface{})
		gt.True(t, ok)

		tickets, ok := data["tickets"].([]interface{})
		gt.True(t, ok)
		gt.Array(t, tickets).Length(1)

		ticket, ok := tickets[0].(map[string]interface{})
		gt.True(t, ok)
		gt.Value(t, ticket["id"]).Equal(ticket1.ID.String())
		gt.Value(t, ticket["status"]).Equal(types.TicketStatusOpen.String())

		alerts, ok := ticket["alerts"].([]interface{})
		gt.True(t, ok)
		gt.Array(t, alerts).Length(2)

		alertResp1, ok := alerts[0].(map[string]interface{})
		gt.True(t, ok)
		gt.Value(t, alertResp1["id"]).Equal(alert1.ID.String())
		gt.Value(t, alertResp1["title"]).Equal("Test Alert 1")

		alertResp2, ok := alerts[1].(map[string]interface{})
		gt.True(t, ok)
		gt.Value(t, alertResp2["id"]).Equal(alert2.ID.String())
		gt.Value(t, alertResp2["title"]).Equal("Test Alert 2")
	})
}

func sendGraphQLRequest(url string, req graphqlRequest) (*graphqlResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result graphqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
