package slack_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/slack"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/urfave/cli/v3"
)

// newAction builds a configured Action pointed at the given stub server URL.
func newAction(t *testing.T, serverURL string) *slack.Action {
	t.Helper()
	action := &slack.Action{}
	action.SetOAuthToken("test-token")
	action.SetTestURL(serverURL)
	gt.NoError(t, action.Configure(context.Background()))
	return action
}

func TestSlackMessageSearch_Delegation(t *testing.T) {
	const successBody = `{
		"ok": true,
		"query": "test",
		"messages": {
			"total": 2,
			"paging": {"count": 2, "total": 2, "page": 1, "pages": 1},
			"matches": [
				{"channel": {"id": "C1", "name": "general"}, "user": "U1", "username": "john", "text": "hello", "ts": "1234567890.123456", "permalink": "https://x/1"},
				{"channel": {"id": "C2", "name": "random"}, "user": "U2", "username": "jane", "text": "world", "ts": "1234567891.123456", "permalink": "https://x/2"}
			]
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gt.S(t, r.Header.Get("Authorization")).Equal("Bearer test-token")
		gt.S(t, r.URL.Path).Equal("/search.messages")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(successBody))
	}))
	defer server.Close()

	action := newAction(t, server.URL)
	result, err := action.Run(context.Background(), "slack_message_search", map[string]any{
		"query": "test",
		"count": float64(20),
	})
	gt.NoError(t, err)
	gt.Number(t, gt.Cast[float64](t, result["total"])).Equal(2)
	messages := gt.Cast[[]any](t, result["messages"])
	gt.A(t, messages).Length(2)

	first := gt.Cast[map[string]any](t, messages[0])
	gt.Value(t, first["channel"]).Equal("C1")
	gt.Value(t, first["channel_name"]).Equal("general")
	gt.Value(t, first["user"]).Equal("U1")
	gt.Value(t, first["text"]).Equal("hello")
}

func TestSlackMessageSearch_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": false, "error": "invalid_auth"}`))
	}))
	defer server.Close()

	action := newAction(t, server.URL)
	_, err := action.Run(context.Background(), "slack_message_search", map[string]any{"query": "test"})
	gt.Error(t, err)
}

func TestSlackMessageSearch_MissingQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true, "messages": {"total": 0, "matches": []}}`))
	}))
	defer server.Close()

	action := newAction(t, server.URL)
	_, err := action.Run(context.Background(), "slack_message_search", map[string]any{"count": float64(10)})
	gt.Error(t, err)
}

func TestSlackMessageSearch_Specs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	action := newAction(t, server.URL)
	specs, err := action.Specs(context.Background())
	gt.NoError(t, err)
	gt.A(t, specs).Length(1)
	gt.Value(t, specs[0].Name).Equal("slack_message_search")
	gt.Map(t, specs[0].Parameters).HasKey("query")
}

func TestSlackConfigure(t *testing.T) {
	t.Run("with token", func(t *testing.T) {
		action := &slack.Action{}
		action.SetOAuthToken("test-token")
		gt.NoError(t, action.Configure(context.Background()))
	})

	t.Run("without token", func(t *testing.T) {
		action := &slack.Action{}
		gt.Equal(t, action.Configure(context.Background()), errutil.ErrActionUnavailable)
	})
}

// TestSlack_FlagSetsToken verifies that parsing --slack-tool-user-token via
// cli.Command.Run binds the value and the configured client sends it as the
// Bearer token on every API request.
func TestSlack_FlagSetsToken(t *testing.T) {
	const successBody = `{"ok": true, "query": "hi", "messages": {"total": 0, "paging": {"count": 0, "total": 0, "page": 1, "pages": 0}, "matches": []}}`

	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(successBody))
	}))
	defer server.Close()

	var action slack.Action
	cmd := cli.Command{
		Name:  "slack",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			action.SetTestURL(server.URL)
			gt.NoError(t, action.Configure(ctx))
			_, err := action.Run(ctx, "slack_message_search", map[string]any{"query": "hi"})
			gt.NoError(t, err)
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"slack",
		"--slack-tool-user-token", "my-secret-token",
	}))
	gt.S(t, gotAuth).Equal("Bearer my-secret-token")
}

func TestSlackMessageSearchIntegration(t *testing.T) {
	token := os.Getenv("TEST_SLACK_USER_TOKEN")
	if token == "" {
		t.Skip("TEST_SLACK_USER_TOKEN not set, skipping integration test")
	}

	action := &slack.Action{}
	action.SetOAuthToken(token)
	ctx := context.Background()
	gt.NoError(t, action.Configure(ctx))

	query := os.Getenv("TEST_SLACK_QUERY")
	if query == "" {
		query = "test"
	}

	result, err := action.Run(ctx, "slack_message_search", map[string]any{
		"query": query,
		"count": float64(10),
	})
	gt.NoError(t, err)
	gt.NotNil(t, result)

	total := gt.Cast[float64](t, result["total"])
	gt.B(t, total >= 0).True()
}
