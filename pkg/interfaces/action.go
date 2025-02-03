package interfaces

import (
	"context"
	"log/slog"

	"github.com/secmon-lab/warren/pkg/model"
	"github.com/urfave/cli/v3"
)

type Action interface {
	Spec() model.ActionSpec
	Execute(ctx context.Context, slack SlackService, ssn GenAIChatSession, args model.Arguments) (*model.ActionResult, error)
	Flags() []cli.Flag
	Enabled() bool
	LogValue() slog.Value
}
