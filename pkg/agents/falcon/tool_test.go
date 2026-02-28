package falcon_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/agents/falcon"
)

// newTestServer creates a test server that handles both OAuth2 token and API requests.
// The apiHandler is called for all non-token requests.
func newTestServer(t *testing.T, apiHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle OAuth2 token request
		if r.URL.Path == "/oauth2/token" {
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"access_token": "test-token",
				"expires_in":   1800,
				"token_type":   "bearer",
			}
			err := json.NewEncoder(w).Encode(resp)
			gt.NoError(t, err)
			return
		}

		// Verify authorization header
		auth := r.Header.Get("Authorization")
		gt.Equal(t, auth, "Bearer test-token")

		apiHandler(w, r)
	}))
}

func TestInternalTool_SpecCount(t *testing.T) {
	tool := falcon.NewInternalToolForTest("id", "secret", "http://localhost")
	count, err := tool.SpecCount(context.Background())
	gt.NoError(t, err)
	gt.Equal(t, count, 8)
}

func TestInternalTool_SearchIncidents(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gt.Equal(t, r.Method, http.MethodGet)
		gt.True(t, r.URL.Path == "/incidents/queries/incidents/v1")
		gt.Equal(t, r.URL.Query().Get("filter"), "status:'30'")
		gt.Equal(t, r.URL.Query().Get("limit"), "10")

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"resources": []string{"inc:abc:123", "inc:abc:456"},
			"meta": map[string]any{
				"pagination": map[string]any{
					"total": 2,
				},
			},
		}
		err := json.NewEncoder(w).Encode(resp)
		gt.NoError(t, err)
	})
	defer srv.Close()

	tool := falcon.NewInternalToolForTest("id", "secret", srv.URL)
	result, err := tool.Run(context.Background(), "falcon_search_incidents", map[string]any{
		"filter": "status:'30'",
		"limit":  float64(10),
	})
	gt.NoError(t, err)

	resources, ok := result["resources"].([]any)
	gt.True(t, ok)
	gt.Equal(t, len(resources), 2)
	gt.Equal(t, resources[0], "inc:abc:123")
	gt.Equal(t, resources[1], "inc:abc:456")
}

func TestInternalTool_GetIncidents(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gt.Equal(t, r.Method, http.MethodPost)
		gt.Equal(t, r.URL.Path, "/incidents/entities/incidents/GET/v1")
		gt.Equal(t, r.Header.Get("Content-Type"), "application/json")

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		gt.NoError(t, err)

		ids, ok := body["ids"].([]any)
		gt.True(t, ok)
		gt.Equal(t, len(ids), 2)
		gt.Equal(t, ids[0], "inc:abc:123")
		gt.Equal(t, ids[1], "inc:abc:456")

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"resources": []map[string]any{
				{"incident_id": "inc:abc:123", "status": 30},
				{"incident_id": "inc:abc:456", "status": 20},
			},
		}
		err = json.NewEncoder(w).Encode(resp)
		gt.NoError(t, err)
	})
	defer srv.Close()

	tool := falcon.NewInternalToolForTest("id", "secret", srv.URL)
	result, err := tool.Run(context.Background(), "falcon_get_incidents", map[string]any{
		"ids": "inc:abc:123, inc:abc:456",
	})
	gt.NoError(t, err)

	resources, ok := result["resources"].([]any)
	gt.True(t, ok)
	gt.Equal(t, len(resources), 2)
}

func TestInternalTool_SearchAlerts(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gt.Equal(t, r.Method, http.MethodPost)
		gt.Equal(t, r.URL.Path, "/alerts/combined/alerts/v1")
		gt.Equal(t, r.Header.Get("Content-Type"), "application/json")

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		gt.NoError(t, err)

		gt.Equal(t, body["filter"], "severity:>50")
		limit, ok := body["limit"].(float64)
		gt.True(t, ok)
		gt.Equal(t, limit, float64(100))

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"resources": []map[string]any{
				{"composite_id": "alert:1", "severity": 80},
			},
		}
		err = json.NewEncoder(w).Encode(resp)
		gt.NoError(t, err)
	})
	defer srv.Close()

	tool := falcon.NewInternalToolForTest("id", "secret", srv.URL)
	result, err := tool.Run(context.Background(), "falcon_search_alerts", map[string]any{
		"filter": "severity:>50",
		"limit":  float64(100),
	})
	gt.NoError(t, err)

	resources, ok := result["resources"].([]any)
	gt.True(t, ok)
	gt.Equal(t, len(resources), 1)
}

func TestInternalTool_GetAlerts(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gt.Equal(t, r.Method, http.MethodPost)
		gt.Equal(t, r.URL.Path, "/alerts/entities/alerts/v2")

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		gt.NoError(t, err)

		compositeIDs, ok := body["composite_ids"].([]any)
		gt.True(t, ok)
		gt.Equal(t, len(compositeIDs), 1)
		gt.Equal(t, compositeIDs[0], "alert:1")

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"resources": []map[string]any{
				{"composite_id": "alert:1", "severity": 80, "status": "new"},
			},
		}
		err = json.NewEncoder(w).Encode(resp)
		gt.NoError(t, err)
	})
	defer srv.Close()

	tool := falcon.NewInternalToolForTest("id", "secret", srv.URL)
	result, err := tool.Run(context.Background(), "falcon_get_alerts", map[string]any{
		"composite_ids": "alert:1",
	})
	gt.NoError(t, err)

	resources, ok := result["resources"].([]any)
	gt.True(t, ok)
	gt.Equal(t, len(resources), 1)
}

func TestInternalTool_SearchBehaviors(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gt.Equal(t, r.Method, http.MethodGet)
		gt.True(t, r.URL.Path == "/incidents/queries/behaviors/v1")
		gt.Equal(t, r.URL.Query().Get("filter"), "tactic:'Lateral Movement'")

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"resources": []string{"beh:abc:123"},
		}
		err := json.NewEncoder(w).Encode(resp)
		gt.NoError(t, err)
	})
	defer srv.Close()

	tool := falcon.NewInternalToolForTest("id", "secret", srv.URL)
	result, err := tool.Run(context.Background(), "falcon_search_behaviors", map[string]any{
		"filter": "tactic:'Lateral Movement'",
	})
	gt.NoError(t, err)

	resources, ok := result["resources"].([]any)
	gt.True(t, ok)
	gt.Equal(t, len(resources), 1)
	gt.Equal(t, resources[0], "beh:abc:123")
}

func TestInternalTool_GetBehaviors(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gt.Equal(t, r.Method, http.MethodPost)
		gt.Equal(t, r.URL.Path, "/incidents/entities/behaviors/GET/v1")

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		gt.NoError(t, err)

		ids, ok := body["ids"].([]any)
		gt.True(t, ok)
		gt.Equal(t, len(ids), 1)
		gt.Equal(t, ids[0], "beh:abc:123")

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"resources": []map[string]any{
				{"behavior_id": "beh:abc:123", "tactic": "Lateral Movement"},
			},
		}
		err = json.NewEncoder(w).Encode(resp)
		gt.NoError(t, err)
	})
	defer srv.Close()

	tool := falcon.NewInternalToolForTest("id", "secret", srv.URL)
	result, err := tool.Run(context.Background(), "falcon_get_behaviors", map[string]any{
		"ids": "beh:abc:123",
	})
	gt.NoError(t, err)

	resources, ok := result["resources"].([]any)
	gt.True(t, ok)
	gt.Equal(t, len(resources), 1)
}

func TestInternalTool_GetCrowdScores(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gt.Equal(t, r.Method, http.MethodGet)
		gt.True(t, r.URL.Path == "/incidents/combined/crowdscores/v1")

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"resources": []map[string]any{
				{"id": "score-1", "score": 42, "adjusted_score": 45},
			},
		}
		err := json.NewEncoder(w).Encode(resp)
		gt.NoError(t, err)
	})
	defer srv.Close()

	tool := falcon.NewInternalToolForTest("id", "secret", srv.URL)
	result, err := tool.Run(context.Background(), "falcon_get_crowdscores", map[string]any{})
	gt.NoError(t, err)

	resources, ok := result["resources"].([]any)
	gt.True(t, ok)
	gt.Equal(t, len(resources), 1)
}

func TestInternalTool_UnknownTool(t *testing.T) {
	tool := falcon.NewInternalToolForTest("id", "secret", "http://localhost")
	_, err := tool.Run(context.Background(), "falcon_unknown", map[string]any{})
	gt.Error(t, err)
}

func TestInternalTool_GetIncidents_MissingIDs(t *testing.T) {
	tool := falcon.NewInternalToolForTest("id", "secret", "http://localhost")
	_, err := tool.Run(context.Background(), "falcon_get_incidents", map[string]any{})
	gt.Error(t, err)
}

func TestInternalTool_GetAlerts_MissingCompositeIDs(t *testing.T) {
	tool := falcon.NewInternalToolForTest("id", "secret", "http://localhost")
	_, err := tool.Run(context.Background(), "falcon_get_alerts", map[string]any{})
	gt.Error(t, err)
}

func TestInternalTool_GetBehaviors_MissingIDs(t *testing.T) {
	tool := falcon.NewInternalToolForTest("id", "secret", "http://localhost")
	_, err := tool.Run(context.Background(), "falcon_get_behaviors", map[string]any{})
	gt.Error(t, err)
}

func TestInternalTool_TokenRetryOn401(t *testing.T) {
	var callCount int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"access_token": "new-token",
				"expires_in":   1800,
				"token_type":   "bearer",
			}
			err := json.NewEncoder(w).Encode(resp)
			gt.NoError(t, err)
			return
		}

		callCount++
		if callCount == 1 {
			// First API call returns 401
			w.WriteHeader(http.StatusUnauthorized)
			_, err := w.Write([]byte(`{"errors": [{"message": "access denied"}]}`))
			gt.NoError(t, err)
			return
		}

		// Second API call succeeds
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"resources": []string{"inc:retry:123"},
		}
		err := json.NewEncoder(w).Encode(resp)
		gt.NoError(t, err)
	}))
	defer srv.Close()

	tool := falcon.NewInternalToolForTest("id", "secret", srv.URL)
	result, err := tool.Run(context.Background(), "falcon_search_incidents", map[string]any{})
	gt.NoError(t, err)

	resources, ok := result["resources"].([]any)
	gt.True(t, ok)
	gt.Equal(t, len(resources), 1)
	gt.Equal(t, resources[0], "inc:retry:123")
	gt.Equal(t, callCount, 2)
}

func TestInternalTool_SearchEvents_Immediate(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/queryjobs") {
			// Create query job
			gt.Equal(t, r.Header.Get("Content-Type"), "application/json")

			var body map[string]any
			err := json.NewDecoder(r.Body).Decode(&body)
			gt.NoError(t, err)
			gt.Equal(t, body["queryString"], "aid=test123")
			gt.Equal(t, body["start"], "1d")
			gt.Equal(t, body["end"], "now")

			w.Header().Set("Content-Type", "application/json")
			err = json.NewEncoder(w).Encode(map[string]any{
				"id":                "job-123",
				"hashedQueryOnView": "abc",
			})
			gt.NoError(t, err)
			return
		}

		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/queryjobs/job-123") {
			// Get results — immediately done
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(map[string]any{
				"done":      true,
				"cancelled": false,
				"events": []map[string]any{
					{
						"timestamp":         "1736264422005",
						"#event_simpleName": "ProcessRollup2",
						"aid":               "test123",
						"ComputerName":      "workstation1",
						"FileName":          "cmd.exe",
					},
				},
			})
			gt.NoError(t, err)
			return
		}

		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	})
	defer srv.Close()

	tool := falcon.NewInternalToolForTest("id", "secret", srv.URL)
	result, err := tool.Run(context.Background(), "falcon_search_events", map[string]any{
		"query_string": "aid=test123",
	})
	gt.NoError(t, err)

	done, ok := result["done"].(bool)
	gt.True(t, ok)
	gt.True(t, done)

	events, ok := result["events"].([]any)
	gt.True(t, ok)
	gt.Equal(t, len(events), 1)
}

func TestInternalTool_SearchEvents_WithPolling(t *testing.T) {
	var pollCount int

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/queryjobs") {
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(map[string]any{
				"id": "job-poll",
			})
			gt.NoError(t, err)
			return
		}

		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/queryjobs/job-poll") {
			pollCount++
			w.Header().Set("Content-Type", "application/json")

			if pollCount < 3 {
				// Not done yet, return partial results
				err := json.NewEncoder(w).Encode(map[string]any{
					"done":      false,
					"cancelled": false,
					"events": []map[string]any{
						{"aid": "test", "event": fmt.Sprintf("event-%d", pollCount)},
					},
				})
				gt.NoError(t, err)
				return
			}

			// Done on third poll
			err := json.NewEncoder(w).Encode(map[string]any{
				"done":      true,
				"cancelled": false,
				"events": []map[string]any{
					{"aid": "test", "event": "event-final"},
				},
			})
			gt.NoError(t, err)
			return
		}

		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	})
	defer srv.Close()

	tool := falcon.NewInternalToolForTest("id", "secret", srv.URL)
	result, err := tool.Run(context.Background(), "falcon_search_events", map[string]any{
		"query_string": "aid=test",
		"repository":   "investigate_view",
		"start":        "7d",
		"end":          "now",
	})
	gt.NoError(t, err)

	done, ok := result["done"].(bool)
	gt.True(t, ok)
	gt.True(t, done)

	// Should have accumulated events from all polls
	events, ok := result["events"].([]any)
	gt.True(t, ok)
	gt.Equal(t, len(events), 3) // 1 from poll 1 + 1 from poll 2 + 1 from final
	gt.Equal(t, pollCount, 3)

	repo, ok := result["repository"].(string)
	gt.True(t, ok)
	gt.Equal(t, repo, "investigate_view")
}

func TestInternalTool_SearchEvents_MissingQueryString(t *testing.T) {
	tool := falcon.NewInternalToolForTest("id", "secret", "http://localhost")
	_, err := tool.Run(context.Background(), "falcon_search_events", map[string]any{})
	gt.Error(t, err)
}

func TestInternalTool_APIError(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, err := w.Write([]byte(`{"errors": [{"message": "rate limit exceeded"}]}`))
		gt.NoError(t, err)
	})
	defer srv.Close()

	tool := falcon.NewInternalToolForTest("id", "secret", srv.URL)
	_, err := tool.Run(context.Background(), "falcon_search_incidents", map[string]any{})
	gt.Error(t, err)
}
