package ipdb_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/tool/ipdb"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

func TestIPDB(t *testing.T) {
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
			name:     "valid ip response",
			funcName: "ipdb_check",
			args: map[string]any{
				"target": "8.8.8.8",
			},
			apiResp:    `{"data":{"ipAddress":"8.8.8.8","isPublic":true,"ipVersion":4,"isWhitelisted":false,"abuseConfidenceScore":0,"countryCode":"US","usageType":"Data Center/Web Hosting/Transit","isp":"Google LLC","domain":"google.com","totalReports":0,"numDistinctUsers":0}}`,
			statusCode: http.StatusOK,
			wantResp: map[string]any{
				"data": map[string]any{
					"ipAddress":            "8.8.8.8",
					"isPublic":             true,
					"ipVersion":            float64(4),
					"isWhitelisted":        false,
					"abuseConfidenceScore": float64(0),
					"countryCode":          "US",
					"usageType":            "Data Center/Web Hosting/Transit",
					"isp":                  "Google LLC",
					"domain":               "google.com",
					"totalReports":         float64(0),
					"numDistinctUsers":     float64(0),
				},
			},
			wantErr: false,
		},
		{
			name:     "valid ip response with maxAgeInDays",
			funcName: "ipdb_check",
			args: map[string]any{
				"target":       "8.8.8.8",
				"maxAgeInDays": float64(90),
			},
			apiResp:    `{"data":{"ipAddress":"8.8.8.8","isPublic":true,"ipVersion":4,"isWhitelisted":false,"abuseConfidenceScore":0,"countryCode":"US","usageType":"Data Center/Web Hosting/Transit","isp":"Google LLC","domain":"google.com","totalReports":0,"numDistinctUsers":0}}`,
			statusCode: http.StatusOK,
			wantResp: map[string]any{
				"data": map[string]any{
					"ipAddress":            "8.8.8.8",
					"isPublic":             true,
					"ipVersion":            float64(4),
					"isWhitelisted":        false,
					"abuseConfidenceScore": float64(0),
					"countryCode":          "US",
					"usageType":            "Data Center/Web Hosting/Transit",
					"isp":                  "Google LLC",
					"domain":               "google.com",
					"totalReports":         float64(0),
					"numDistinctUsers":     float64(0),
				},
			},
			wantErr: false,
		},
		{
			name:     "api error response",
			funcName: "ipdb_check",
			args: map[string]any{
				"target": "8.8.8.8",
			},
			apiResp:    `{"errors":[{"detail":"Invalid API key","status":401}]}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:     "invalid ip response",
			funcName: "ipdb_check",
			args: map[string]any{
				"target": "invalid-ip",
			},
			apiResp:    `{"errors":[{"detail":"Invalid IP address","status":422}]}`,
			statusCode: http.StatusUnprocessableEntity,
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gt.Value(t, r.Header.Get("Key")).Equal("test-key")
				gt.Value(t, r.Header.Get("Accept")).Equal("application/json")

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

			var action ipdb.Action
			cmd := cli.Command{
				Name:  "ipdb",
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
				"ipdb",
				"--ipdb-api-key", "test-key",
				"--ipdb-base-url", ts.URL,
			}))
		})
	}
}

func TestIPDB_Specs(t *testing.T) {
	var action ipdb.Action
	specs, err := action.Specs(context.Background())
	gt.NoError(t, err)
	gt.A(t, specs).Length(1) // Verify there is 1 tool specification

	// Verify each tool specification
	for _, spec := range specs {
		gt.Map(t, spec.Parameters).HasKey("target")
		gt.Value(t, spec.Parameters["target"].Type).Equal("string")
	}

	// Verify specific tool specification
	var found bool
	for _, spec := range specs {
		if spec.Name == "ipdb_check" {
			found = true
			gt.Value(t, spec.Description).Equal("Check IP address information from AbuseIPDB.")
			break
		}
	}
	gt.Value(t, found).Equal(true)
}

func TestIPDB_Enabled(t *testing.T) {
	var action ipdb.Action

	cmd := cli.Command{
		Name:  "ipdb",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.Equal(t, action.Configure(ctx), errs.ErrActionUnavailable)
			return nil
		},
	}

	t.Setenv("WARREN_IPDB_API_KEY", "")
	t.Setenv("TEST_IPDB_API_KEY", "")
	gt.NoError(t, cmd.Run(t.Context(), []string{
		"ipdb",
		"--ipdb-base-url", "https://api.abuseipdb.com/api/v2",
	}))
}

// TestSendRequest tests the Run method of the Action struct.
// It sets up a test environment with a actual API key and target IP address,
// and then runs the command to send a request to the AbuseIPDB API.
// The test verifies that the request is sent successfully and the response is not nil.
// It also checks that the response type is JSON and contains the expected data field.
func TestSendRequest(t *testing.T) {
	var act ipdb.Action

	vars := test.NewEnvVars(t, "TEST_IPDB_API_KEY", "TEST_IPDB_TARGET_IPADDR")
	cmd := cli.Command{
		Name:  "ipdb",
		Flags: act.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			resp, err := act.Run(ctx, "ipdb_check", map[string]any{
				"target": vars.Get("TEST_IPDB_TARGET_IPADDR"),
			})
			gt.NoError(t, err)
			gt.NotEqual(t, resp, nil)
			gt.Map(t, resp).HasKey("data")
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"ipdb",
		"--ipdb-api-key", vars.Get("TEST_IPDB_API_KEY"),
	}))
}
