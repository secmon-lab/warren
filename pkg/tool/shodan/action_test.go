package shodan_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	extshodan "github.com/gollem-dev/tools/shodan"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/shodan"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

// TestShodan_OptionAppended verifies the base-url flag Action appends WithBaseURL
// carrying the provided value, by applying the accumulated options to a fresh
// external ToolSet and reading its unexported field.
func TestShodan_OptionAppended(t *testing.T) {
	var action shodan.Action
	cmd := cli.Command{
		Name:   "shodan",
		Flags:  action.Flags(),
		Action: func(context.Context, *cli.Command) error { return nil },
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{
		"shodan", "--shodan-api-key", "k", "--shodan-base-url", "https://example.test/api",
	}))

	var ts extshodan.ToolSet
	for _, o := range action.Opts() {
		o(&ts)
	}
	gt.Value(t, test.PrivateField(t, &ts, "baseURL")).Equal("https://example.test/api")
}

// TestShodan_Delegation verifies that the warren wrapper builds the external
// toolset on Configure and delegates Run to it (against a stub server).
func TestShodan_Delegation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gt.Value(t, r.URL.Query().Get("key")).Equal("test-key")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"ip_str": "8.8.8.8", "ports": [53]}`)); err != nil {
			t.Fatal("failed to write response:", err)
		}
	}))
	defer ts.Close()

	var action shodan.Action
	cmd := cli.Command{
		Name:  "shodan",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			resp, err := action.Run(ctx, "shodan_host", map[string]any{"target": "8.8.8.8"})
			gt.NoError(t, err)
			gt.Value(t, resp).Equal(map[string]any{
				"ip_str": "8.8.8.8",
				"ports":  []any{float64(53)},
			})
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"shodan",
		"--shodan-api-key", "test-key",
		"--shodan-base-url", ts.URL,
	}))
}

func TestShodan_Specs(t *testing.T) {
	var action shodan.Action
	cmd := cli.Command{
		Name:  "shodan",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			specs, err := action.Specs(ctx)
			gt.NoError(t, err)
			gt.A(t, specs).Length(3)

			var found bool
			for _, spec := range specs {
				if spec.Name == "shodan_host" {
					found = true
				}
			}
			gt.Value(t, found).Equal(true)
			return nil
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{"shodan", "--shodan-api-key", "test-key"}))
}

func TestShodan_Unavailable(t *testing.T) {
	var action shodan.Action
	cmd := cli.Command{
		Name:  "shodan",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.Equal(t, action.Configure(ctx), errutil.ErrActionUnavailable)
			return nil
		},
	}

	t.Setenv("WARREN_SHODAN_API_KEY", "")
	gt.NoError(t, cmd.Run(t.Context(), []string{
		"shodan",
		"--shodan-base-url", "https://api.shodan.io",
	}))
}

// TestSendRequest hits the real Shodan API and only runs when the credentials
// are provided via environment variables.
func TestSendRequest(t *testing.T) {
	var act shodan.Action

	vars := test.NewEnvVars(t, "TEST_SHODAN_API_KEY", "TEST_SHODAN_TARGET_IPADDR")
	cmd := cli.Command{
		Name:  "shodan",
		Flags: act.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, act.Configure(ctx))
			resp, err := act.Run(ctx, "shodan_host", map[string]any{
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
