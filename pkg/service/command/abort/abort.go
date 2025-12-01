package abort

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/command/core"
)

// Execute aborts the currently running session for the ticket
func Execute(ctx context.Context, clients *core.Clients, msg *slack.Message, input string) error {
	// Get ticket by thread
	ticket, err := clients.Repo().GetTicketByThread(ctx, msg.Thread())
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket by thread")
	}

	if ticket == nil {
		if err := clients.Thread().PostComment(ctx, "😣 No ticket found for this thread."); err != nil {
			return goerr.Wrap(err, "failed to post error message")
		}
		return nil
	}

	// Get running session for this ticket
	session, err := clients.Repo().GetSessionByTicket(ctx, ticket.ID)
	if err != nil {
		return goerr.Wrap(err, "failed to get session by ticket")
	}

	if session == nil {
		if err := clients.Thread().PostComment(ctx, "ℹ️ No running session found for this ticket."); err != nil {
			return goerr.Wrap(err, "failed to post info message")
		}
		return nil
	}

	// Update session status to aborted
	session.UpdateStatus(ctx, types.SessionStatusAborted)
	if err := clients.Repo().PutSession(ctx, session); err != nil {
		return goerr.Wrap(err, "failed to update session status")
	}

	// Notify user
	if err := clients.Thread().PostComment(ctx, "🛑 Session aborted. The agent will stop at the next checkpoint."); err != nil {
		return goerr.Wrap(err, "failed to post confirmation message")
	}

	return nil
}
