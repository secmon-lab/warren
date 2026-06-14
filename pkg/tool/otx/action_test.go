package otx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/otx"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

// TestOTX_Delegation verifies that the warren wrapper builds the external
// toolset on Configure and delegates Run to it (against a stub server).
func TestOTX_Delegation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gt.Value(t, r.Header.Get("X-OTX-API-KEY")).Equal("test-key")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"pulse_count": 5, "reputation": 0}`)); err != nil {
			t.Fatal("failed to write response:", err)
		}
	}))
	defer ts.Close()

	var action otx.Action
	cmd := cli.Command{
		Name:  "otx",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			resp, err := action.Run(ctx, "otx_domain", map[string]any{"target": "example.com"})
			gt.NoError(t, err)
			gt.Value(t, resp).Equal(map[string]any{
				"pulse_count": float64(5),
				"reputation":  float64(0),
			})
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"otx",
		"--otx-api-key", "test-key",
		"--otx-base-url", ts.URL,
	}))
}

func TestOTX_Specs(t *testing.T) {
	var action2 otx.Action
	cmd := cli.Command{
		Name:  "otx",
		Flags: action2.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action2.Configure(ctx))
			specs, err := action2.Specs(ctx)
			gt.NoError(t, err)
			gt.A(t, specs).Length(5)

			var found bool
			for _, spec := range specs {
				gt.Map(t, spec.Parameters).HasKey("target")
				gt.Value(t, spec.Parameters["target"].Type).Equal("string")
				if spec.Name == "otx_ipv4" {
					found = true
				}
			}
			gt.Value(t, found).Equal(true)
			return nil
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{"otx", "--otx-api-key", "test-key"}))
}

func TestOTX_Unavailable(t *testing.T) {
	var action otx.Action
	cmd := cli.Command{
		Name:  "otx",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.Equal(t, action.Configure(ctx), errutil.ErrActionUnavailable)
			return nil
		},
	}

	t.Setenv("WARREN_OTX_API_KEY", "")
	gt.NoError(t, cmd.Run(t.Context(), []string{
		"otx",
		"--otx-base-url", "https://otx.alienvault.com",
	}))
}

// TestSendRequest hits the real OTX API and only runs when the credentials are
// provided via environment variables.
func TestSendRequest(t *testing.T) {
	var act otx.Action

	vars := test.NewEnvVars(t, "TEST_OTX_API_KEY", "TEST_OTX_TARGET_IPADDR")
	cmd := cli.Command{
		Name:  "otx",
		Flags: act.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, act.Configure(ctx))
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
