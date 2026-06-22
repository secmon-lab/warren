package falcon

import (
	"context"
	"log/slog"

	"github.com/gollem-dev/gollem"
	extfalcon "github.com/gollem-dev/tools/falcon"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/urfave/cli/v3"
)

const defaultBaseURL = "https://api.crowdstrike.com"

// Action is the warren-side wrapper around github.com/gollem-dev/tools/falcon.
// It implements interfaces.Tool, binding CLI flags and warren-specific planner
// metadata onto the external gollem.ToolSet that carries the read-only
// CrowdStrike Falcon query logic (incidents, alerts, behaviors, devices,
// CrowdScores, and EDR telemetry events).
type Action struct {
	clientID     string
	clientSecret string
	baseURL      string

	opts  []extfalcon.Option
	inner gollem.ToolSet
}

var _ interfaces.Tool = &Action{}

func (x *Action) ID() string {
	return "falcon"
}

func (x *Action) Description() string {
	return "CrowdStrike Falcon read-only query (incidents, alerts, behaviors, devices, CrowdScores, EDR events)"
}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "falcon-client-id",
			Usage:       "CrowdStrike Falcon API client ID",
			Destination: &x.clientID,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_FALCON_CLIENT_ID"),
		},
		&cli.StringFlag{
			Name:        "falcon-client-secret",
			Usage:       "CrowdStrike Falcon API client secret",
			Destination: &x.clientSecret,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_FALCON_CLIENT_SECRET"),
		},
		&cli.StringFlag{
			Name:        "falcon-base-url",
			Usage:       "CrowdStrike Falcon API base URL",
			Destination: &x.baseURL,
			Category:    "Tool",
			Value:       defaultBaseURL,
			Sources:     cli.EnvVars("WARREN_FALCON_BASE_URL"),
			Action: func(_ context.Context, _ *cli.Command, v string) error {
				x.opts = append(x.opts, extfalcon.WithBaseURL(v))
				return nil
			},
		},
	}
}

func (x *Action) Configure(_ context.Context) error {
	if x.clientID == "" || x.clientSecret == "" {
		return errutil.ErrActionUnavailable
	}

	ts, err := extfalcon.New(x.clientID, x.clientSecret, x.opts...)
	if err != nil {
		return goerr.Wrap(err, "failed to configure Falcon tool")
	}
	x.inner = ts
	return nil
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("client_id.len", len(x.clientID)),
		slog.Int("client_secret.len", len(x.clientSecret)),
		slog.String("base_url", x.baseURL),
	)
}

func (x *Action) Prompt(_ context.Context) (string, error) {
	return "", nil
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	if x.inner == nil {
		return nil, goerr.New("Falcon tool is not configured")
	}
	return x.inner.Specs(ctx)
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.inner == nil {
		return nil, goerr.New("Falcon tool is not configured")
	}
	return x.inner.Run(ctx, name, args)
}
