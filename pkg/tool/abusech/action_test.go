package abusech_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/abusech"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

func TestAbusech(t *testing.T) {
	testCases := []struct {
		name       string
		funcName   string
		args       map[string]any
		apiResp    string
		statusCode int
		wantResp   map[string]any
		wantErr    bool
	}{
		{
			name:     "valid hash response",
			funcName: "abusech.bazaar.query",
			args: map[string]any{
				"hash": "test-hash",
			},
			apiResp:    `{"query_status": "ok", "data": [{"sha256_hash": "test-hash", "file_type": "exe", "signature": "test-signature"}]}`,
			statusCode: http.StatusOK,
			wantResp: map[string]any{
				"query_status": "ok",
				"data": []any{
					map[string]any{
						"sha256_hash": "test-hash",
						"file_type":   "exe",
						"signature":   "test-signature",
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "api error response",
			funcName: "abusech.bazaar.query",
			args: map[string]any{
				"hash": "test-hash",
			},
			apiResp:    `{"query_status": "error", "error_message": "invalid request"}`,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:     "api unauthorized response",
			funcName: "abusech.bazaar.query",
			args: map[string]any{
				"hash": "test-hash",
			},
			apiResp:    `{"query_status": "error", "error_message": "unauthorized"}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gt.Value(t, r.Header.Get("Auth-Key")).Equal("test-key")
				gt.Value(t, r.Header.Get("Content-Type")).Equal("application/x-www-form-urlencoded")

				if tc.statusCode == 0 {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(tc.statusCode)
				if _, err := w.Write([]byte(tc.apiResp)); err != nil {
					t.Fatal("failed to write response:", err)
				}
			}))
			defer ts.Close()

			var action abusech.Action
			cmd := cli.Command{
				Name:  "abusech",
				Flags: action.Flags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					resp, err := action.Run(ctx, tc.funcName, tc.args)
					if tc.wantErr {
						gt.Error(t, err)
						return nil
					}

					gt.NoError(t, err)
					gt.NotEqual(t, resp, nil)
					gt.Value(t, resp).Equal(tc.wantResp)
					return nil
				},
			}

			gt.NoError(t, cmd.Run(context.Background(), []string{
				"abusech",
				"--abusech-api-key", "test-key",
				"--abusech-base-url", ts.URL,
			}))
		})
	}
}

func TestAbusech_Specs(t *testing.T) {
	var action abusech.Action
	specs, err := action.Specs(context.Background())
	gt.NoError(t, err)
	gt.A(t, specs).Length(1) // Verify there is 1 tool specification

	// Verify each tool specification
	for _, spec := range specs {
		gt.Map(t, spec.Parameters).HasKey("hash")
		gt.Value(t, spec.Parameters["hash"].Type).Equal("string")
	}

	// Verify specific tool specification
	var found bool
	for _, spec := range specs {
		if spec.Name == "abusech.bazaar.query" {
			found = true
			gt.Value(t, spec.Description).Equal("Query malware information from MalwareBazaar by file hash value.")
			break
		}
	}
	gt.Value(t, found).Equal(true)
}

func TestAbusech_Enabled(t *testing.T) {
	var action abusech.Action

	cmd := cli.Command{
		Name:  "abusech",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			err := action.Configure(ctx)
			if err == nil {
				t.Error("expected error, but got nil")
				return nil
			}
			gt.Value(t, err.Error()).Equal("action is not available")
			return nil
		},
	}

	// Clear environment variables
	t.Setenv("WARREN_ABUSECH_AUTH_KEY", "")

	// Explicitly set flags
	gt.NoError(t, cmd.Run(t.Context(), []string{
		"abusech",
		"--abusech-api-key", "", // Explicitly set empty API key
		"--abusech-base-url", "https://mb-api.abuse.ch/api/v1",
	}))
}

// TestSendRequest tests the Run method of the Action struct with actual API.
func TestSendRequest(t *testing.T) {
	var act abusech.Action

	vars := test.NewEnvVars(t, "TEST_ABUSECH_AUTH_KEY")
	cmd := cli.Command{
		Name:  "abusech",
		Flags: act.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			// Using a known malware hash value
			hash := "7de2c1bf58bce09eecc70476747d88a26163c3d6bb1d85235c24a558d1f16754"
			resp, err := act.Run(ctx, "abusech.bazaar.query", map[string]any{
				"hash": hash,
			})
			if err != nil {
				t.Logf("Response error: %+v", err)
				return err
			}

			gt.NotEqual(t, resp, nil)
			gt.Map(t, resp).HasKey("query_status")
			gt.Map(t, resp).HasKey("data")

			status, ok := resp["query_status"].(string)
			gt.Value(t, ok).Equal(true)
			gt.Value(t, status).Equal("ok")

			data, ok := resp["data"].([]interface{})
			gt.Value(t, ok).Equal(true)
			gt.Value(t, len(data)).Equal(1)

			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"abusech",
		"--abusech-api-key", vars.Get("TEST_ABUSECH_AUTH_KEY"),
	}))
}
