package urlscan_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/urlscan"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

// TestURLScan_Delegation verifies that the warren wrapper builds the external
// toolset on Configure and delegates Run to it (against a stub server).
func TestURLScan_Delegation(t *testing.T) {
	// The external module POSTs to /scan/ and then GETs /result/<uuid>/.
	// We respond to both in this single handler.
	const fakeUUID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	var scanHits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gt.Value(t, r.Header.Get("API-Key")).Equal("test-key")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/scan/":
			w.WriteHeader(http.StatusOK)
			resp := map[string]string{"uuid": fakeUUID, "result": "https://urlscan.io/result/" + fakeUUID + "/"}
			b, _ := json.Marshal(resp)
			_, _ = w.Write(b)

		case r.Method == http.MethodGet && r.URL.Path == "/result/"+fakeUUID+"/":
			scanHits++
			w.WriteHeader(http.StatusOK)
			result := map[string]any{"data": map[string]any{"requests": []any{}}}
			b, _ := json.Marshal(result)
			_, _ = w.Write(b)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	var action urlscan.Action
	cmd := cli.Command{
		Name:  "urlscan",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			resp, err := action.Run(ctx, "urlscan_scan", map[string]any{"url": "https://example.com"})
			gt.NoError(t, err)
			gt.NotEqual(t, resp, nil)
			gt.Map(t, resp).HasKey("data")
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"urlscan",
		"--urlscan-api-key", "test-key",
		"--urlscan-base-url", ts.URL,
		"--urlscan-backoff", "1ms",
	}))
	gt.Value(t, scanHits).Equal(1)
}

func TestURLScan_Specs(t *testing.T) {
	var action urlscan.Action
	cmd := cli.Command{
		Name:  "urlscan",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			specs, err := action.Specs(ctx)
			gt.NoError(t, err)
			gt.A(t, specs).Length(1)

			gt.Value(t, specs[0].Name).Equal("urlscan_scan")
			gt.Map(t, specs[0].Parameters).HasKey("url")
			gt.Value(t, specs[0].Parameters["url"].Type).Equal("string")
			return nil
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{"urlscan", "--urlscan-api-key", "test-key"}))
}

func TestURLScan_Unavailable(t *testing.T) {
	var action urlscan.Action
	cmd := cli.Command{
		Name:  "urlscan",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.Equal(t, action.Configure(ctx), errutil.ErrActionUnavailable)
			return nil
		},
	}

	t.Setenv("WARREN_URLSCAN_API_KEY", "")
	gt.NoError(t, cmd.Run(t.Context(), []string{
		"urlscan",
		"--urlscan-base-url", "https://urlscan.io/api/v1",
	}))
}

// TestSendRequest hits the real urlscan.io API and only runs when credentials
// are provided via environment variables.
func TestSendRequest(t *testing.T) {
	var act urlscan.Action

	vars := test.NewEnvVars(t, "TEST_URLSCAN_API_KEY", "TEST_URLSCAN_TARGET_URL")
	cmd := cli.Command{
		Name:  "urlscan",
		Flags: act.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, act.Configure(ctx))
			resp, err := act.Run(ctx, "urlscan_scan", map[string]any{
				"url": vars.Get("TEST_URLSCAN_TARGET_URL"),
			})
			gt.NoError(t, err)
			gt.NotEqual(t, resp, nil)
			data := gt.Cast[map[string]any](t, resp["data"])
			gt.NotNil(t, data["requests"])
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"urlscan",
		"--urlscan-api-key", vars.Get("TEST_URLSCAN_API_KEY"),
		"--urlscan-timeout", "120s",
	}))
}
