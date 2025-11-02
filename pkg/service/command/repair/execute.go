package repair

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/service/command/core"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

func Run(ctx context.Context, clients *core.Clients, slackMsg *slack.Message, input string) (any, error) {
	args := strings.Fields(input)
	if len(args) < 1 {
		return nil, fmt.Errorf("usage: repair embeddings <alerts|tickets|all>")
	}

	if args[0] != "embeddings" {
		return nil, fmt.Errorf("unknown repair command: %s", args[0])
	}

	target := "all"
	if len(args) > 1 {
		target = args[1]
	}

	switch target {
	case "alerts":
		if err := repairAlertEmbeddings(ctx, clients); err != nil {
			return nil, err
		}
	case "tickets":
		if err := repairTicketEmbeddings(ctx, clients); err != nil {
			return nil, err
		}
	case "all":
		if err := repairAlertEmbeddings(ctx, clients); err != nil {
			return nil, err
		}
		if err := repairTicketEmbeddings(ctx, clients); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown target: %s. Use 'alerts', 'tickets', or 'all'", target)
	}

	return nil, nil
}

func repairAlertEmbeddings(ctx context.Context, clients *core.Clients) error {
	msg.Notify(ctx, "🔧 Starting alert embedding repair...")

	// Get alerts with invalid embeddings (null, empty, or zero vector)
	alerts, err := clients.Repo().GetAlertsWithInvalidEmbedding(ctx)
	if err != nil {
		return err
	}

	if len(alerts) == 0 {
		msg.Trace(ctx, "✅ No alerts with invalid embeddings found")
		return nil
	}

	msg.Notify(ctx, "📊 Found %d alerts with invalid embeddings", len(alerts))

	successCount := 0
	failCount := 0
	ts := clock.Now(ctx)
	notifyPeriod := time.Minute * 2

	for i, alert := range alerts {
		if clock.Since(ctx, ts) > notifyPeriod {
			msg.Trace(ctx, "⌛ Repairing alert embeddings %d/%d (Success: %d, Failed: %d)",
				i+1, len(alerts), successCount, failCount)
			ts = clock.Now(ctx)
		}

		// No need to check again since GetAlertsWithInvalidEmbedding already filtered

		// Fill metadata (which includes embedding generation)
		if err := alert.FillMetadata(ctx, clients.LLM()); err != nil {
			msg.Notify(ctx, "⚠️ Failed to repair alert %s: %v", alert.ID, err)
			failCount++
			continue
		}

		// Update the alert
		if err := clients.Repo().PutAlert(ctx, *alert); err != nil {
			msg.Notify(ctx, "⚠️ Failed to update alert %s: %v", alert.ID, err)
			failCount++
			continue
		}

		successCount++
	}

	msg.Trace(ctx, "✅ Alert embedding repair completed - Success: %d, Failed: %d",
		successCount, failCount)
	return nil
}

func repairTicketEmbeddings(ctx context.Context, clients *core.Clients) error {
	msg.Notify(ctx, "🔧 Starting ticket embedding repair...")

	// Get tickets with invalid embeddings
	tickets, err := clients.Repo().GetTicketsWithInvalidEmbedding(ctx)
	if err != nil {
		return err
	}

	if len(tickets) == 0 {
		msg.Trace(ctx, "✅ No tickets with invalid embeddings found")
		return nil
	}

	msg.Notify(ctx, "📊 Found %d tickets with invalid embeddings", len(tickets))

	successCount := 0
	failCount := 0
	ts := clock.Now(ctx)
	notifyPeriod := time.Minute * 2
	processed := 0

	for _, ticket := range tickets {
		// No need to check again since GetTicketsWithInvalidEmbedding already filtered
		processed++
		if clock.Since(ctx, ts) > notifyPeriod {
			msg.Trace(ctx, "⌛ Repairing ticket embeddings %d/%d (Success: %d, Failed: %d)",
				processed, len(tickets), successCount, failCount)
			ts = clock.Now(ctx)
		}

		// Generate embedding for ticket
		if err := ticket.FillMetadata(ctx, clients.LLM(), clients.Repo()); err != nil {
			msg.Notify(ctx, "⚠️ Failed to repair ticket %s: %v", ticket.ID, err)
			failCount++
			continue
		}

		// Update the ticket
		if err := clients.Repo().PutTicket(ctx, *ticket); err != nil {
			msg.Notify(ctx, "⚠️ Failed to update ticket %s: %v", ticket.ID, err)
			failCount++
			continue
		}

		successCount++
	}

	msg.Trace(ctx, "✅ Ticket embedding repair completed - Success: %d, Failed: %d",
		successCount, failCount)
	return nil
}
