package config_test

import (
	"context"
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/urfave/cli/v3"
)

func TestWebUIConfig(t *testing.T) {
	ctx := context.Background()

	// Save and clear environment variables to avoid interference
	origClientID := os.Getenv("WARREN_SLACK_CLIENT_ID")
	origClientSecret := os.Getenv("WARREN_SLACK_CLIENT_SECRET")
	origFrontendURL := os.Getenv("WARREN_FRONTEND_URL")
	origNoAuth := os.Getenv("WARREN_NO_AUTHENTICATION")

	os.Unsetenv("WARREN_SLACK_CLIENT_ID")
	os.Unsetenv("WARREN_SLACK_CLIENT_SECRET")
	os.Unsetenv("WARREN_FRONTEND_URL")
	os.Unsetenv("WARREN_NO_AUTHENTICATION")

	defer func() {
		if origClientID != "" {
			os.Setenv("WARREN_SLACK_CLIENT_ID", origClientID)
		}
		if origClientSecret != "" {
			os.Setenv("WARREN_SLACK_CLIENT_SECRET", origClientSecret)
		}
		if origFrontendURL != "" {
			os.Setenv("WARREN_FRONTEND_URL", origFrontendURL)
		}
		if origNoAuth != "" {
			os.Setenv("WARREN_NO_AUTHENTICATION", origNoAuth)
		}
	}()

	t.Run("IsConfigured with no-authn", func(t *testing.T) {
		cfg := &config.WebUI{}

		// Not configured without any settings
		gt.False(t, cfg.IsConfigured())

		// Create a test CLI app to set flags
		app := &cli.Command{
			Flags: cfg.Flags(),
			Action: func(ctx context.Context, c *cli.Command) error {
				// Test that no-authn makes it configured
				return nil
			},
		}

		// Test with no-authn flag
		err := app.Run(ctx, []string{"test", "--no-authn"})
		gt.NoError(t, err)
		gt.True(t, cfg.IsConfigured())
	})

	t.Run("Configure with Slack auth overrides no-authn", func(t *testing.T) {
		cfg := &config.WebUI{}
		repo := repository.NewMemory()

		// Create a test CLI app
		app := &cli.Command{
			Flags: cfg.Flags(),
			Action: func(ctx context.Context, c *cli.Command) error {
				authUC, err := cfg.Configure(t.Context(), repo, nil)
				gt.NoError(t, err)
				gt.NotNil(t, authUC)

				// Should use regular auth, not no-auth
				url := authUC.GetAuthURL("test-state")
				gt.True(t, len(url) > 1) // Should not be "/" for regular auth
				return nil
			},
		}

		// Test with both Slack auth and no-authn
		err := app.Run(ctx, []string{"test",
			"--slack-client-id", "test-id",
			"--slack-client-secret", "test-secret",
			"--frontend-url", "http://localhost:3000",
			"--no-authn",
		})
		gt.NoError(t, err)
	})

	t.Run("Configure with only no-authn", func(t *testing.T) {
		cfg := &config.WebUI{}
		repo := repository.NewMemory()

		// Create a test CLI app
		app := &cli.Command{
			Flags: cfg.Flags(),
			Action: func(ctx context.Context, c *cli.Command) error {
				authUC, err := cfg.Configure(t.Context(), repo, nil)
				gt.NoError(t, err)
				gt.NotNil(t, authUC)

				// Should use no-auth
				url := authUC.GetAuthURL("test-state")
				gt.Equal(t, url, "/")
				return nil
			},
		}

		// Test with only no-authn
		err := app.Run(ctx, []string{"test", "--no-authn"})
		gt.NoError(t, err)
	})

	t.Run("Configure without any auth returns nil", func(t *testing.T) {
		cfg := &config.WebUI{}
		repo := repository.NewMemory()

		authUC, err := cfg.Configure(t.Context(), repo, nil)
		gt.NoError(t, err)
		gt.Nil(t, authUC)
	})
}
