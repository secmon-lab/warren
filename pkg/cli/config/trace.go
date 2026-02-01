package config

import (
	"context"
	"log/slog"

	traceAdapter "github.com/secmon-lab/warren/pkg/adapter/trace"

	"github.com/urfave/cli/v3"
)

// Trace holds CLI configuration for trace data storage.
type Trace struct {
	bucket string
	prefix string
}

// Flags returns CLI flags for trace configuration.
func (x *Trace) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "trace-bucket",
			Usage:       "GCS bucket for trace data storage",
			Category:    "Trace",
			Destination: &x.bucket,
			Sources:     cli.EnvVars("WARREN_TRACE_BUCKET"),
		},
		&cli.StringFlag{
			Name:        "trace-prefix",
			Usage:       "Object prefix for trace data",
			Category:    "Trace",
			Destination: &x.prefix,
			Value:       "traces/",
			Sources:     cli.EnvVars("WARREN_TRACE_PREFIX"),
		},
	}
}

// LogValue returns structured log representation.
func (x *Trace) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("bucket", x.bucket),
		slog.String("prefix", x.prefix),
	)
}

// Configure creates a GCS Trace Repository if bucket is configured.
// Returns nil if trace-bucket is not set (tracing disabled).
func (x *Trace) Configure(ctx context.Context) (*traceAdapter.Repository, error) {
	if x.bucket == "" {
		return nil, nil
	}

	repo, err := traceAdapter.New(ctx, x.bucket)
	if err != nil {
		return nil, err
	}

	return repo.WithPrefix(x.prefix), nil
}

// IsConfigured returns true if trace storage is configured.
func (x *Trace) IsConfigured() bool {
	return x.bucket != ""
}
