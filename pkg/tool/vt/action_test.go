package vt_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/vt"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

// TestVT_Delegation verifies that the warren wrapper builds the external
// toolset on Configure and delegates Run to it (against a stub server).
func TestVT_Delegation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gt.Value(t, r.Header.Get("x-apikey")).Equal("test-key")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"data": {"attributes": {"last_analysis_stats": {"harmless": 5, "malicious": 0}}}}`)); err != nil {
			t.Fatal("failed to write response:", err)
		}
	}))
	defer ts.Close()

	var action vt.Action
	cmd := cli.Command{
		Name:  "vt",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			resp, err := action.Run(ctx, "vt_domain", map[string]any{"target": "example.com"})
			gt.NoError(t, err)
			gt.Value(t, resp).Equal(map[string]any{
				"data": map[string]any{
					"attributes": map[string]any{
						"last_analysis_stats": map[string]any{
							"harmless":  float64(5),
							"malicious": float64(0),
						},
					},
				},
			})
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"vt",
		"--vt-api-key", "test-key",
		"--vt-base-url", ts.URL,
	}))
}

func TestVT_Specs(t *testing.T) {
	var action vt.Action
	cmd := cli.Command{
		Name:  "vt",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			specs, err := action.Specs(ctx)
			gt.NoError(t, err)
			gt.A(t, specs).Length(4)

			var found bool
			for _, spec := range specs {
				gt.Map(t, spec.Parameters).HasKey("target")
				gt.Value(t, spec.Parameters["target"].Type).Equal("string")
				if spec.Name == "vt_ip" {
					found = true
				}
			}
			gt.Value(t, found).Equal(true)
			return nil
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{"vt", "--vt-api-key", "test-key"}))
}

func TestVT_Unavailable(t *testing.T) {
	var action vt.Action
	cmd := cli.Command{
		Name:  "vt",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.Equal(t, action.Configure(ctx), errutil.ErrActionUnavailable)
			return nil
		},
	}

	t.Setenv("WARREN_VT_API_KEY", "")
	gt.NoError(t, cmd.Run(t.Context(), []string{
		"vt",
		"--vt-base-url", "https://www.virustotal.com/api/v3",
	}))
}

// TestSendRequest hits the real VirusTotal API and only runs when the credentials
// are provided via environment variables.
func TestSendRequest(t *testing.T) {
	var act vt.Action

	vars := test.NewEnvVars(t, "TEST_VT_API_KEY", "TEST_VT_TARGET_IPADDR")
	cmd := cli.Command{
		Name:  "vt",
		Flags: act.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, act.Configure(ctx))
			resp, err := act.Run(ctx, "vt_ip", map[string]any{
				"target": vars.Get("TEST_VT_TARGET_IPADDR"),
			})
			gt.NoError(t, err)
			gt.NotEqual(t, resp, nil)
			gt.Map(t, resp).HasKey("data")
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"vt",
		"--vt-api-key", vars.Get("TEST_VT_API_KEY"),
	}))
}
