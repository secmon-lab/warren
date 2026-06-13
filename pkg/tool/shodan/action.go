package shodan

import (
	"context"
	"log/slog"

	"github.com/gollem-dev/gollem"
	extshodan "github.com/gollem-dev/tools/shodan"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/urfave/cli/v3"
)

// Action is the warren-side wrapper around github.com/gollem-dev/tools/shodan.
// It implements interfaces.Tool, binding CLI flags and warren-specific planner
// metadata onto the external gollem.ToolSet that carries the Specs/Run logic.
type Action struct {
	apiKey  string
	baseURL string

	inner gollem.ToolSet
}

var _ interfaces.Tool = &Action{}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) ID() string {
	return "shodan"
}

func (x *Action) Description() string {
	return "Shodan internet-facing asset and service search"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "shodan-api-key",
			Usage:       "Shodan API key",
			Destination: &x.apiKey,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_SHODAN_API_KEY"),
		},
		&cli.StringFlag{
			Name:        "shodan-base-url",
			Usage:       "Shodan API base URL",
			Destination: &x.baseURL,
			Category:    "Tool",
			Value:       "https://api.shodan.io",
			Sources:     cli.EnvVars("WARREN_SHODAN_BASE_URL"),
		},
	}
}

func (x *Action) Configure(_ context.Context) error {
	if x.apiKey == "" {
		return errutil.ErrActionUnavailable
	}

	var opts []extshodan.Option
	if x.baseURL != "" {
		opts = append(opts, extshodan.WithBaseURL(x.baseURL))
	}

	ts, err := extshodan.New(x.apiKey, opts...)
	if err != nil {
		return goerr.Wrap(err, "failed to configure Shodan tool")
	}
	x.inner = ts
	return nil
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	if x.inner == nil {
		return nil, goerr.New("Shodan tool is not configured")
	}
	return x.inner.Specs(ctx)
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.inner == nil {
		return nil, goerr.New("Shodan tool is not configured")
	}
	return x.inner.Run(ctx, name, args)
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("api_key.len", len(x.apiKey)),
		slog.String("base_url", x.baseURL),
	)
}

// Prompt returns additional instructions for the system prompt.
func (x *Action) Prompt(_ context.Context) (string, error) {
	return "", nil
}
