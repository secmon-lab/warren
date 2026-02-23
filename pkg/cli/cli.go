package cli

import (
	"context"

	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func Run(ctx context.Context, args []string) error {
	var loggerCfg config.Logger
	var language lang.Lang
	var closer func()
	app := &cli.Command{
		Name:  "warren",
		Usage: "warren",
		Flags: append(loggerCfg.Flags(),
			&cli.StringFlag{
				Name:        "lang",
				Usage:       "Language for text output (e.g., English, Japanese, Spanish, French, German, etc.)",
				Value:       "English",
				Sources:     cli.EnvVars("WARREN_LANG"),
				Destination: (*string)(&language),
			},
		),
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			f, err := loggerCfg.Configure()
			if err != nil {
				return ctx, err
			}
			closer = f

			logging.Default().Info("base options", "language", language, "logger", loggerCfg)

			if err := language.Validate(); err != nil {
				return ctx, err
			}

			return lang.With(ctx, language), nil
		},
		After: func(ctx context.Context, c *cli.Command) error {
			if closer != nil {
				closer()
			}
			return nil
		},
		Commands: []*cli.Command{
			cmdServe(),
			cmdTest(),
			cmdChat(),
			cmdTool(),
			cmdAlert(),
			cmdMigrate(),
			cmdRefine(),
		},
	}

	if err := app.Run(ctx, args); err != nil {
		logging.Default().Error("failed to run app", "error", err)
		return err
	}

	return nil
}
