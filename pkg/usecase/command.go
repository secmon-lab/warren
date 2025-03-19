package usecase

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

type Command struct {
	Command string
	Args    []string
}

func (uc *UseCases) HandleCommand(ctx context.Context, thread interfaces.SlackThreadService, args []string) error {
	logger := logging.From(ctx)
	logger.Info("slack command", "thread", thread, "args", args)

	if len(args) == 0 {
		if thread != nil {
			thread.Reply(ctx, "Please specify the command.")
		}
		return nil
	}

	return nil
}
