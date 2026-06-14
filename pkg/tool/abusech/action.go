package abusech

import (
	"context"
	"log/slog"

	"github.com/gollem-dev/gollem"
	extabusech "github.com/gollem-dev/tools/abusech"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/urfave/cli/v3"
)

// Action is the warren-side wrapper around github.com/gollem-dev/tools/abusech.
// It implements interfaces.Tool, binding CLI flags and warren-specific planner
// metadata onto the external gollem.ToolSet that carries the Specs/Run logic.
type Action struct {
	apiKey  string
	baseURL string

	opts  []extabusech.Option
	inner gollem.ToolSet
}

var _ interfaces.Tool = &Action{}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) ID() string {
	return "abusech"
}

func (x *Action) Description() string {
	return "MalwareBazaar malware hash lookup via abuse.ch"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "abusech-api-key",
			Usage:       "abuse.ch API key",
			Destination: &x.apiKey,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_ABUSECH_AUTH_KEY"),
		},
		&cli.StringFlag{
			Name:        "abusech-base-url",
			Usage:       "abuse.ch API base URL",
			Destination: &x.baseURL,
			Category:    "Tool",
			Value:       "https://mb-api.abuse.ch/api/v1",
			Sources:     cli.EnvVars("WARREN_ABUSECH_BASE_URL"),
			Action: func(_ context.Context, _ *cli.Command, v string) error {
				x.opts = append(x.opts, extabusech.WithBaseURL(v))
				return nil
			},
		},
	}
}

func (x *Action) Configure(_ context.Context) error {
	if x.apiKey == "" {
		return errutil.ErrActionUnavailable
	}

	ts, err := extabusech.New(x.apiKey, x.opts...)
	if err != nil {
		return goerr.Wrap(err, "failed to configure abusech tool")
	}
	x.inner = ts
	return nil
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	if x.inner == nil {
		return nil, goerr.New("abusech tool is not configured")
	}
	return x.inner.Specs(ctx)
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.inner == nil {
		return nil, goerr.New("abusech tool is not configured")
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
