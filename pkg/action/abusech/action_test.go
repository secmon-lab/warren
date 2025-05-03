package abusech_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/action/abusech"
)

func TestAction(t *testing.T) {
	type testCase struct {
		name     string
		args     map[string]any
		handler  http.HandlerFunc
		expected map[string]any
		hasError bool
	}

	runTest := func(tc testCase) func(t *testing.T) {
		return func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			action := abusech.New()
			action.SetAPIKey("test-key")
			action.SetBaseURL(server.URL)

			result, err := action.Run(context.Background(), "abusech.bazaar.query", tc.args)

			if tc.hasError {
				gt.Error(t, err)
				return
			}

			gt.NoError(t, err)
			if tc.expected != nil {
				gt.Value(t, result).Equal(tc.expected)
			}
		}
	}

	t.Run("success case - valid hash query", runTest(testCase{
		name: "valid hash query",
		args: map[string]any{
			"hash": "test-hash",
		},
		handler: func(w http.ResponseWriter, r *http.Request) {
			gt.Equal(t, r.Method, "POST")
			gt.Equal(t, r.Header.Get("Auth-Key"), "test-key")
			gt.Equal(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")

			err := r.ParseForm()
			gt.NoError(t, err)
			gt.Equal(t, r.Form.Get("query"), "get_info")
			gt.Equal(t, r.Form.Get("hash"), "test-hash")

			response := map[string]interface{}{
				"query_status": "ok",
				"data": []map[string]interface{}{
					{
						"sha256_hash": "test-hash",
						"file_type":   "exe",
						"signature":   "test-signature",
						"tags":        []string{"malware", "test"},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		},
		expected: map[string]any{
			"query_status": "ok",
			"data": []any{
				map[string]any{
					"sha256_hash": "test-hash",
					"file_type":   "exe",
					"signature":   "test-signature",
					"tags":        []any{"malware", "test"},
				},
			},
		},
	}))

	t.Run("error case - invalid function name", runTest(testCase{
		name:     "invalid function name",
		args:     map[string]any{},
		handler:  func(w http.ResponseWriter, r *http.Request) {},
		hasError: true,
	}))

	t.Run("error case - missing API key", runTest(testCase{
		name: "missing API key",
		args: map[string]any{
			"hash": "test-hash",
		},
		handler:  func(w http.ResponseWriter, r *http.Request) {},
		hasError: true,
	}))

	t.Run("error case - server error", runTest(testCase{
		name: "server error",
		args: map[string]any{
			"hash": "test-hash",
		},
		handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		},
		hasError: true,
	}))

	t.Run("error case - invalid JSON response", runTest(testCase{
		name: "invalid JSON response",
		args: map[string]any{
			"hash": "test-hash",
		},
		handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))
		},
		hasError: true,
	}))

	t.Run("error case - API error response", runTest(testCase{
		name: "API error response",
		args: map[string]any{
			"hash": "test-hash",
		},
		handler: func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"query_status": "error",
				"error":        "Invalid hash format",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		},
		hasError: true,
	}))

	t.Run("error case - missing hash parameter", runTest(testCase{
		name:     "missing hash parameter",
		args:     map[string]any{},
		handler:  func(w http.ResponseWriter, r *http.Request) {},
		hasError: true,
	}))
}

func TestConfigure(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		action := abusech.New()
		action.SetAPIKey("test-key")
		err := action.Configure(context.Background())
		gt.NoError(t, err)
	})

	t.Run("error case - missing API key", func(t *testing.T) {
		action := abusech.New()
		err := action.Configure(context.Background())
		gt.Error(t, err)
	})

	t.Run("error case - invalid base URL", func(t *testing.T) {
		action := abusech.New()
		action.SetAPIKey("test-key")
		action.SetBaseURL("invalid://url")
		err := action.Configure(context.Background())
		gt.Error(t, err)
	})
}

func TestLogValue(t *testing.T) {
	action := abusech.New()
	action.SetAPIKey("test-key")
	action.SetBaseURL("https://test.example.com")

	value := action.LogValue()
	gt.Value(t, value.Kind()).Equal(slog.KindGroup)
}
