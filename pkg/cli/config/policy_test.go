package config_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/urfave/cli/v3"
)

type policyTestOutput struct {
	Alerts []map[string]any `json:"alerts"`
}

func TestPolicy_HasPolicies(t *testing.T) {
	t.Run("empty when nothing configured", func(t *testing.T) {
		cfg := &config.Policy{}
		gt.False(t, cfg.HasPolicies())
	})

	t.Run("true when file path is configured", func(t *testing.T) {
		cfg := &config.Policy{}
		app := &cli.Command{
			Flags: cfg.Flags(),
			Action: func(_ context.Context, _ *cli.Command) error {
				return nil
			},
		}
		err := app.Run(context.Background(), []string{"test", "--policy", "testdata/policy"})
		gt.NoError(t, err)
		gt.True(t, cfg.HasPolicies())
	})
}

func TestPolicy_Configure_FileSource(t *testing.T) {
	t.Run("loads .rego file from directory and evaluates", func(t *testing.T) {
		cfg := &config.Policy{}
		app := &cli.Command{
			Flags: cfg.Flags(),
			Action: func(_ context.Context, _ *cli.Command) error {
				return nil
			},
		}
		err := app.Run(context.Background(), []string{"test", "--policy", "testdata/policy"})
		gt.NoError(t, err)

		client, err := cfg.Configure()
		gt.NoError(t, err)
		gt.NotNil(t, client)

		input := map[string]any{
			"event_type": "test",
			"title":      "hello",
		}
		var out policyTestOutput
		err = client.Query(context.Background(), "data.ingest.test", input, &out)
		gt.NoError(t, err)
		gt.A(t, out.Alerts).Length(1)
		gt.Equal(t, out.Alerts[0]["title"], "hello")
	})

	t.Run("loads .rego file from single file path", func(t *testing.T) {
		cfg := &config.Policy{}
		app := &cli.Command{
			Flags: cfg.Flags(),
			Action: func(_ context.Context, _ *cli.Command) error {
				return nil
			},
		}
		err := app.Run(context.Background(), []string{"test", "--policy", "testdata/policy/sample.rego"})
		gt.NoError(t, err)

		client, err := cfg.Configure()
		gt.NoError(t, err)
		gt.NotNil(t, client)

		input := map[string]any{
			"event_type": "test",
			"title":      "world",
		}
		var out policyTestOutput
		err = client.Query(context.Background(), "data.ingest.test", input, &out)
		gt.NoError(t, err)
		gt.A(t, out.Alerts).Length(1)
		gt.Equal(t, out.Alerts[0]["title"], "world")
	})

	t.Run("returns error for non-existent path", func(t *testing.T) {
		cfg := &config.Policy{}
		app := &cli.Command{
			Flags: cfg.Flags(),
			Action: func(_ context.Context, _ *cli.Command) error {
				return nil
			},
		}
		err := app.Run(context.Background(), []string{"test", "--policy", "testdata/does-not-exist"})
		gt.NoError(t, err)

		_, err = cfg.Configure()
		gt.Error(t, err)
	})
}

func TestPolicy_Configure_NoSources_ReturnsEmptyClient(t *testing.T) {
	cfg := &config.Policy{}
	gt.False(t, cfg.HasPolicies())

	client, err := cfg.Configure()
	gt.NoError(t, err)
	gt.NotNil(t, client)
	gt.M(t, client.Sources()).Length(0)
}

func TestPolicy_Configure_GitHubFlagValidation(t *testing.T) {
	t.Run("invalid repo format rejected", func(t *testing.T) {
		cfg := &config.Policy{}
		app := &cli.Command{
			Flags: cfg.Flags(),
			Action: func(_ context.Context, _ *cli.Command) error {
				return nil
			},
		}
		err := app.Run(context.Background(), []string{
			"test",
			"--policy-github-repo", "no-slash",
			"--policy-github-app-id", "1",
			"--policy-github-app-installation-id", "2",
			"--policy-github-app-private-key", "dummy",
		})
		gt.NoError(t, err)

		_, err = cfg.Configure()
		gt.Error(t, err)
	})

	t.Run("missing app credentials rejected", func(t *testing.T) {
		cfg := &config.Policy{}
		app := &cli.Command{
			Flags: cfg.Flags(),
			Action: func(_ context.Context, _ *cli.Command) error {
				return nil
			},
		}
		err := app.Run(context.Background(), []string{
			"test",
			"--policy-github-repo", "owner/repo",
			// app credentials intentionally omitted
		})
		gt.NoError(t, err)

		_, err = cfg.Configure()
		gt.Error(t, err)
	})

	t.Run("HasPolicies true when github repo set", func(t *testing.T) {
		cfg := &config.Policy{}
		app := &cli.Command{
			Flags: cfg.Flags(),
			Action: func(_ context.Context, _ *cli.Command) error {
				return nil
			},
		}
		err := app.Run(context.Background(), []string{
			"test",
			"--policy-github-repo", "owner/repo",
		})
		gt.NoError(t, err)
		gt.True(t, cfg.HasPolicies())
	})
}
