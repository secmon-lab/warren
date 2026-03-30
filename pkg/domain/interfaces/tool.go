package interfaces

import (
	"context"
	"log/slog"

	"github.com/urfave/cli/v3"
)

// Tool extends ToolSet with CLI-level concerns (flags, configuration, helpers).
// All tools in pkg/tool/* implement this interface.
type Tool interface {
	ToolSet
	Flags() []cli.Flag
	Configure(ctx context.Context) error
	LogValue() slog.Value
	Helper() *cli.Command
}
