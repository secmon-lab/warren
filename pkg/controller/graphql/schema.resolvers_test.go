package graphql

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase"
)

var now = time.Now()

func TestUpdateTicketConclusion(t *testing.T) {
	repo := repository.NewMemory()

	// Create LLM client mock for embedding generation
	llmMock := &mock.LLMClientMock{
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, input []string) ([][]float64, error) {
			embedding := make([]float64, dimension)
			for i := range embedding {
				embedding[i] = 0.1 + float64(i)*0.01
			}
			return [][]float64{embedding}, nil
		},
	}

	uc := usecase.New(usecase.WithRepository(repo), usecase.WithLLMClient(llmMock))
	resolver := NewResolver(repo, nil, uc)

	// Create a resolved ticket
	testTicket := &ticket.Ticket{
		ID:     types.NewTicketID(),
		Status: types.TicketStatusResolved,
		Metadata: ticket.Metadata{
			Title:       "Test Ticket",
			Description: "Test Description",
		},
	}
	gt.NoError(t, repo.PutTicket(context.Background(), *testTicket))

	runTest := func(tc struct {
		name           string
		ticketID       string
		conclusion     string
		reason         string
		expectedError  bool
		expectedStatus types.TicketStatus
	}) func(t *testing.T) {
		return func(t *testing.T) {
			ctx := context.Background()
			result, err := resolver.Mutation().UpdateTicketConclusion(ctx, tc.ticketID, tc.conclusion, tc.reason)

			if tc.expectedError {
				gt.Error(t, err)
				return
			}

			gt.NoError(t, err)
			gt.NotNil(t, result)
			gt.Equal(t, tc.conclusion, string(result.Conclusion))
			gt.Equal(t, tc.reason, result.Reason)
		}
	}

	t.Run("success case", runTest(struct {
		name           string
		ticketID       string
		conclusion     string
		reason         string
		expectedError  bool
		expectedStatus types.TicketStatus
	}{
		name:           "success",
		ticketID:       string(testTicket.ID),
		conclusion:     string(types.AlertConclusionTruePositive),
		reason:         "This is a test reason",
		expectedError:  false,
		expectedStatus: types.TicketStatusResolved,
	}))

	t.Run("invalid conclusion", runTest(struct {
		name           string
		ticketID       string
		conclusion     string
		reason         string
		expectedError  bool
		expectedStatus types.TicketStatus
	}{
		name:           "invalid conclusion",
		ticketID:       string(testTicket.ID),
		conclusion:     "invalid_conclusion",
		reason:         "This is a test reason",
		expectedError:  true,
		expectedStatus: types.TicketStatusResolved,
	}))

	t.Run("non-resolved ticket", runTest(struct {
		name           string
		ticketID       string
		conclusion     string
		reason         string
		expectedError  bool
		expectedStatus types.TicketStatus
	}{
		name: "non-resolved ticket",
		ticketID: func() string {
			openTicket := &ticket.Ticket{
				ID:     types.NewTicketID(),
				Status: types.TicketStatusOpen,
				Metadata: ticket.Metadata{
					Title:       "Open Ticket",
					Description: "Open Description",
				},
			}
			gt.NoError(t, repo.PutTicket(context.Background(), *openTicket))
			return string(openTicket.ID)
		}(),
		conclusion:     string(types.AlertConclusionTruePositive),
		reason:         "This is a test reason",
		expectedError:  true,
		expectedStatus: types.TicketStatusOpen,
	}))

	t.Run("ticket not found", runTest(struct {
		name           string
		ticketID       string
		conclusion     string
		reason         string
		expectedError  bool
		expectedStatus types.TicketStatus
	}{
		name:           "ticket not found",
		ticketID:       string(types.NewTicketID()),
		conclusion:     string(types.AlertConclusionTruePositive),
		reason:         "This is a test reason",
		expectedError:  true,
		expectedStatus: types.TicketStatusResolved,
	}))
}

func TestAlertsPaginated(t *testing.T) {
	repo := repository.NewMemory()
	resolver := NewResolver(repo, nil, nil)

	// Create a test ticket with multiple alerts
	ticketID := types.NewTicketID()
	alertIDs := make([]types.AlertID, 7) // Create 7 alerts for pagination testing

	// Create alerts
	for i := 0; i < 7; i++ {
		alertID := types.NewAlertID()
		alertIDs[i] = alertID
		testAlert := &alert.Alert{
			ID:       alertID,
			Schema:   types.AlertSchema("test"),
			TicketID: ticketID,
			Data:     map[string]interface{}{"test": "data"},
		}
		gt.NoError(t, repo.PutAlert(context.Background(), *testAlert))
	}

	// Create ticket with alerts
	testTicket := &ticket.Ticket{
		ID:       ticketID,
		Status:   types.TicketStatusOpen,
		AlertIDs: alertIDs,
		Metadata: ticket.Metadata{
			Title:       "Test Ticket",
			Description: "Test ticket with multiple alerts",
		},
	}
	gt.NoError(t, repo.PutTicket(context.Background(), *testTicket))

	ctx := context.Background()

	t.Run("default pagination", func(t *testing.T) {
		response, err := resolver.Ticket().AlertsPaginated(ctx, testTicket, nil, nil)
		gt.NoError(t, err)
		gt.V(t, response).NotEqual(nil)
		gt.V(t, response.TotalCount).Equal(7)
		gt.V(t, len(response.Alerts)).Equal(5) // Default limit is 5
	})

	t.Run("custom offset and limit", func(t *testing.T) {
		offset := 2
		limit := 3
		response, err := resolver.Ticket().AlertsPaginated(ctx, testTicket, &offset, &limit)
		gt.NoError(t, err)
		gt.V(t, response).NotEqual(nil)
		gt.V(t, response.TotalCount).Equal(7)
		gt.V(t, len(response.Alerts)).Equal(3)
	})

	t.Run("offset beyond range", func(t *testing.T) {
		offset := 10
		limit := 5
		response, err := resolver.Ticket().AlertsPaginated(ctx, testTicket, &offset, &limit)
		gt.NoError(t, err)
		gt.V(t, response).NotEqual(nil)
		gt.V(t, response.TotalCount).Equal(7)
		gt.V(t, len(response.Alerts)).Equal(0) // No alerts returned
	})

	t.Run("partial last page", func(t *testing.T) {
		offset := 5
		limit := 5
		response, err := resolver.Ticket().AlertsPaginated(ctx, testTicket, &offset, &limit)
		gt.NoError(t, err)
		gt.V(t, response).NotEqual(nil)
		gt.V(t, response.TotalCount).Equal(7)
		gt.V(t, len(response.Alerts)).Equal(2) // Only 2 alerts remaining
	})
}

func TestQueuedAlertsSearch_ViaJSONDecode(t *testing.T) {
	// Reproduce the EXACT production flow:
	// 1. JSON body arrives via HTTP → json.Decoder.Decode(&alertData)
	// 2. alertData (any) is stored as QueuedAlert.Data
	// 3. SearchQueuedAlerts should find keywords in that data
	repo := repository.NewMemory()
	uc := usecase.New(usecase.WithRepository(repo))
	resolver := NewResolver(repo, nil, uc)
	ctx := context.Background()

	// Simulate the exact JSON that an SCC alert would send
	rawJSON := `{
		"finding": {
			"canonicalName": "projects/897022298505/sources/1211682098317240122/findings/adac0ac3",
			"category": "SOFTWARE_VULNERABILITY",
			"createTime": "2026-03-19T00:28:20.169Z",
			"description": "A discrepancy between how Go and C/C++ comments were parsed allowed for them",
			"eventTime": "2026-03-24T04:25:09.916873506Z",
			"files": [
				{"diskPath": {}, "path": "bin/operator"},
				{"diskPath": {}, "path": "bin/config-reloader"}
			]
		}
	}`

	// Decode exactly as the HTTP handler does
	var alertData any
	gt.NoError(t, json.NewDecoder(strings.NewReader(rawJSON)).Decode(&alertData))

	qa := &alert.QueuedAlert{
		ID:        types.NewQueuedAlertID(),
		Schema:    "scc",
		Data:      alertData,
		Title:     "",
		CreatedAt: now,
	}
	gt.NoError(t, repo.PutQueuedAlert(ctx, qa))

	// Search by category value
	keyword := "SOFTWARE_VULNERABILITY"
	response, err := resolver.Query().QueuedAlerts(ctx, &keyword, nil, nil)
	gt.NoError(t, err)
	gt.V(t, response.TotalCount).Equal(1)
	gt.A(t, response.Alerts).Length(1)
	gt.V(t, response.Alerts[0].ID).Equal(qa.ID)

	// Verify Data resolver returns valid JSON containing the keyword
	dataStr, err := resolver.QueuedAlert().Data(ctx, response.Alerts[0])
	gt.NoError(t, err)
	gt.True(t, strings.Contains(dataStr, "SOFTWARE_VULNERABILITY"))

	// Search by nested path
	keyword2 := "bin/operator"
	response, err = resolver.Query().QueuedAlerts(ctx, &keyword2, nil, nil)
	gt.NoError(t, err)
	gt.V(t, response.TotalCount).Equal(1)

	// Search by partial string
	keyword3 := "discrepancy"
	response, err = resolver.Query().QueuedAlerts(ctx, &keyword3, nil, nil)
	gt.NoError(t, err)
	gt.V(t, response.TotalCount).Equal(1)
}

func TestQueuedAlertsSearch(t *testing.T) {
	repo := repository.NewMemory()
	uc := usecase.New(usecase.WithRepository(repo))
	resolver := NewResolver(repo, nil, uc)
	ctx := context.Background()

	// Insert queued alerts with realistic nested JSON data
	qa1 := &alert.QueuedAlert{
		ID:     types.NewQueuedAlertID(),
		Schema: "scc",
		Data: map[string]any{
			"finding": map[string]any{
				"category":    "SOFTWARE_VULNERABILITY",
				"description": "CVE-2024-1234 in golang.org/x/net",
				"severity":    "HIGH",
			},
		},
		Title:     "",
		CreatedAt: now,
	}
	qa2 := &alert.QueuedAlert{
		ID:     types.NewQueuedAlertID(),
		Schema: "guardduty",
		Data: map[string]any{
			"detail": map[string]any{
				"type":      "UnauthorizedAccess:EC2/SSHBruteForce",
				"source_ip": "192.168.1.100",
			},
		},
		Title:     "",
		CreatedAt: now,
	}
	qa3 := &alert.QueuedAlert{
		ID:     types.NewQueuedAlertID(),
		Schema: "scc",
		Data: map[string]any{
			"finding": map[string]any{
				"category":    "MISCONFIGURATION",
				"description": "Public bucket access detected",
			},
		},
		Title:     "",
		CreatedAt: now,
	}

	gt.NoError(t, repo.PutQueuedAlert(ctx, qa1))
	gt.NoError(t, repo.PutQueuedAlert(ctx, qa2))
	gt.NoError(t, repo.PutQueuedAlert(ctx, qa3))

	t.Run("search by data content - SOFTWARE_VULNERABILITY", func(t *testing.T) {
		keyword := "SOFTWARE_VULNERABILITY"
		response, err := resolver.Query().QueuedAlerts(ctx, &keyword, nil, nil)
		gt.NoError(t, err)
		gt.V(t, response.TotalCount).Equal(1)
		gt.A(t, response.Alerts).Length(1)
		gt.V(t, response.Alerts[0].ID).Equal(qa1.ID)
	})

	t.Run("search by nested data - SSHBruteForce", func(t *testing.T) {
		keyword := "SSHBruteForce"
		response, err := resolver.Query().QueuedAlerts(ctx, &keyword, nil, nil)
		gt.NoError(t, err)
		gt.V(t, response.TotalCount).Equal(1)
		gt.A(t, response.Alerts).Length(1)
		gt.V(t, response.Alerts[0].ID).Equal(qa2.ID)
	})

	t.Run("search by schema name", func(t *testing.T) {
		keyword := "scc"
		response, err := resolver.Query().QueuedAlerts(ctx, &keyword, nil, nil)
		gt.NoError(t, err)
		gt.V(t, response.TotalCount).Equal(2)
		gt.A(t, response.Alerts).Length(2)
	})

	t.Run("search by IP address in data", func(t *testing.T) {
		keyword := "192.168"
		response, err := resolver.Query().QueuedAlerts(ctx, &keyword, nil, nil)
		gt.NoError(t, err)
		gt.V(t, response.TotalCount).Equal(1)
		gt.V(t, response.Alerts[0].ID).Equal(qa2.ID)
	})

	t.Run("search case-insensitive", func(t *testing.T) {
		keyword := "software_vulnerability"
		response, err := resolver.Query().QueuedAlerts(ctx, &keyword, nil, nil)
		gt.NoError(t, err)
		gt.V(t, response.TotalCount).Equal(1)
	})

	t.Run("search no match", func(t *testing.T) {
		keyword := "nonexistent_keyword_xyz"
		response, err := resolver.Query().QueuedAlerts(ctx, &keyword, nil, nil)
		gt.NoError(t, err)
		gt.V(t, response.TotalCount).Equal(0)
		gt.A(t, response.Alerts).Length(0)
	})

	t.Run("list all without keyword", func(t *testing.T) {
		response, err := resolver.Query().QueuedAlerts(ctx, nil, nil, nil)
		gt.NoError(t, err)
		gt.V(t, response.TotalCount).Equal(3)
		gt.A(t, response.Alerts).Length(3)
	})

	t.Run("verify data field is accessible via resolver", func(t *testing.T) {
		response, err := resolver.Query().QueuedAlerts(ctx, nil, nil, nil)
		gt.NoError(t, err)
		gt.A(t, response.Alerts).Length(3).Required()

		// Verify that Data can be serialized to JSON via the resolver
		for _, qa := range response.Alerts {
			dataStr, err := resolver.QueuedAlert().Data(ctx, qa)
			gt.NoError(t, err)
			gt.V(t, dataStr).NotEqual("")
			gt.V(t, dataStr).NotEqual("null")
		}
	})
}

func TestQueuedAlertsSearchWithDataField(t *testing.T) {
	// This test specifically verifies that json.Marshal on qa.Data works
	// for search matching, which could fail if Data is stored/retrieved
	// in an incompatible format.
	repo := repository.NewMemory()
	ctx := context.Background()

	qa := &alert.QueuedAlert{
		ID:     types.NewQueuedAlertID(),
		Schema: "test",
		Data: map[string]any{
			"deeply": map[string]any{
				"nested": map[string]any{
					"value": "UNIQUE_SEARCH_TOKEN_12345",
				},
			},
		},
		Title:     "",
		CreatedAt: now,
	}
	gt.NoError(t, repo.PutQueuedAlert(ctx, qa))

	// Search by deeply nested value
	results, total, err := repo.SearchQueuedAlerts(ctx, "UNIQUE_SEARCH_TOKEN_12345", 0, 10)
	gt.NoError(t, err)
	gt.V(t, total).Equal(1)
	gt.A(t, results).Length(1)
	gt.V(t, results[0].ID).Equal(qa.ID)
}
