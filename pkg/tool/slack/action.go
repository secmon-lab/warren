package slack

import (
	"context"
	"log/slog"

	"github.com/gollem-dev/gollem"
	extslack "github.com/gollem-dev/tools/slack"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/urfave/cli/v3"
)

// Action is the warren-side wrapper around github.com/gollem-dev/tools/slack.
// It binds the CLI flag and warren planner metadata onto the external
// gollem.ToolSet that carries the search.messages Specs/Run logic.
type Action struct {
	oauthToken string

	opts  []extslack.Option
	inner gollem.ToolSet
}

var _ interfaces.Tool = &Action{}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) ID() string {
	return "slack_message_search"
}

func (x *Action) Description() string {
	return "Slack message search"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "slack-tool-user-token",
			Usage:       "Slack User token for search API access (must be User token, not Bot token)",
			Destination: &x.oauthToken,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_SLACK_TOOL_USER_TOKEN"),
		},
	}
}

func (x *Action) Configure(_ context.Context) error {
	if x.oauthToken == "" {
		return errutil.ErrActionUnavailable
	}

	ts, err := extslack.New(x.oauthToken, x.opts...)
	if err != nil {
		return goerr.Wrap(err, "failed to configure Slack tool")
	}
	x.inner = ts
	return nil
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	if x.inner == nil {
		return nil, goerr.New("Slack tool is not configured")
	}
	return x.inner.Specs(ctx)
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.inner == nil {
		return nil, goerr.New("Slack tool is not configured")
	}
	return x.inner.Run(ctx, name, args)
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("oauth_token.len", len(x.oauthToken)),
	)
}

// Prompt returns additional instructions for the system prompt.
func (x *Action) Prompt(_ context.Context) (string, error) {
	return "", nil
}
