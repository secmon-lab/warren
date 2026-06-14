package intune

import (
	"context"
	"log/slog"

	"github.com/gollem-dev/gollem"
	extintune "github.com/gollem-dev/tools/intune"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/urfave/cli/v3"
)

// Action is the warren-side wrapper around github.com/gollem-dev/tools/intune.
// It implements interfaces.Tool, binding CLI flags and warren-specific planner
// metadata onto the external gollem.ToolSet that carries the Specs/Run logic.
type Action struct {
	tenantID     string
	clientID     string
	clientSecret string
	baseURL      string

	opts  []extintune.Option
	inner gollem.ToolSet
}

var _ interfaces.Tool = &Action{}

func (x *Action) ID() string {
	return "intune"
}

func (x *Action) Description() string {
	return "Microsoft Intune device compliance and management lookup"
}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "intune-tenant-id",
			Usage:       "Azure AD tenant ID",
			Destination: &x.tenantID,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_INTUNE_TENANT_ID"),
		},
		&cli.StringFlag{
			Name:        "intune-client-id",
			Usage:       "Azure AD application (client) ID",
			Destination: &x.clientID,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_INTUNE_CLIENT_ID"),
		},
		&cli.StringFlag{
			Name:        "intune-client-secret",
			Usage:       "Azure AD client secret",
			Destination: &x.clientSecret,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_INTUNE_CLIENT_SECRET"),
		},
		&cli.StringFlag{
			Name:        "intune-base-url",
			Usage:       "Microsoft Graph API base URL",
			Destination: &x.baseURL,
			Category:    "Tool",
			Value:       "https://graph.microsoft.com/v1.0",
			Sources:     cli.EnvVars("WARREN_INTUNE_BASE_URL"),
			Action: func(_ context.Context, _ *cli.Command, v string) error {
				x.opts = append(x.opts, extintune.WithBaseURL(v))
				return nil
			},
		},
	}
}

func (x *Action) Configure(_ context.Context) error {
	if x.tenantID == "" || x.clientID == "" || x.clientSecret == "" {
		return errutil.ErrActionUnavailable
	}

	ts, err := extintune.New(x.tenantID, x.clientID, x.clientSecret, x.opts...)
	if err != nil {
		return goerr.Wrap(err, "failed to configure Intune tool")
	}
	x.inner = ts
	return nil
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("tenant_id", x.tenantID),
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
		return nil, goerr.New("Intune tool is not configured")
	}
	return x.inner.Specs(ctx)
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.inner == nil {
		return nil, goerr.New("Intune tool is not configured")
	}
	return x.inner.Run(ctx, name, args)
}
