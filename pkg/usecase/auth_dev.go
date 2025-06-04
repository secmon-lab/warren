package usecase

import (
	"context"
	"crypto/subtle"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
)

// AuthUseCaseForDev is a development-only authentication use case that bypasses Slack OAuth
type AuthUseCaseForDev struct {
	devUser   string
	devTokens map[auth.TokenID]*auth.Token // In-memory storage for dev tokens
}

// NewAuthUseCaseForDev creates an AuthUseCase for development mode
func NewAuthUseCaseForDev(devUser string) *AuthUseCaseForDev {
	return &AuthUseCaseForDev{
		devUser:   devUser,
		devTokens: make(map[auth.TokenID]*auth.Token),
	}
}

// GetAuthURL returns a dummy URL for dev mode (will be handled by login handler)
func (uc *AuthUseCaseForDev) GetAuthURL(state string) string {
	return "/api/auth/callback?state=" + state + "&code=dev" // Dev mode doesn't need OAuth flow
}

// HandleCallback creates a dev token directly
func (uc *AuthUseCaseForDev) HandleCallback(ctx context.Context, code string) (*auth.Token, error) {
	// Create and store dev token in memory
	token := auth.NewToken(uc.devUser, uc.devUser, "Dev User ("+uc.devUser+")")
	uc.devTokens[token.ID] = token
	return token, nil
}

// ValidateToken validates token from in-memory storage
func (uc *AuthUseCaseForDev) ValidateToken(ctx context.Context, tokenID auth.TokenID, tokenSecret auth.TokenSecret) (*auth.Token, error) {
	token, exists := uc.devTokens[tokenID]
	if !exists {
		return nil, goerr.New("token not found")
	}

	// Validate secret (constant-time comparison)
	if subtle.ConstantTimeCompare([]byte(token.Secret), []byte(tokenSecret)) != 1 {
		return nil, goerr.New("invalid token secret")
	}

	if token.IsExpired() {
		delete(uc.devTokens, tokenID)
		return nil, goerr.New("token expired")
	}

	return token, nil
}

// Logout removes token from in-memory storage
func (uc *AuthUseCaseForDev) Logout(ctx context.Context, tokenID auth.TokenID) error {
	delete(uc.devTokens, tokenID)
	return nil
}

// DevModeAuth creates a dev token (same as HandleCallback for dev mode)
func (uc *AuthUseCaseForDev) DevModeAuth(ctx context.Context) (*auth.Token, error) {
	return uc.HandleCallback(ctx, "dev")
}
