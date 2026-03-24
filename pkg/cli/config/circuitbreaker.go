package config

import (
	"time"

	"github.com/secmon-lab/warren/pkg/service/circuitbreaker"
	"github.com/urfave/cli/v3"
)

// CircuitBreaker represents configuration for alert circuit breaker
type CircuitBreaker struct {
	Enabled bool
	Window  time.Duration
	Limit   int
}

// Flags returns CLI flags for circuit breaker configuration
func (cfg *CircuitBreaker) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "circuit-breaker-enabled",
			Sources:     cli.EnvVars("WARREN_CIRCUIT_BREAKER_ENABLED"),
			Usage:       "Enable alert circuit breaker (rate limiting)",
			Category:    "Circuit Breaker",
			Value:       false,
			Destination: &cfg.Enabled,
		},
		&cli.DurationFlag{
			Name:        "circuit-breaker-window",
			Sources:     cli.EnvVars("WARREN_CIRCUIT_BREAKER_WINDOW"),
			Usage:       "Sliding window duration for rate limiting",
			Category:    "Circuit Breaker",
			Value:       1 * time.Hour,
			Destination: &cfg.Window,
		},
		&cli.IntFlag{
			Name:        "circuit-breaker-limit",
			Sources:     cli.EnvVars("WARREN_CIRCUIT_BREAKER_LIMIT"),
			Usage:       "Maximum number of alerts per window",
			Category:    "Circuit Breaker",
			Value:       60,
			Destination: &cfg.Limit,
		},
	}
}

// ToConfig converts to circuitbreaker.Config
func (cfg *CircuitBreaker) ToConfig() circuitbreaker.Config {
	return circuitbreaker.Config{
		Enabled: cfg.Enabled,
		Window:  cfg.Window,
		Limit:   cfg.Limit,
	}
}
