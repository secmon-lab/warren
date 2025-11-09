package memory

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
)

// Token related methods
func (r *Memory) PutToken(ctx context.Context, token *auth.Token) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tokens[token.ID] = token
	return nil
}

func (r *Memory) GetToken(ctx context.Context, tokenID auth.TokenID) (*auth.Token, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	token, ok := r.tokens[tokenID]
	if !ok {
		return nil, goerr.New("token not found", goerr.V("token_id", tokenID))
	}
	return token, nil
}

func (r *Memory) DeleteToken(ctx context.Context, tokenID auth.TokenID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.tokens, tokenID)
	return nil
}
