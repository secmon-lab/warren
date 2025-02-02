package config

import (
	"log/slog"

	"github.com/getsentry/sentry-go"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

type Sentry struct {
	dsn string
	env string
}

func (x *Sentry) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "sentry-dsn",
			Usage:       "Sentry DSN",
			Category:    "Sentry",
			Sources:     cli.EnvVars("WARREN_SENTRY_DSN"),
			Destination: &x.dsn,
		},
		&cli.StringFlag{
			Name:        "sentry-env",
			Usage:       "Sentry environment",
			Category:    "Sentry",
			Sources:     cli.EnvVars("WARREN_SENTRY_ENV"),
			Destination: &x.env,
		},
	}
}

func (x Sentry) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("dsn", x.dsn),
		slog.String("env", x.env),
	)
}

func (x *Sentry) Configure() error {
	if x.dsn == "" {
		logging.Default().Warn("Sentry is not configured")
		return nil
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         x.dsn,
		Environment: x.env,
	}); err != nil {
		return goerr.Wrap(err, "failed to initialize sentry")
	}
	return nil
}
