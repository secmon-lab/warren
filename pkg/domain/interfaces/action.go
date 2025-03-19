package interfaces

import (
	"context"
	"log/slog"

	"github.com/secmon-lab/warren/pkg/domain/model"
	"github.com/urfave/cli/v3"
)

type Action interface {
	Spec() model.ActionSpec
	Execute(ctx context.Context, slack SlackThreadService, ssn LLMSession, args model.Arguments) (*model.ActionResult, error)
	Flags() []cli.Flag
	Configure(ctx context.Context) error
	LogValue() slog.Value
}
