package whois

import (
	"context"
	"log/slog"

	"github.com/gollem-dev/gollem"
	extwhois "github.com/gollem-dev/tools/whois"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/urfave/cli/v3"
)

// Action is the warren-side wrapper around github.com/gollem-dev/tools/whois.
// It implements interfaces.Tool, binding warren-specific planner metadata onto
// the external gollem.ToolSet that carries the Specs/Run logic.
type Action struct {
	inner gollem.ToolSet
}

var _ interfaces.Tool = &Action{}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) ID() string {
	return "whois"
}

func (x *Action) Description() string {
	return "WHOIS domain and IP registration lookups"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{}
}

func (x *Action) Configure(_ context.Context) error {
	ts, err := extwhois.New()
	if err != nil {
		return goerr.Wrap(err, "failed to configure whois tool")
	}
	x.inner = ts
	return nil
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	if x.inner == nil {
		return nil, goerr.New("whois tool is not configured")
	}
	return x.inner.Specs(ctx)
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.inner == nil {
		return nil, goerr.New("whois tool is not configured")
	}
	return x.inner.Run(ctx, name, args)
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue()
}

func (x *Action) Prompt(_ context.Context) (string, error) {
	return "", nil
}
