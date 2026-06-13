package abusech_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	extabusech "github.com/gollem-dev/tools/abusech"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/abusech"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

// TestAbusech_OptionAppended verifies the base-url flag Action appends WithBaseURL
// carrying the provided value, by applying the accumulated options to a fresh
// external ToolSet and reading its unexported field.
func TestAbusech_OptionAppended(t *testing.T) {
	var action abusech.Action
	cmd := cli.Command{
		Name:   "abusech",
		Flags:  action.Flags(),
		Action: func(context.Context, *cli.Command) error { return nil },
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{
		"abusech", "--abusech-api-key", "k", "--abusech-base-url", "https://example.test/api",
	}))

	var ts extabusech.ToolSet
	for _, o := range action.Opts() {
		o(&ts)
	}
	gt.Value(t, test.PrivateField(t, &ts, "baseURL")).Equal("https://example.test/api")
}

func TestAbusech_Unavailable(t *testing.T) {
	var action abusech.Action
	cmd := cli.Command{
		Name:  "abusech",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.Equal(t, action.Configure(ctx), errutil.ErrActionUnavailable)
			return nil
		},
	}

	t.Setenv("WARREN_ABUSECH_AUTH_KEY", "")
	gt.NoError(t, cmd.Run(t.Context(), []string{
		"abusech",
		"--abusech-base-url", "https://mb-api.abuse.ch/api/v1",
	}))
}

func TestAbusech_Specs(t *testing.T) {
	var action abusech.Action
	cmd := cli.Command{
		Name:  "abusech",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			specs, err := action.Specs(ctx)
			gt.NoError(t, err)
			gt.A(t, specs).Length(1)

			var found bool
			for _, spec := range specs {
				if spec.Name == "abusech.bazaar.query" {
					found = true
					gt.Map(t, spec.Parameters).HasKey("hash")
					gt.Value(t, spec.Parameters["hash"].Type).Equal("string")
				}
			}
			gt.Value(t, found).Equal(true)
			return nil
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{"abusech", "--abusech-api-key", "test-key"}))
}

// TestAbusech_Delegation verifies that the warren wrapper builds the external
// toolset on Configure and delegates Run to it (against a stub server).
func TestAbusech_Delegation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gt.Value(t, r.Header.Get("Auth-Key")).Equal("test-key")
		gt.Value(t, r.Header.Get("Content-Type")).Equal("application/x-www-form-urlencoded")
		gt.NoError(t, r.ParseForm())
		gt.Value(t, r.FormValue("query")).Equal("get_info")
		gt.Value(t, r.FormValue("hash")).Equal("abc123")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"query_status":"ok","data":[{"sha256_hash":"abc123"}]}`)); err != nil {
			t.Fatal("failed to write response:", err)
		}
	}))
	defer ts.Close()

	var action abusech.Action
	cmd := cli.Command{
		Name:  "abusech",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			resp, err := action.Run(ctx, "abusech.bazaar.query", map[string]any{"hash": "abc123"})
			gt.NoError(t, err)
			gt.NotEqual(t, resp, nil)
			gt.Value(t, resp["query_status"]).Equal("ok")
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"abusech",
		"--abusech-api-key", "test-key",
		"--abusech-base-url", ts.URL,
	}))
}

// TestAbusech_Live hits the real abuse.ch API and only runs when credentials
// are provided via environment variables.
func TestAbusech_Live(t *testing.T) {
	var act abusech.Action

	vars := test.NewEnvVars(t, "TEST_ABUSECH_AUTH_KEY")
	cmd := cli.Command{
		Name:  "abusech",
		Flags: act.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, act.Configure(ctx))
			// Use a known malware hash present in MalwareBazaar.
			hash := "7de2c1bf58bce09eecc70476747d88a26163c3d6bb1d85235c24a558d1f16754"
			resp, err := act.Run(ctx, "abusech.bazaar.query", map[string]any{"hash": hash})
			gt.NoError(t, err)
			gt.NotEqual(t, resp, nil)
			gt.Map(t, resp).HasKey("query_status")
			gt.Map(t, resp).HasKey("data")
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"abusech",
		"--abusech-api-key", vars.Get("TEST_ABUSECH_AUTH_KEY"),
	}))
}
