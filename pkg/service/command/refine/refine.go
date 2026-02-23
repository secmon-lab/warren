package refine

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/service/command/core"
)

// Run executes the refine command to review open tickets and consolidate unbound alerts.
func Run(ctx context.Context, clients *core.Clients, msg *slack.Message, input string) error {
	refineUC := clients.RefineUseCase()
	if refineUC == nil {
		return goerr.New("refine use case not configured")
	}

	clients.Thread().Reply(ctx, "🔄 Starting refine process...")

	if err := refineUC.Refine(ctx); err != nil {
		return goerr.Wrap(err, "failed to run refine")
	}

	clients.Thread().Reply(ctx, "✅ Refine process completed")
	return nil
}
