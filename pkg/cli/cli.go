package cli

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func Run(ctx context.Context, args []string) error {
	var loggerCfg config.Logger
	var closer func()
	app := &cli.Command{
		Name:  "warren",
		Usage: "warren",
		Flags: loggerCfg.Flags(),
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			f, err := loggerCfg.Configure()
			if err != nil {
				return ctx, goerr.Wrap(err, "failed to configure logger")
			}
			closer = f

			return ctx, nil
		},
		After: func(ctx context.Context, c *cli.Command) error {
			if closer != nil {
				closer()
			}
			return nil
		},
		Commands: []*cli.Command{
			cmdServe(),
		},
	}

	if err := app.Run(ctx, args); err != nil {
		logging.Default().Error("failed to run app", "error", err)
		return err
	}

	return nil
}
