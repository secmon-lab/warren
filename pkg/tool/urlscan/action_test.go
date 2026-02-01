package urlscan_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/urlscan"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

func TestURLScan(t *testing.T) {
	testCases := []struct {
		name       string
		funcName   string
		args       map[string]any
		apiResp    string
		statusCode int
		wantResp   string
		wantErr    bool
	}{
		{
			name:     "valid scan response",
			funcName: "urlscan_scan",
			args: map[string]any{
				"url": "https://example.com",
			},
			apiResp:    `{"uuid": "test-uuid"}`,
			statusCode: http.StatusOK,
			wantResp:   `{"result": "test result"}`,
			wantErr:    false,
		},
		{
			name:     "api error response",
			funcName: "urlscan_scan",
			args: map[string]any{
				"url": "https://example.com",
			},
			apiResp:    `{"error": "invalid request"}`,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:     "api unauthorized response",
			funcName: "urlscan_scan",
			args: map[string]any{
				"url": "https://example.com",
			},
			apiResp:    `{"error": "unauthorized"}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gt.Value(t, r.Header.Get("API-Key")).Equal("test-key")

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

			var action urlscan.Action
			cmd := cli.Command{
				Name:  "urlscan",
				Flags: action.Flags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					resp, err := action.Run(ctx, tc.funcName, tc.args)
					if tc.wantErr {
						gt.Error(t, err)
						return nil
					}

					gt.NoError(t, err)
					gt.NotEqual(t, resp, nil)
					data := gt.Cast[string](t, resp["uuid"])
					gt.Equal(t, data, "test-uuid")
					return nil
				},
			}

			gt.NoError(t, cmd.Run(context.Background(), []string{
				"urlscan",
				"--urlscan-api-key", "test-key",
				"--urlscan-base-url", ts.URL,
			}))
		})
	}
}

func TestURLScan_Specs(t *testing.T) {
	var action urlscan.Action
	specs, err := action.Specs(context.Background())
	gt.NoError(t, err)
	gt.A(t, specs).Length(1) // Verify there is 1 tool specification

	// Verify tool specification
	spec := specs[0]
	gt.Value(t, spec.Name).Equal("urlscan_scan")
	gt.Value(t, spec.Description).Equal("Scan a URL with URLScan")
	gt.Map(t, spec.Parameters).HasKey("url")
	gt.Value(t, spec.Parameters["url"].Type).Equal("string")
}

func TestURLScan_Enabled(t *testing.T) {
	var action urlscan.Action

	cmd := cli.Command{
		Name:  "urlscan",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.Equal(t, action.Configure(ctx), errutil.ErrActionUnavailable)
			return nil
		},
	}

	t.Setenv("WARREN_URLSCAN_API_KEY", "")
	t.Setenv("TEST_URLSCAN_API_KEY", "")
	gt.NoError(t, cmd.Run(t.Context(), []string{
		"urlscan",
		"--urlscan-base-url", "https://urlscan.io/api/v1",
	}))
}

// TestSendRequest tests the Run method of the Action struct.
// It sets up a test environment with a actual API key and target URL,
// and then runs the command to send a request to the URLScan API.
// The test verifies that the request is sent successfully and the response is not nil.
// It also checks that the response type is JSON and contains the expected result field.
func TestSendRequest(t *testing.T) {
	var act urlscan.Action

	vars := test.NewEnvVars(t, "TEST_URLSCAN_API_KEY", "TEST_URLSCAN_TARGET_URL")
	cmd := cli.Command{
		Name:  "urlscan",
		Flags: act.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			resp, err := act.Run(ctx, "urlscan_scan", map[string]any{
				"url": vars.Get("TEST_URLSCAN_TARGET_URL"),
			})
			gt.NoError(t, err)
			gt.NotEqual(t, resp, nil)
			data := gt.Cast[map[string]any](t, resp["data"])
			gt.NotNil(t, data["requests"])
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"urlscan",
		"--urlscan-api-key", vars.Get("TEST_URLSCAN_API_KEY"),
		"--urlscan-timeout", "120s",
	}))
}
