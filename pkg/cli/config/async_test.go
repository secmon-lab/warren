package config_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/urfave/cli/v3"
)

func TestAsyncAlertHook(t *testing.T) {
	t.Run("parses individual values", func(t *testing.T) {
		app := &cli.Command{
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name: "async-alert-hook",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				cfg := &config.AsyncAlertHook{}
				err := cfg.Parse(cmd)
				gt.NoError(t, err)
				gt.True(t, cfg.Raw)
				gt.True(t, cfg.PubSub)
				gt.False(t, cfg.SNS)
				return nil
			},
		}

		err := app.Run(context.Background(), []string{"test", "--async-alert-hook", "raw", "--async-alert-hook", "pubsub"})
		gt.NoError(t, err)
	})

	t.Run("parses all value", func(t *testing.T) {
		app := &cli.Command{
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name: "async-alert-hook",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				cfg := &config.AsyncAlertHook{}
				err := cfg.Parse(cmd)
				gt.NoError(t, err)
				gt.True(t, cfg.Raw)
				gt.True(t, cfg.PubSub)
				gt.True(t, cfg.SNS)
				return nil
			},
		}

		err := app.Run(context.Background(), []string{"test", "--async-alert-hook", "all"})
		gt.NoError(t, err)
	})

	t.Run("returns error for invalid value", func(t *testing.T) {
		app := &cli.Command{
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name: "async-alert-hook",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				cfg := &config.AsyncAlertHook{}
				err := cfg.Parse(cmd)
				gt.Error(t, err)
				gt.S(t, err.Error()).Contains("invalid async-alert-hook value")
				return nil
			},
		}

		err := app.Run(context.Background(), []string{"test", "--async-alert-hook", "invalid"})
		gt.NoError(t, err)
	})

	t.Run("defaults to all false", func(t *testing.T) {
		app := &cli.Command{
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name: "async-alert-hook",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				cfg := &config.AsyncAlertHook{}
				err := cfg.Parse(cmd)
				gt.NoError(t, err)
				gt.False(t, cfg.Raw)
				gt.False(t, cfg.PubSub)
				gt.False(t, cfg.SNS)
				return nil
			},
		}

		err := app.Run(context.Background(), []string{"test"})
		gt.NoError(t, err)
	})
}
