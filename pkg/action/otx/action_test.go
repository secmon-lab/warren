package otx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/action/otx"
	"github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

func TestOTX(t *testing.T) {
	testCases := []struct {
		name       string
		args       action.Arguments
		apiResp    string
		statusCode int
		wantResp   string
		wantErr    bool
	}{
		{
			name: "valid domain response",
			args: action.Arguments{
				"domain": "example.com",
			},
			apiResp:    `{"pulse_count": 5, "reputation": 0}`,
			statusCode: http.StatusOK,
			wantResp:   `{"pulse_count": 5, "reputation": 0}`,
			wantErr:    false,
		},
		{
			name: "valid ipv4 response",
			args: action.Arguments{
				"ipv4": "8.8.8.8",
			},
			apiResp:    `{"pulse_count": 10, "reputation": 0}`,
			statusCode: http.StatusOK,
			wantResp:   `{"pulse_count": 10, "reputation": 0}`,
			wantErr:    false,
		},
		{
			name:    "missing indicator",
			args:    action.Arguments{},
			wantErr: true,
		},
		{
			name: "api error response",
			args: action.Arguments{
				"domain": "example.com",
			},
			apiResp:    `{"error": "invalid request"}`,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "api unauthorized response",
			args: action.Arguments{
				"domain": "example.com",
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
					resp, err := action.Execute(ctx, nil, nil, tc.args)
					if tc.wantErr {
						gt.Error(t, err)
						return nil
					}

					gt.NoError(t, err)
					gt.Value(t, resp.Data).Equal(tc.wantResp)
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

func TestSendRequest(t *testing.T) {
	var action otx.Action

	vars := test.NewEnvVars(t, "TEST_OTX_API_KEY", "TEST_OTX_TARGET_IPADDR")
	cmd := cli.Command{
		Name:  "otx",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			resp, err := action.Execute(ctx, nil, nil, action.Arguments{"ipv4": vars.Get("TEST_OTX_TARGET_IPADDR")})
			gt.NoError(t, err)
			gt.NotEqual(t, resp, nil)
			gt.Equal(t, resp.Type, action.ActionResultTypeJSON)
			gt.String(t, resp.Data).Contains(`"pulse_info"`)
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"otx",
		"--otx-api-key", vars.Get("TEST_OTX_API_KEY"),
	}))
}
