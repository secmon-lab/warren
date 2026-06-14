package ipdb_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/ipdb"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

// TestIPDB_Delegation verifies that the warren wrapper builds the external
// toolset on Configure and delegates Run to it (against a stub server).
func TestIPDB_Delegation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gt.Value(t, r.Header.Get("Key")).Equal("test-key")
		gt.Value(t, r.Header.Get("Accept")).Equal("application/json")
		gt.Value(t, r.URL.Query().Get("ipAddress")).Equal("8.8.8.8")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"data":{"ipAddress":"8.8.8.8","abuseConfidenceScore":0}}`)); err != nil {
			t.Fatal("failed to write response:", err)
		}
	}))
	defer ts.Close()

	var action ipdb.Action
	cmd := cli.Command{
		Name:  "ipdb",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			resp, err := action.Run(ctx, "ipdb_check", map[string]any{"target": "8.8.8.8"})
			gt.NoError(t, err)
			gt.Value(t, resp).Equal(map[string]any{
				"data": map[string]any{
					"ipAddress":            "8.8.8.8",
					"abuseConfidenceScore": float64(0),
				},
			})
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"ipdb",
		"--ipdb-api-key", "test-key",
		"--ipdb-base-url", ts.URL,
	}))
}

func TestIPDB_Specs(t *testing.T) {
	var action ipdb.Action
	cmd := cli.Command{
		Name:  "ipdb",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			specs, err := action.Specs(ctx)
			gt.NoError(t, err)
			gt.A(t, specs).Length(1)

			var found bool
			for _, spec := range specs {
				gt.Map(t, spec.Parameters).HasKey("target")
				gt.Value(t, spec.Parameters["target"].Type).Equal("string")
				if spec.Name == "ipdb_check" {
					found = true
					gt.Map(t, spec.Parameters).HasKey("maxAgeInDays")
				}
			}
			gt.Value(t, found).Equal(true)
			return nil
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{"ipdb", "--ipdb-api-key", "test-key"}))
}

func TestIPDB_Unavailable(t *testing.T) {
	var action ipdb.Action
	cmd := cli.Command{
		Name:  "ipdb",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.Equal(t, action.Configure(ctx), errutil.ErrActionUnavailable)
			return nil
		},
	}

	t.Setenv("WARREN_IPDB_API_KEY", "")
	gt.NoError(t, cmd.Run(t.Context(), []string{
		"ipdb",
		"--ipdb-base-url", "https://api.abuseipdb.com/api/v2",
	}))
}

// TestSendRequest hits the real AbuseIPDB API and only runs when credentials
// are provided via environment variables.
func TestSendRequest(t *testing.T) {
	var act ipdb.Action

	vars := test.NewEnvVars(t, "TEST_IPDB_API_KEY", "TEST_IPDB_TARGET_IPADDR")
	cmd := cli.Command{
		Name:  "ipdb",
		Flags: act.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, act.Configure(ctx))
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
