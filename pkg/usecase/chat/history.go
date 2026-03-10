package chat

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// LoadHistory loads the chat history for a ticket from storage.
// It first tries to load from latest.json, falling back to history/{id}.json.
func LoadHistory(ctx context.Context, repo interfaces.Repository, ticketID types.TicketID, storageSvc *storage.Service) (*gollem.History, error) {
	logger := logging.From(ctx)

	// Try latest.json first
	if history, err := storageSvc.GetLatestHistory(ctx, ticketID); err != nil {
		logger.Warn("failed to load latest history, falling back to history record", "error", err)
	} else if history != nil {
		if history.Version > 0 && history.ToCount() > 0 {
			logger.Debug("loaded history from latest.json", "version", history.Version, "message_count", history.ToCount())
			return history, nil
		}
		logger.Warn("latest history incompatible, falling back to history record",
			"version", history.Version, "message_count", history.ToCount())
	}

	// Fallback to history/{id}.json
	historyRecord, err := repo.GetLatestHistory(ctx, ticketID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get latest history")
	}

	if historyRecord == nil {
		return nil, nil
	}

	history, err := storageSvc.GetHistory(ctx, ticketID, historyRecord.ID)
	if err != nil {
		msg.Notify(ctx, "⚠️ Failed to load chat history, starting fresh: %s", err.Error())
		logger.Warn("failed to get history data, starting with new history", "error", err)
		return nil, nil
	}

	if history != nil && (history.Version <= 0 || history.ToCount() <= 0) {
		msg.Notify(ctx, "⚠️ Chat history incompatible (version=%d, messages=%d), starting fresh", history.Version, history.ToCount())
		logger.Warn("history incompatible, starting with new history",
			"version", history.Version,
			"message_count", history.ToCount(),
			"history_id", historyRecord.ID)
		return nil, nil
	}

	return history, nil
}

// SaveHistory saves a gollem History to storage and records it in the repository.
func SaveHistory(ctx context.Context, repo interfaces.Repository, storageClient interfaces.StorageClient, storageSvc *storage.Service, ticketID types.TicketID, history *gollem.History) error {
	logger := logging.From(ctx)

	if history == nil {
		return goerr.New("history is nil after execution")
	}

	logger.Debug("saving chat history",
		"history_version", history.Version,
		"message_count", history.ToCount())

	if history.ToCount() <= 0 {
		logger.Warn("history has no messages, but saving anyway to maintain consistency",
			"version", history.Version,
			"message_count", history.ToCount(),
			"ticket_id", ticketID)
	}

	if history.Version > 0 && storageClient != nil {
		newRecord := ticket.NewHistory(ctx, ticketID)

		if err := storageSvc.PutHistory(ctx, ticketID, newRecord.ID, history); err != nil {
			msg.Notify(ctx, "💥 Failed to save chat history: %s", err.Error())
			return goerr.Wrap(err, "failed to put history")
		}

		if err := repo.PutHistory(ctx, ticketID, &newRecord); err != nil {
			logger := logging.From(ctx)
			if data, jsonErr := json.Marshal(&newRecord); jsonErr == nil {
				logger.Error("failed to save history", "error", err, "history", string(data))
			}
			msg.Notify(ctx, "💥 Failed to save chat record: %s", err.Error())
			return goerr.Wrap(err, "failed to put history", goerr.V("history", &newRecord))
		}

		logger.Debug("history saved", "history_id", newRecord.ID, "ticket_id", ticketID)
	}

	return nil
}
