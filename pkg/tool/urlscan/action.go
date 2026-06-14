package urlscan

import (
	"context"
	"log/slog"
	"time"

	"github.com/gollem-dev/gollem"
	exturlscan "github.com/gollem-dev/tools/urlscan"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/urfave/cli/v3"
)

// Action is the warren-side wrapper around github.com/gollem-dev/tools/urlscan.
// It implements interfaces.Tool, binding CLI flags and warren-specific planner
// metadata onto the external gollem.ToolSet that carries the Specs/Run logic.
type Action struct {
	apiKey  string
	baseURL string
	backoff time.Duration
	timeout time.Duration

	opts  []exturlscan.Option
	inner gollem.ToolSet
}

var _ interfaces.Tool = &Action{}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) ID() string {
	return "urlscan"
}

func (x *Action) Description() string {
	return "URL scanning and analysis via urlscan.io"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "urlscan-api-key",
			Usage:       "URLScan API key",
			Destination: &x.apiKey,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_URLSCAN_API_KEY"),
		},
		&cli.StringFlag{
			Name:        "urlscan-base-url",
			Usage:       "URLScan API base URL",
			Destination: &x.baseURL,
			Category:    "Tool",
			Value:       "https://urlscan.io/api/v1",
			Sources:     cli.EnvVars("WARREN_URLSCAN_BASE_URL"),
			Action: func(_ context.Context, _ *cli.Command, v string) error {
				x.opts = append(x.opts, exturlscan.WithBaseURL(v))
				return nil
			},
		},
		&cli.DurationFlag{
			Name:        "urlscan-backoff",
			Usage:       "URLScan API backoff duration",
			Destination: &x.backoff,
			Category:    "Tool",
			Value:       time.Duration(3) * time.Second,
			Sources:     cli.EnvVars("WARREN_URLSCAN_BACKOFF"),
			Action: func(_ context.Context, _ *cli.Command, v time.Duration) error {
				x.opts = append(x.opts, exturlscan.WithBackoff(v))
				return nil
			},
		},
		&cli.DurationFlag{
			Name:        "urlscan-timeout",
			Usage:       "URLScan API timeout duration",
			Destination: &x.timeout,
			Category:    "Tool",
			Value:       time.Duration(30) * time.Second,
			Action: func(_ context.Context, _ *cli.Command, v time.Duration) error {
				x.opts = append(x.opts, exturlscan.WithTimeout(v))
				return nil
			},
		},
	}
}

func (x *Action) Configure(_ context.Context) error {
	if x.apiKey == "" {
		return errutil.ErrActionUnavailable
	}

	ts, err := exturlscan.New(x.apiKey, x.opts...)
	if err != nil {
		return goerr.Wrap(err, "failed to configure urlscan tool")
	}
	x.inner = ts
	return nil
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	if x.inner == nil {
		return nil, goerr.New("urlscan tool is not configured")
	}
	return x.inner.Specs(ctx)
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.inner == nil {
		return nil, goerr.New("urlscan tool is not configured")
	}
	return x.inner.Run(ctx, name, args)
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("api_key.len", len(x.apiKey)),
		slog.String("base_url", x.baseURL),
		slog.Duration("backoff", x.backoff),
	)
}

// Prompt returns additional instructions for the system prompt.
func (x *Action) Prompt(_ context.Context) (string, error) {
	return "", nil
}
