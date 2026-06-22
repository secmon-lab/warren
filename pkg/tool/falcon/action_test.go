package falcon_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/falcon"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

// newStubServer returns an httptest server that answers the Falcon OAuth2 token
// endpoint and returns an empty (but well-formed) search response for any other
// path. capturedAuth receives the Authorization header of the last API call.
func newStubServer(t *testing.T, capturedAuth *string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/oauth2/token" {
			_, _ = w.Write([]byte(`{"access_token":"test-token","expires_in":1800,"token_type":"bearer"}`))
			return
		}
		if capturedAuth != nil {
			*capturedAuth = r.Header.Get("Authorization")
		}
		_, _ = w.Write([]byte(`{"resources":[],"meta":{"pagination":{"total":0}}}`))
	}))
	t.Cleanup(server.Close)
	return server
}

func newConfiguredAction(t *testing.T, serverURL string) *falcon.Action {
	t.Helper()
	action := &falcon.Action{}
	action.SetCredentials("test-client-id", "test-client-secret")
	action.SetTestURL(serverURL)
	gt.NoError(t, action.Configure(context.Background()))
	return action
}

func TestFalcon_Configure(t *testing.T) {
	t.Run("with credentials", func(t *testing.T) {
		action := &falcon.Action{}
		action.SetCredentials("id", "secret")
		gt.NoError(t, action.Configure(context.Background()))
	})

	t.Run("missing client id", func(t *testing.T) {
		action := &falcon.Action{}
		action.SetCredentials("", "secret")
		gt.Equal(t, action.Configure(context.Background()), errutil.ErrActionUnavailable)
	})

	t.Run("missing client secret", func(t *testing.T) {
		action := &falcon.Action{}
		action.SetCredentials("id", "")
		gt.Equal(t, action.Configure(context.Background()), errutil.ErrActionUnavailable)
	})
}

func TestFalcon_Specs(t *testing.T) {
	server := newStubServer(t, nil)
	action := newConfiguredAction(t, server.URL)

	specs, err := action.Specs(context.Background())
	gt.NoError(t, err)
	gt.A(t, specs).Length(10)

	got := map[string]bool{}
	for _, s := range specs {
		got[s.Name] = true
	}
	for _, name := range []string{
		"falcon_search_incidents",
		"falcon_get_incidents",
		"falcon_search_alerts",
		"falcon_get_alerts",
		"falcon_search_behaviors",
		"falcon_get_behaviors",
		"falcon_search_devices",
		"falcon_get_devices",
		"falcon_get_crowdscores",
		"falcon_search_events",
	} {
		gt.True(t, got[name])
	}
}

func TestFalcon_RunDelegation(t *testing.T) {
	var gotAuth string
	server := newStubServer(t, &gotAuth)
	action := newConfiguredAction(t, server.URL)

	result, err := action.Run(context.Background(), "falcon_search_incidents", map[string]any{})
	gt.NoError(t, err)
	gt.NotNil(t, result)
	// The token fetched from /oauth2/token must be sent as a Bearer credential.
	gt.Value(t, gotAuth).Equal("Bearer test-token")
}

func TestFalcon_RunNotConfigured(t *testing.T) {
	action := &falcon.Action{}
	_, err := action.Run(context.Background(), "falcon_search_incidents", map[string]any{})
	gt.Error(t, err)

	_, err = action.Specs(context.Background())
	gt.Error(t, err)
}

func TestFalcon_Identity(t *testing.T) {
	action := &falcon.Action{}
	gt.Value(t, action.ID()).Equal("falcon")
	gt.True(t, action.Description() != "")
	prompt, err := action.Prompt(context.Background())
	gt.NoError(t, err)
	gt.Value(t, prompt).Equal("")
}

// TestFalcon_FlagBinding verifies that --falcon-client-id / --falcon-client-secret
// bind via cli.Command parsing and that the resulting tool uses them: the stub
// server is reached and answers a search call.
func TestFalcon_FlagBinding(t *testing.T) {
	var gotAuth string
	server := newStubServer(t, &gotAuth)

	action := &falcon.Action{}
	cmd := cli.Command{
		Name:  "falcon",
		Flags: action.Flags(),
		Action: func(ctx context.Context, _ *cli.Command) error {
			// Point at the stub before configuring; the parsed flags supply credentials.
			action.SetTestURL(server.URL)
			gt.NoError(t, action.Configure(ctx))
			_, err := action.Run(ctx, "falcon_search_devices", map[string]any{})
			gt.NoError(t, err)
			return nil
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{
		"falcon",
		"--falcon-client-id", "flag-client-id",
		"--falcon-client-secret", "flag-client-secret",
	}))

	gt.Value(t, gotAuth).Equal("Bearer test-token")
}

func TestFalcon_Integration(t *testing.T) {
	vars := test.NewEnvVars(t, "TEST_FALCON_CLIENT_ID", "TEST_FALCON_CLIENT_SECRET")

	action := &falcon.Action{}
	cmd := cli.Command{
		Name:  "falcon",
		Flags: action.Flags(),
		Action: func(ctx context.Context, _ *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			resp, err := action.Run(ctx, "falcon_search_incidents", map[string]any{})
			gt.NoError(t, err)
			gt.NotNil(t, resp)
			return nil
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{
		"falcon",
		"--falcon-client-id", vars.Get("TEST_FALCON_CLIENT_ID"),
		"--falcon-client-secret", vars.Get("TEST_FALCON_CLIENT_SECRET"),
	}))
}
