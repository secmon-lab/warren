package falcon_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/adapter/storage"
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
	gt.Equal(t, count, 10)
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

	// Pagination metadata is present even without storage.
	gt.Equal(t, result["total"].(int), 1)
	gt.Equal(t, result["returned"].(int), 1)
	gt.Equal(t, result["offset"].(int), 0)
	gt.Equal(t, result["limit"].(int), 100)
	gt.Equal(t, result["has_more"].(bool), false)
	// No storage configured -> no result_set_id for further paging.
	_, hasID := result["result_set_id"]
	gt.False(t, hasID)
}

// TestInternalTool_SearchEvents_WithPolling verifies that polls are not
// accumulated: the query job returns the cumulative result set on each poll,
// so only the final (done) response is authoritative and events must not be
// duplicated across polls.
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

			// Each poll returns the cumulative result set computed so far.
			cumulative := []map[string]any{{"aid": "test", "event": "event-1"}}
			if pollCount >= 2 {
				cumulative = append(cumulative, map[string]any{"aid": "test", "event": "event-2"})
			}
			if pollCount >= 3 {
				cumulative = append(cumulative, map[string]any{"aid": "test", "event": "event-3"})
			}

			err := json.NewEncoder(w).Encode(map[string]any{
				"done":      pollCount >= 3,
				"cancelled": false,
				"events":    cumulative,
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

	// Only the final cumulative result is used: exactly 3 distinct events,
	// no duplication from intermediate polls.
	events, ok := result["events"].([]any)
	gt.True(t, ok)
	gt.Equal(t, len(events), 3)
	gt.Equal(t, result["total"].(int), 3)
	gt.Equal(t, pollCount, 3)

	seen := map[string]bool{}
	for _, e := range events {
		ev := e.(map[string]any)
		name := ev["event"].(string)
		gt.False(t, seen[name]) // no duplicates
		seen[name] = true
	}
	gt.True(t, seen["event-1"] && seen["event-2"] && seen["event-3"])

	repo, ok := result["repository"].(string)
	gt.True(t, ok)
	gt.Equal(t, repo, "investigate_view")
}

func TestInternalTool_SearchEvents_MissingQueryString(t *testing.T) {
	tool := falcon.NewInternalToolForTest("id", "secret", "http://localhost")
	_, err := tool.Run(context.Background(), "falcon_search_events", map[string]any{})
	gt.Error(t, err)
}

// eventsSearchServer returns a test server whose query job completes
// immediately with the given number of generated events, and counts how many
// search-related (job create + poll) requests it received.
func eventsSearchServer(t *testing.T, jobID string, numEvents int, eventCount any, hits *int) *httptest.Server {
	t.Helper()
	return newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/queryjobs") {
			*hits++
			w.Header().Set("Content-Type", "application/json")
			gt.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": jobID}))
			return
		}
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/queryjobs/"+jobID) {
			*hits++
			events := make([]map[string]any, numEvents)
			for i := range events {
				events[i] = map[string]any{"idx": i, "name": fmt.Sprintf("event-%d", i)}
			}
			resp := map[string]any{"done": true, "cancelled": false, "events": events}
			if eventCount != nil {
				resp["metaData"] = map[string]any{"eventCount": eventCount}
			}
			w.Header().Set("Content-Type", "application/json")
			gt.NoError(t, json.NewEncoder(w).Encode(resp))
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	})
}

func TestInternalTool_SearchEvents_SnapshotPagination(t *testing.T) {
	var hits int
	srv := eventsSearchServer(t, "job-page", 250, nil, &hits)
	defer srv.Close()

	store := storage.NewMemoryClient()
	tool := falcon.NewInternalToolForTestWithStorage("id", "secret", srv.URL, store, "tenant/")

	// First call: runs the query, snapshots, returns first page.
	first, err := tool.Run(context.Background(), "falcon_search_events", map[string]any{
		"query_string": "#event_simpleName=ProcessRollup2 | tail(250)",
	})
	gt.NoError(t, err)
	gt.Equal(t, first["total"].(int), 250)
	gt.Equal(t, first["returned"].(int), 100)
	gt.Equal(t, first["has_more"].(bool), true)
	rsID := first["result_set_id"].(string)
	gt.Equal(t, rsID, "job-page")
	firstEvents := first["events"].([]any)
	gt.Equal(t, len(firstEvents), 100)
	gt.Equal(t, firstEvents[0].(map[string]any)["idx"].(float64), float64(0))

	hitsAfterFirst := hits

	// Second call: paginate from the snapshot, no re-query.
	second, err := tool.Run(context.Background(), "falcon_search_events", map[string]any{
		"result_set_id": rsID,
		"offset":        100,
	})
	gt.NoError(t, err)
	gt.Equal(t, second["total"].(int), 250)
	gt.Equal(t, second["offset"].(int), 100)
	gt.Equal(t, second["returned"].(int), 100)
	gt.Equal(t, second["has_more"].(bool), true)
	secondEvents := second["events"].([]any)
	gt.Equal(t, len(secondEvents), 100)
	gt.Equal(t, secondEvents[0].(map[string]any)["idx"].(float64), float64(100))

	// Last page.
	third, err := tool.Run(context.Background(), "falcon_search_events", map[string]any{
		"result_set_id": rsID,
		"offset":        200,
	})
	gt.NoError(t, err)
	gt.Equal(t, third["returned"].(int), 50)
	gt.Equal(t, third["has_more"].(bool), false)
	gt.Equal(t, third["events"].([]any)[49].(map[string]any)["idx"].(float64), float64(249))

	// Pagination must not have triggered any additional Falcon API requests.
	gt.Equal(t, hits, hitsAfterFirst)
}

func TestInternalTool_SearchEvents_LimitClamp(t *testing.T) {
	var hits int
	srv := eventsSearchServer(t, "job-clamp", 150, nil, &hits)
	defer srv.Close()

	store := storage.NewMemoryClient()
	tool := falcon.NewInternalToolForTestWithStorage("id", "secret", srv.URL, store, "")

	result, err := tool.Run(context.Background(), "falcon_search_events", map[string]any{
		"query_string": "aid=x | tail(150)",
		"limit":        500, // exceeds max, must clamp to 100
	})
	gt.NoError(t, err)
	gt.Equal(t, result["limit"].(int), 100)
	gt.Equal(t, result["returned"].(int), 100)
	gt.Equal(t, result["total"].(int), 150)
	gt.Equal(t, result["has_more"].(bool), true)
}

func TestInternalTool_SearchEvents_TotalMatchedFromMetadata(t *testing.T) {
	var hits int
	// API returns 200 events but reports 5000 matched via metaData.eventCount.
	srv := eventsSearchServer(t, "job-meta", 200, float64(5000), &hits)
	defer srv.Close()

	store := storage.NewMemoryClient()
	tool := falcon.NewInternalToolForTestWithStorage("id", "secret", srv.URL, store, "")

	result, err := tool.Run(context.Background(), "falcon_search_events", map[string]any{
		"query_string": "aid=x",
	})
	gt.NoError(t, err)
	// total reflects the pageable snapshot size; total_matched reports the
	// true match count from the API.
	gt.Equal(t, result["total"].(int), 200)
	gt.Equal(t, result["total_matched"].(int), 5000)
}

func TestInternalTool_SearchEvents_ResultSetNotFound(t *testing.T) {
	store := storage.NewMemoryClient()
	tool := falcon.NewInternalToolForTestWithStorage("id", "secret", "http://localhost", store, "")

	_, err := tool.Run(context.Background(), "falcon_search_events", map[string]any{
		"result_set_id": "does-not-exist",
		"offset":        0,
	})
	gt.Error(t, err)
}

func TestParseLimit(t *testing.T) {
	gt.Equal(t, falcon.ParseLimit(map[string]any{}), 100)                      // default
	gt.Equal(t, falcon.ParseLimit(map[string]any{"limit": float64(50)}), 50)   // in range
	gt.Equal(t, falcon.ParseLimit(map[string]any{"limit": float64(500)}), 100) // clamp
	gt.Equal(t, falcon.ParseLimit(map[string]any{"limit": float64(0)}), 100)   // invalid -> default
	gt.Equal(t, falcon.ParseLimit(map[string]any{"limit": "30"}), 30)          // string number
	gt.Equal(t, falcon.ParseLimit(map[string]any{"limit": "bad"}), 100)        // unparsable -> default
}

func TestParseOffset(t *testing.T) {
	gt.Equal(t, falcon.ParseOffset(map[string]any{}), 0)
	gt.Equal(t, falcon.ParseOffset(map[string]any{"offset": float64(100)}), 100)
	gt.Equal(t, falcon.ParseOffset(map[string]any{"offset": float64(-5)}), 0) // negative -> 0
	gt.Equal(t, falcon.ParseOffset(map[string]any{"offset": "20"}), 20)
}

func TestPaginate(t *testing.T) {
	events := []any{"a", "b", "c", "d", "e"}

	page, returned, hasMore := falcon.Paginate(events, 0, 2)
	gt.Equal(t, returned, 2)
	gt.Equal(t, hasMore, true)
	gt.Equal(t, page, []any{"a", "b"})

	page, returned, hasMore = falcon.Paginate(events, 4, 2)
	gt.Equal(t, returned, 1)
	gt.Equal(t, hasMore, false)
	gt.Equal(t, page, []any{"e"})

	page, returned, hasMore = falcon.Paginate(events, 10, 2)
	gt.Equal(t, returned, 0)
	gt.Equal(t, hasMore, false)
	gt.Equal(t, len(page), 0)
}

func TestInternalTool_SearchDevices(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gt.Equal(t, r.Method, http.MethodGet)
		gt.True(t, r.URL.Path == "/devices/queries/devices-scroll/v1")
		gt.Equal(t, r.URL.Query().Get("filter"), "platform_name:'Windows'")
		gt.Equal(t, r.URL.Query().Get("limit"), "5")

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"resources": []string{"device-abc123", "device-def456"},
			"meta": map[string]any{
				"pagination": map[string]any{
					"total":  2,
					"offset": "",
				},
			},
		}
		err := json.NewEncoder(w).Encode(resp)
		gt.NoError(t, err)
	})
	defer srv.Close()

	tool := falcon.NewInternalToolForTest("id", "secret", srv.URL)
	result, err := tool.Run(context.Background(), "falcon_search_devices", map[string]any{
		"filter": "platform_name:'Windows'",
		"limit":  float64(5),
	})
	gt.NoError(t, err)

	resources, ok := result["resources"].([]any)
	gt.True(t, ok)
	gt.Equal(t, len(resources), 2)
	gt.Equal(t, resources[0], "device-abc123")
	gt.Equal(t, resources[1], "device-def456")
}

func TestInternalTool_GetDevices(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gt.Equal(t, r.Method, http.MethodPost)
		gt.Equal(t, r.URL.Path, "/devices/entities/devices/v2")
		gt.Equal(t, r.Header.Get("Content-Type"), "application/json")

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		gt.NoError(t, err)

		ids, ok := body["ids"].([]any)
		gt.True(t, ok)
		gt.Equal(t, len(ids), 2)
		gt.Equal(t, ids[0], "device-abc123")
		gt.Equal(t, ids[1], "device-def456")

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"resources": []map[string]any{
				{
					"device_id":     "device-abc123",
					"hostname":      "web-server-01",
					"platform_name": "Windows",
					"os_version":    "Windows 11",
					"external_ip":   "203.0.113.10",
					"local_ip":      "10.0.0.5",
					"status":        "normal",
					"agent_version": "7.10.18207.0",
				},
				{
					"device_id":     "device-def456",
					"hostname":      "db-server-02",
					"platform_name": "Linux",
					"os_version":    "RHEL 9.2",
					"external_ip":   "203.0.113.11",
					"local_ip":      "10.0.0.6",
					"status":        "normal",
					"agent_version": "7.10.18207.0",
				},
			},
		}
		err = json.NewEncoder(w).Encode(resp)
		gt.NoError(t, err)
	})
	defer srv.Close()

	tool := falcon.NewInternalToolForTest("id", "secret", srv.URL)
	result, err := tool.Run(context.Background(), "falcon_get_devices", map[string]any{
		"ids": "device-abc123, device-def456",
	})
	gt.NoError(t, err)

	resources, ok := result["resources"].([]any)
	gt.True(t, ok)
	gt.Equal(t, len(resources), 2)
}

func TestInternalTool_GetDevices_MissingIDs(t *testing.T) {
	tool := falcon.NewInternalToolForTest("id", "secret", "http://localhost")
	_, err := tool.Run(context.Background(), "falcon_get_devices", map[string]any{})
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

func newE2EInternalTool(t *testing.T) *falcon.InternalToolForTest {
	t.Helper()

	clientID := os.Getenv("TEST_AGENT_FALCON_CLIENT_ID")
	clientSecret := os.Getenv("TEST_AGENT_FALCON_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		t.Skip("TEST_AGENT_FALCON_CLIENT_ID and TEST_AGENT_FALCON_CLIENT_SECRET are not set")
	}

	baseURL := os.Getenv("TEST_AGENT_FALCON_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.crowdstrike.com"
	}

	return falcon.NewInternalToolForTest(clientID, clientSecret, baseURL)
}

func TestInternalTool_E2E_SearchDevices(t *testing.T) {
	tool := newE2EInternalTool(t)

	result, err := tool.Run(context.Background(), "falcon_search_devices", map[string]any{
		"limit": float64(5),
	})
	gt.NoError(t, err)

	resources, ok := result["resources"].([]any)
	gt.True(t, ok)
	gt.True(t, len(resources) > 0)
}

func TestInternalTool_E2E_GetDevices(t *testing.T) {
	tool := newE2EInternalTool(t)

	// First, search for device IDs
	searchResult, err := tool.Run(context.Background(), "falcon_search_devices", map[string]any{
		"limit": float64(2),
	})
	gt.NoError(t, err)

	resources, ok := searchResult["resources"].([]any)
	gt.True(t, ok)
	gt.True(t, len(resources) > 0)

	// Build comma-separated IDs from search results
	var ids []string
	for _, r := range resources {
		if id, ok := r.(string); ok {
			ids = append(ids, id)
		}
	}
	gt.True(t, len(ids) > 0)

	// Get device details
	getResult, err := tool.Run(context.Background(), "falcon_get_devices", map[string]any{
		"ids": strings.Join(ids, ","),
	})
	gt.NoError(t, err)

	devices, ok := getResult["resources"].([]any)
	gt.True(t, ok)
	gt.Equal(t, len(devices), len(ids))

	// Verify first device has expected fields
	device, ok := devices[0].(map[string]any)
	gt.True(t, ok)
	_, hasHostname := device["hostname"]
	gt.True(t, hasHostname)
	_, hasDeviceID := device["device_id"]
	gt.True(t, hasDeviceID)
}
