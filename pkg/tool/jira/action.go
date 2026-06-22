package jira

import (
	"context"
	"log/slog"

	"github.com/gollem-dev/gollem"
	extjira "github.com/gollem-dev/tools/jira"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/urfave/cli/v3"
)

// Action is the warren-side wrapper around github.com/gollem-dev/tools/jira.
// It implements interfaces.Tool, binding CLI flags and warren-specific planner
// metadata onto the external gollem.ToolSet that carries the read-only Jira
// Cloud query logic (list projects, search issues via JQL, fetch issues).
type Action struct {
	baseURL  string
	email    string
	apiToken string

	inner gollem.ToolSet
}

var _ interfaces.Tool = &Action{}

func (x *Action) ID() string {
	return "jira"
}

func (x *Action) Description() string {
	return "Jira Cloud read-only query (list projects, search issues via JQL, fetch issue content)"
}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "jira-base-url",
			Usage:       "Jira Cloud site URL (e.g. https://your-domain.atlassian.net)",
			Destination: &x.baseURL,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_JIRA_BASE_URL"),
		},
		&cli.StringFlag{
			Name:        "jira-user-email",
			Usage:       "Jira account email for Basic authentication",
			Destination: &x.email,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_JIRA_USER_EMAIL"),
		},
		&cli.StringFlag{
			Name:        "jira-api-token",
			Usage:       "Jira Cloud API token (paired with the account email)",
			Destination: &x.apiToken,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_JIRA_API_TOKEN"),
		},
	}
}

func (x *Action) Configure(_ context.Context) error {
	if x.baseURL == "" || x.email == "" || x.apiToken == "" {
		return errutil.ErrActionUnavailable
	}

	ts, err := extjira.New(x.baseURL, x.email, x.apiToken)
	if err != nil {
		return goerr.Wrap(err, "failed to configure Jira tool")
	}
	x.inner = ts
	return nil
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("base_url", x.baseURL),
		slog.String("email", x.email),
		slog.Int("api_token.len", len(x.apiToken)),
	)
}

func (x *Action) Prompt(_ context.Context) (string, error) {
	return "", nil
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	if x.inner == nil {
		return nil, goerr.New("Jira tool is not configured")
	}
	return x.inner.Specs(ctx)
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.inner == nil {
		return nil, goerr.New("Jira tool is not configured")
	}
	return x.inner.Run(ctx, name, args)
}
