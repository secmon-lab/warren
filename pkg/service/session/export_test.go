package session

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func (x *Service) GetHistory(ctx context.Context, sessionID types.SessionID) (*session.History, error) {
	return x.getHistory(ctx, sessionID)
}

func (x *Service) PutHistory(ctx context.Context, history *session.History) error {
	return x.putHistory(ctx, history)
}
