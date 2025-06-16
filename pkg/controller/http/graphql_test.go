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
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
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
		_ = repo.PutTicket(context.Background(), *ticketObj)
		_ = repo.PutAlert(context.Background(), *alertObj)
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
		_ = repo.PutAlert(context.Background(), *alertNoTicket)
		server := httptest.NewServer(graphqlHandler(repo, nil, nil))
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

	t.Run("query ticket with comments", func(t *testing.T) {
		repo := repository.NewMemory()
		ticketID := types.NewTicketID()
		ticketObj := &ticket.Ticket{
			ID:        ticketID,
			Status:    types.TicketStatusOpen,
			CreatedAt: time.Now(),
		}
		_ = repo.PutTicket(context.Background(), *ticketObj)

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
		_ = repo.PutTicketComment(context.Background(), comment)

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
