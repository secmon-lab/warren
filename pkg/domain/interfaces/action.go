package interfaces

import (
	"context"
	"log/slog"

	"github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/urfave/cli/v3"
)

type Action interface {
	Spec() action.ActionSpec
	Execute(ctx context.Context, ssn LLMClient, args action.Arguments) (*action.ActionResult, error)
	Flags() []cli.Flag
	Configure(ctx context.Context) error
	LogValue() slog.Value
}
