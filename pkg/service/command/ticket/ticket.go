package ticket

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/service/command/core"
)

//go:embed ticket.help.md
var ticketHelp string

func Create(ctx context.Context, clients *core.Clients, msg *slack.Message, input string) (any, error) {
	// Show help if requested
	if input == "help" {
		clients.Thread().Reply(ctx, ticketHelp)
		return nil, nil
	}

	// Check if UseCase is available
	ticketUC := clients.TicketUseCase()
	if ticketUC == nil {
		clients.Thread().Reply(ctx, "Ticket creation from conversation is not available")
		return nil, goerr.New("ticket use case not available")
	}

	// Create ticket from conversation using UseCase
	// Use entire input as user context
	ticket, err := ticketUC.CreateTicketFromConversation(
		ctx,
		msg.Thread(),
		msg.User(),
		input,
	)
	if err != nil {
		clients.Thread().Reply(ctx, fmt.Sprintf("Failed to create ticket: %v", err))
		return nil, goerr.Wrap(err, "failed to create ticket from conversation")
	}

	// Success message is not needed as ticket posting is done by UseCase
	_ = ticket
	return nil, nil
}
