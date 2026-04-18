package storage

import (
	"encoding/json"
	"fmt"

	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/safe"
)

// Session-scoped gollem.History storage for the chat-session-redesign.
//
// The legacy PutHistory / GetHistory / PutLatestHistory / GetLatestHistory
// are keyed by TicketID and will be retired in Phase 7. These new methods
// are keyed by SessionID so each Session's AI working memory is
// self-contained (a Slack Session spanning many Turns keeps a single
// rolling history; a fresh Web/CLI Session starts empty).

func pathToSessionHistory(prefix string, sessionID types.SessionID) string {
	return fmt.Sprintf("%s%s/sessions/%s/history.json", prefix, StorageSchemaVersion, sessionID)
}

// PutSessionHistory saves the gollem.History for a Session. The object is
// overwritten on each call (we only keep the most recent state; older
// turns' state is reconstructed from Messages if needed).
func (s *Service) PutSessionHistory(ctx context.Context, sessionID types.SessionID, history *gollem.History) error {
	if s.storageClient == nil {
		return nil
	}
	path := pathToSessionHistory(s.prefix, sessionID)

	w := s.storageClient.PutObject(ctx, path)
	if err := json.NewEncoder(w).Encode(history); err != nil {
		return goerr.Wrap(err, "failed to save session history",
			goerr.V("path", path),
			goerr.V("session_id", sessionID))
	}
	if err := w.Close(); err != nil {
		return goerr.Wrap(err, "failed to close session history writer",
			goerr.V("path", path),
			goerr.V("session_id", sessionID))
	}
	return nil
}

// GetSessionHistory loads the gollem.History for a Session. Returns
// (nil, nil) when no history has been saved yet so callers can start
// fresh without special-casing.
func (s *Service) GetSessionHistory(ctx context.Context, sessionID types.SessionID) (*gollem.History, error) {
	if s.storageClient == nil {
		return nil, nil
	}
	path := pathToSessionHistory(s.prefix, sessionID)

	r, err := s.storageClient.GetObject(ctx, path)
	if err != nil || r == nil {
		// Treat any read error (including "object not found") as an
		// empty history. The AI's working memory is not authoritative
		// data; a missing or transiently unreadable file should result
		// in a fresh start rather than failing the chat request.
		return nil, nil
	}
	defer safe.Close(ctx, r)

	var history gollem.History
	if err := json.NewDecoder(r).Decode(&history); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal session history",
			goerr.V("path", path),
			goerr.V("session_id", sessionID))
	}
	return &history, nil
}
