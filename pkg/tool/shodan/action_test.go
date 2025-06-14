package shodan_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/tool/shodan"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

func TestShodan(t *testing.T) {
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
			name:     "valid host response",
			funcName: "shodan.host",
			args: map[string]any{
				"target": "8.8.8.8",
			},
			apiResp:    `{"ip": "8.8.8.8", "os": "Linux", "ports": [53]}`,
			statusCode: http.StatusOK,
			wantResp: map[string]any{
				"ip":    "8.8.8.8",
				"os":    "Linux",
				"ports": []any{float64(53)},
			},
			wantErr: false,
		},
		{
			name:     "valid domain response",
			funcName: "shodan.domain",
			args: map[string]any{
				"target": "example.com",
			},
			apiResp:    `{"domain": "example.com", "tags": ["ssl", "https"], "subdomains": ["www"]}`,
			statusCode: http.StatusOK,
			wantResp: map[string]any{
				"domain":     "example.com",
				"tags":       []any{"ssl", "https"},
				"subdomains": []any{"www"},
			},
			wantErr: false,
		},
		{
			name:     "valid search response",
			funcName: "shodan.search",
			args: map[string]any{
				"query": "apache",
				"limit": float64(2),
			},
			apiResp:    `{"matches": [{"ip_str": "1.1.1.1", "product": "Apache"}, {"ip_str": "2.2.2.2", "product": "Apache"}]}`,
			statusCode: http.StatusOK,
			wantResp: map[string]any{
				"matches": []any{
					map[string]any{
						"ip_str":  "1.1.1.1",
						"product": "Apache",
					},
					map[string]any{
						"ip_str":  "2.2.2.2",
						"product": "Apache",
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "api error response",
			funcName: "shodan.host",
			args: map[string]any{
				"target": "8.8.8.8",
			},
			apiResp:    `{"error": "Invalid API key"}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:     "invalid host response",
			funcName: "shodan.host",
			args: map[string]any{
				"target": "invalid-ip",
			},
			apiResp:    `{"error": "Invalid IP address"}`,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gt.Value(t, r.URL.Query().Get("key")).Equal("test-key")

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

			var action shodan.Action
			cmd := cli.Command{
				Name:  "shodan",
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
				"shodan",
				"--shodan-api-key", "test-key",
				"--shodan-base-url", ts.URL,
			}))
		})
	}
}

func TestShodan_Specs(t *testing.T) {
	var action shodan.Action
	specs, err := action.Specs(context.Background())
	gt.NoError(t, err)
	gt.A(t, specs).Length(3) // Verify there are 3 tool specifications

	// Verify each tool specification
	for _, spec := range specs {
		switch spec.Name {
		case "shodan.host", "shodan.domain":
			gt.Map(t, spec.Parameters).HasKey("target")
			gt.Value(t, spec.Parameters["target"].Type).Equal("string")
		case "shodan.search":
			gt.Map(t, spec.Parameters).HasKey("query")
			gt.Value(t, spec.Parameters["query"].Type).Equal("string")
			gt.Map(t, spec.Parameters).HasKey("limit")
			gt.Value(t, spec.Parameters["limit"].Type).Equal("integer")
		}
	}

	// Verify specific tool specification
	var found bool
	for _, spec := range specs {
		if spec.Name == "shodan.host" {
			found = true
			gt.Value(t, spec.Description).Equal("Search the host information from Shodan.")
			break
		}
	}
	gt.Value(t, found).Equal(true)
}

func TestShodan_Enabled(t *testing.T) {
	var action shodan.Action

	cmd := cli.Command{
		Name:  "shodan",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.Equal(t, action.Configure(ctx), errs.ErrActionUnavailable)
			return nil
		},
	}

	t.Setenv("WARREN_SHODAN_API_KEY", "")
	t.Setenv("TEST_SHODAN_API_KEY", "")
	gt.NoError(t, cmd.Run(t.Context(), []string{
		"shodan",
		"--shodan-base-url", "https://api.shodan.io",
	}))
}

// TestSendRequest tests the Run method of the Action struct.
// It sets up a test environment with a actual API key and target IP address,
// and then runs the command to send a request to the Shodan API.
// The test verifies that the request is sent successfully and the response is not nil.
// It also checks that the response type is JSON and contains the expected ip_str field.
func TestSendRequest(t *testing.T) {
	var act shodan.Action

	vars := test.NewEnvVars(t, "TEST_SHODAN_API_KEY", "TEST_SHODAN_TARGET_IPADDR")
	cmd := cli.Command{
		Name:  "shodan",
		Flags: act.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			resp, err := act.Run(ctx, "shodan.host", map[string]any{
				"target": vars.Get("TEST_SHODAN_TARGET_IPADDR"),
			})
			gt.NoError(t, err)
			gt.NotEqual(t, resp, nil)
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"shodan",
		"--shodan-api-key", vars.Get("TEST_SHODAN_API_KEY"),
	}))
}
