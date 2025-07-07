package vt_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/tool/vt"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

func TestVT(t *testing.T) {
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
			funcName: "vt.domain",
			args: map[string]any{
				"target": "example.com",
			},
			apiResp:    `{"data": {"attributes": {"last_analysis_stats": {"harmless": 5, "malicious": 0}}}}`,
			statusCode: http.StatusOK,
			wantResp: map[string]any{
				"data": map[string]any{
					"attributes": map[string]any{
						"last_analysis_stats": map[string]any{
							"harmless":  float64(5),
							"malicious": float64(0),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "valid ip response",
			funcName: "vt.ip",
			args: map[string]any{
				"target": "8.8.8.8",
			},
			apiResp:    `{"data": {"attributes": {"last_analysis_stats": {"harmless": 10, "malicious": 0}}}}`,
			statusCode: http.StatusOK,
			wantResp: map[string]any{
				"data": map[string]any{
					"attributes": map[string]any{
						"last_analysis_stats": map[string]any{
							"harmless":  float64(10),
							"malicious": float64(0),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "valid file hash response",
			funcName: "vt.file_hash",
			args: map[string]any{
				"target": "44d88612fea8a8f36de82e1278abb02f",
			},
			apiResp:    `{"data": {"attributes": {"last_analysis_stats": {"harmless": 15, "malicious": 0}}}}`,
			statusCode: http.StatusOK,
			wantResp: map[string]any{
				"data": map[string]any{
					"attributes": map[string]any{
						"last_analysis_stats": map[string]any{
							"harmless":  float64(15),
							"malicious": float64(0),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "valid url response",
			funcName: "vt.url",
			args: map[string]any{
				"target": "https://example.com",
			},
			apiResp:    `{"data": {"attributes": {"last_analysis_stats": {"harmless": 20, "malicious": 0}}}}`,
			statusCode: http.StatusOK,
			wantResp: map[string]any{
				"data": map[string]any{
					"attributes": map[string]any{
						"last_analysis_stats": map[string]any{
							"harmless":  float64(20),
							"malicious": float64(0),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "api error response",
			funcName: "vt.domain",
			args: map[string]any{
				"target": "example.com",
			},
			apiResp:    `{"error": {"code": "InvalidArgumentError", "message": "Invalid domain"}}`,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:     "api unauthorized response",
			funcName: "vt.domain",
			args: map[string]any{
				"target": "example.com",
			},
			apiResp:    `{"error": {"code": "AuthenticationRequiredError", "message": "Invalid API key"}}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gt.Value(t, r.Header.Get("x-apikey")).Equal("test-key")

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

			var action vt.Action
			cmd := cli.Command{
				Name:  "vt",
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
				"vt",
				"--vt-api-key", "test-key",
				"--vt-base-url", ts.URL,
			}))
		})
	}
}

func TestVT_Specs(t *testing.T) {
	var action vt.Action
	specs, err := action.Specs(context.Background())
	gt.NoError(t, err)
	gt.A(t, specs).Length(4) // Verify there are 4 tool specifications

	// Verify each tool specification
	for _, spec := range specs {
		gt.Map(t, spec.Parameters).HasKey("target")
		gt.Value(t, spec.Parameters["target"].Type).Equal("string")
	}

	// Verify specific tool specification
	var found bool
	for _, spec := range specs {
		if spec.Name == "vt.ip" {
			found = true
			gt.Value(t, spec.Description).Equal("Search the indicator of IPv4/IPv6 from VirusTotal.")
			break
		}
	}
	gt.Value(t, found).Equal(true)
}

func TestVT_Enabled(t *testing.T) {
	var action vt.Action

	cmd := cli.Command{
		Name:  "vt",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.Equal(t, action.Configure(ctx), errs.ErrActionUnavailable)
			return nil
		},
	}

	t.Setenv("WARREN_VT_API_KEY", "")
	t.Setenv("TEST_VT_API_KEY", "")
	gt.NoError(t, cmd.Run(t.Context(), []string{
		"vt",
		"--vt-base-url", "https://www.virustotal.com/api/v3",
	}))
}

// TestSendRequest tests the Run method of the Action struct.
// It sets up a test environment with a actual API key and target IP address,
// and then runs the command to send a request to the VirusTotal API.
// The test verifies that the request is sent successfully and the response is not nil.
// It also checks that the response type is JSON and contains the expected last_analysis_stats field.
func TestSendRequest(t *testing.T) {
	var act vt.Action

	vars := test.NewEnvVars(t, "TEST_VT_API_KEY", "TEST_VT_TARGET_IPADDR")
	cmd := cli.Command{
		Name:  "vt",
		Flags: act.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			resp, err := act.Run(ctx, "vt.ip", map[string]any{
				"target": vars.Get("TEST_VT_TARGET_IPADDR"),
			})
			gt.NoError(t, err)
			gt.NotEqual(t, resp, nil)
			gt.Map(t, resp).HasKey("data")
			gt.Map(t, resp["data"].(map[string]any)).HasKey("attributes")
			gt.Map(t, resp["data"].(map[string]any)["attributes"].(map[string]any)).HasKey("last_analysis_stats")
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"vt",
		"--vt-api-key", vars.Get("TEST_VT_API_KEY"),
	}))
}
