package abort

import (
	"context"
	"fmt"

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

	// Get all sessions for this ticket
	sessions, err := clients.Repo().GetSessionsByTicket(ctx, ticket.ID)
	if err != nil {
		return goerr.Wrap(err, "failed to get sessions by ticket")
	}

	// Find and abort all running sessions
	var abortedCount int
	for _, session := range sessions {
		if session.Status == types.SessionStatusRunning {
			session.UpdateStatus(ctx, types.SessionStatusAborted)
			if err := clients.Repo().PutSession(ctx, session); err != nil {
				return goerr.Wrap(err, "failed to update session status")
			}
			abortedCount++
		}
	}

	// Notify user
	if abortedCount == 0 {
		if err := clients.Thread().PostComment(ctx, "ℹ️ No running session found for this ticket."); err != nil {
			return goerr.Wrap(err, "failed to post info message")
		}
	} else {
		var message string
		if abortedCount == 1 {
			message = "🛑 Session aborted. The agent will stop at the next checkpoint."
		} else {
			message = fmt.Sprintf("🛑 %d sessions aborted. The agents will stop at the next checkpoint.", abortedCount)
		}
		if err := clients.Thread().PostComment(ctx, message); err != nil {
			return goerr.Wrap(err, "failed to post confirmation message")
		}
	}

	return nil
}
