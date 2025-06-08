package interfaces

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/gollem"
	"github.com/urfave/cli/v3"
)

type Tool interface {
	Name() string
	Flags() []cli.Flag
	Configure(ctx context.Context) error
	LogValue() slog.Value
	Helper() *cli.Command
	Prompt(ctx context.Context) (string, error)
	gollem.ToolSet
}
