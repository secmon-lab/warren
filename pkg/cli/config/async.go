package config

import (
	"github.com/m-mizutani/goerr/v2"
	"github.com/urfave/cli/v3"
)

// AsyncAlertHook represents configuration for asynchronous alert hooks
type AsyncAlertHook struct {
	Raw    bool
	PubSub bool
	SNS    bool
}

// Flags returns CLI flags for async alert hook configuration
func (cfg *AsyncAlertHook) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:    "async-alert-hook",
			Sources: cli.EnvVars("WARREN_ASYNC_ALERT_HOOK"),
			Usage:   "Enable async processing for alert hooks (raw, pubsub, sns, all)",
		},
	}
}

// Parse parses CLI context and sets configuration values
func (cfg *AsyncAlertHook) Parse(ctx *cli.Command) error {
	values := ctx.StringSlice("async-alert-hook")
	for _, v := range values {
		switch v {
		case "raw":
			cfg.Raw = true
		case "pubsub":
			cfg.PubSub = true
		case "sns":
			cfg.SNS = true
		case "all":
			cfg.Raw = true
			cfg.PubSub = true
			cfg.SNS = true
		default:
			return goerr.New("invalid async-alert-hook value", goerr.V("value", v))
		}
	}
	return nil
}
