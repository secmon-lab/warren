package session

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func toHistoryObjectName(sessionID types.SessionID, historyID types.HistoryID) string {
	return fmt.Sprintf("session/%s/history/%s.json", sessionID, historyID)
}

func (x *Service) putHistory(ctx context.Context, history *session.History) error {
	if err := x.clients.Repository().PutHistory(ctx, history.SessionID, history); err != nil {
		return err
	}

	objectName := toHistoryObjectName(history.SessionID, history.ID)

	w := x.clients.Storage().PutObject(ctx, objectName)

	if err := json.NewEncoder(w).Encode(history); err != nil {
		return goerr.Wrap(err, "failed to encode history")
	}

	if err := w.Close(); err != nil {
		return goerr.Wrap(err, "failed to close writer")
	}

	return nil
}

func (x *Service) getHistory(ctx context.Context, sessionID types.SessionID) (*session.History, error) {
	historyRecord, err := x.clients.Repository().GetLatestHistory(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if historyRecord == nil {
		return nil, nil
	}

	objectName := toHistoryObjectName(historyRecord.SessionID, historyRecord.ID)

	reader, err := x.clients.Storage().GetObject(ctx, objectName)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var history session.History

	if err := json.NewDecoder(reader).Decode(&history); err != nil {
		return nil, goerr.Wrap(err, "failed to decode history")
	}

	return &history, nil
}
