package chat

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

// LoadSessionHistory loads the gollem.History for a Session. Returns
// (nil, nil) when the Session has no saved history yet so callers can
// start with an empty gollem session.
//
// chat-session-redesign Phase 4: keyed on SessionID so Slack
// (long-lived) and Web/CLI (per-invocation) sessions keep independent
// working memory. The pre-redesign ticket-scoped `LoadHistory` /
// `SaveHistory` functions were removed together with the
// `Repository.GetLatestHistory` / `PutHistory` surface they relied on.
func LoadSessionHistory(ctx context.Context, sessionID types.SessionID, storageSvc *storage.Service) (*gollem.History, error) {
	history, err := storageSvc.GetSessionHistory(ctx, sessionID)
	if err != nil {
		logging.From(ctx).Warn("failed to load session history; starting fresh",
			"error", err, "session_id", sessionID)
		return nil, nil
	}
	return history, nil
}

// SaveSessionHistory writes history into the Session's rolling storage
// slot. Errors are logged but not returned so a storage hiccup cannot
// interrupt the chat response path.
func SaveSessionHistory(ctx context.Context, sessionID types.SessionID, storageSvc *storage.Service, history *gollem.History) error {
	if history == nil {
		return goerr.New("history is nil after execution")
	}
	if err := storageSvc.PutSessionHistory(ctx, sessionID, history); err != nil {
		return goerr.Wrap(err, "failed to put session history",
			goerr.V("session_id", sessionID))
	}
	return nil
}
