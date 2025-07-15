package otx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/tool/otx"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

func TestOTX(t *testing.T) {
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
			name:     "valid domain response",
			funcName: "otx_domain",
			args: map[string]any{
				"target": "example.com",
			},
			apiResp:    `{"pulse_count": 5, "reputation": 0}`,
			statusCode: http.StatusOK,
			wantResp: map[string]any{
				"pulse_count": float64(5),
				"reputation":  float64(0),
			},
			wantErr: false,
		},
		{
			name:     "valid ipv4 response",
			funcName: "otx_ipv4",
			args: map[string]any{
				"target": "8.8.8.8",
			},
			apiResp:    `{"pulse_count": 10, "reputation": 0}`,
			statusCode: http.StatusOK,
			wantResp: map[string]any{
				"pulse_count": float64(10),
				"reputation":  float64(0),
			},
			wantErr: false,
		},
		{
			name:     "api error response",
			funcName: "otx_domain",
			args: map[string]any{
				"target": "example.com",
			},
			apiResp:    `{"error": "invalid request"}`,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:     "api unauthorized response",
			funcName: "otx_domain",
			args: map[string]any{
				"target": "example.com",
			},
			apiResp:    `{"error": "unauthorized"}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gt.Value(t, r.Header.Get("X-OTX-API-KEY")).Equal("test-key")

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

			var action otx.Action
			cmd := cli.Command{
				Name:  "otx",
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
				"otx",
				"--otx-api-key", "test-key",
				"--otx-base-url", ts.URL,
			}))
		})
	}
}

func TestOTX_Specs(t *testing.T) {
	var action otx.Action
	specs, err := action.Specs(context.Background())
	gt.NoError(t, err)
	gt.A(t, specs).Length(5) // Verify there are 5 tool specifications

	// Verify each tool specification
	for _, spec := range specs {
		gt.Map(t, spec.Parameters).HasKey("target")
		gt.Value(t, spec.Parameters["target"].Type).Equal("string")
	}

	// Verify specific tool specification
	var found bool
	for _, spec := range specs {
		if spec.Name == "otx_ipv4" {
			found = true
			gt.Value(t, spec.Description).Equal("Search the indicator of IPv4 from OTX.")
			break
		}
	}
	gt.Value(t, found).Equal(true)
}

func TestOTX_Enabled(t *testing.T) {
	var action otx.Action

	cmd := cli.Command{
		Name:  "otx",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.Equal(t, action.Configure(ctx), errs.ErrActionUnavailable)
			return nil
		},
	}

	t.Setenv("WARREN_OTX_API_KEY", "")
	t.Setenv("TEST_OTX_API_KEY", "")
	gt.NoError(t, cmd.Run(t.Context(), []string{
		"otx",
		"--otx-base-url", "https://otx.alienvault.com",
	}))
}

// TestSendRequest tests the Run method of the Action struct.
// It sets up a test environment with a actual API key and target IP address,
// and then runs the command to send a request to the OTX API.
// The test verifies that the request is sent successfully and the response is not nil.
// It also checks that the response type is JSON and contains the expected pulse_info field.
func TestSendRequest(t *testing.T) {
	var act otx.Action

	vars := test.NewEnvVars(t, "TEST_OTX_API_KEY", "TEST_OTX_TARGET_IPADDR")
	cmd := cli.Command{
		Name:  "otx",
		Flags: act.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			resp, err := act.Run(ctx, "otx_ipv4", map[string]any{
				"target": vars.Get("TEST_OTX_TARGET_IPADDR"),
			})
			gt.NoError(t, err)
			gt.NotEqual(t, resp, nil)
			gt.Map(t, resp).HasKey("pulse_info")
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"otx",
		"--otx-api-key", vars.Get("TEST_OTX_API_KEY"),
	}))
}
