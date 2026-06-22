package jira_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/jira"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

const (
	testEmail = "user@example.com"
	testToken = "test-api-token"
)

// newStubServer returns an httptest server that answers the Jira project search
// endpoint with an empty result and records the Authorization header.
func newStubServer(t *testing.T, capturedAuth *string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capturedAuth != nil {
			*capturedAuth = r.Header.Get("Authorization")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"isLast":true,"total":0,"values":[]}`))
	}))
	t.Cleanup(server.Close)
	return server
}

func newConfiguredAction(t *testing.T, serverURL string) *jira.Action {
	t.Helper()
	action := &jira.Action{}
	action.SetConfig(serverURL, testEmail, testToken)
	gt.NoError(t, action.Configure(context.Background()))
	return action
}

func TestJira_Configure(t *testing.T) {
	t.Run("with all settings", func(t *testing.T) {
		action := &jira.Action{}
		action.SetConfig("https://example.atlassian.net", testEmail, testToken)
		gt.NoError(t, action.Configure(context.Background()))
	})

	t.Run("missing base url", func(t *testing.T) {
		action := &jira.Action{}
		action.SetConfig("", testEmail, testToken)
		gt.Equal(t, action.Configure(context.Background()), errutil.ErrActionUnavailable)
	})

	t.Run("missing email", func(t *testing.T) {
		action := &jira.Action{}
		action.SetConfig("https://example.atlassian.net", "", testToken)
		gt.Equal(t, action.Configure(context.Background()), errutil.ErrActionUnavailable)
	})

	t.Run("missing api token", func(t *testing.T) {
		action := &jira.Action{}
		action.SetConfig("https://example.atlassian.net", testEmail, "")
		gt.Equal(t, action.Configure(context.Background()), errutil.ErrActionUnavailable)
	})
}

func TestJira_Specs(t *testing.T) {
	server := newStubServer(t, nil)
	action := newConfiguredAction(t, server.URL)

	specs, err := action.Specs(context.Background())
	gt.NoError(t, err)
	gt.A(t, specs).Length(3)

	got := map[string]bool{}
	for _, s := range specs {
		got[s.Name] = true
	}
	for _, name := range []string{
		"jira_list_projects",
		"jira_search_issues",
		"jira_get_issues",
	} {
		gt.True(t, got[name])
	}
}

func TestJira_RunDelegation(t *testing.T) {
	var gotAuth string
	server := newStubServer(t, &gotAuth)
	action := newConfiguredAction(t, server.URL)

	result, err := action.Run(context.Background(), "jira_list_projects", map[string]any{})
	gt.NoError(t, err)
	gt.NotNil(t, result)

	// Basic auth header must be base64("email:apiToken").
	wantCred := base64.StdEncoding.EncodeToString([]byte(testEmail + ":" + testToken))
	gt.Value(t, gotAuth).Equal("Basic " + wantCred)
}

func TestJira_RunNotConfigured(t *testing.T) {
	action := &jira.Action{}
	_, err := action.Run(context.Background(), "jira_list_projects", map[string]any{})
	gt.Error(t, err)

	_, err = action.Specs(context.Background())
	gt.Error(t, err)
}

func TestJira_Identity(t *testing.T) {
	action := &jira.Action{}
	gt.Value(t, action.ID()).Equal("jira")
	gt.True(t, action.Description() != "")
	prompt, err := action.Prompt(context.Background())
	gt.NoError(t, err)
	gt.Value(t, prompt).Equal("")
}

// TestJira_FlagBinding verifies that --jira-base-url / --jira-user-email /
// --jira-api-token bind via cli.Command parsing and reach the configured tool.
func TestJira_FlagBinding(t *testing.T) {
	var gotAuth string
	server := newStubServer(t, &gotAuth)

	action := &jira.Action{}
	cmd := cli.Command{
		Name:  "jira",
		Flags: action.Flags(),
		Action: func(ctx context.Context, _ *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			_, err := action.Run(ctx, "jira_list_projects", map[string]any{})
			gt.NoError(t, err)
			return nil
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{
		"jira",
		"--jira-base-url", server.URL,
		"--jira-user-email", testEmail,
		"--jira-api-token", testToken,
	}))

	wantCred := base64.StdEncoding.EncodeToString([]byte(testEmail + ":" + testToken))
	gt.Value(t, gotAuth).Equal("Basic " + wantCred)
}

func TestJira_Integration(t *testing.T) {
	vars := test.NewEnvVars(t, "TEST_JIRA_BASE_URL", "TEST_JIRA_USER_EMAIL", "TEST_JIRA_API_TOKEN")

	action := &jira.Action{}
	cmd := cli.Command{
		Name:  "jira",
		Flags: action.Flags(),
		Action: func(ctx context.Context, _ *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			resp, err := action.Run(ctx, "jira_list_projects", map[string]any{})
			gt.NoError(t, err)
			gt.NotNil(t, resp)
			return nil
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{
		"jira",
		"--jira-base-url", vars.Get("TEST_JIRA_BASE_URL"),
		"--jira-user-email", vars.Get("TEST_JIRA_USER_EMAIL"),
		"--jira-api-token", vars.Get("TEST_JIRA_API_TOKEN"),
	}))
}
