package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
)

// NoAuthnUseCase provides a mock authentication that always returns an anonymous user
type NoAuthnUseCase struct {
	repo interfaces.Repository
}

// NewNoAuthnUseCase creates a new NoAuthnUseCase instance
func NewNoAuthnUseCase(repo interfaces.Repository) *NoAuthnUseCase {
	return &NoAuthnUseCase{
		repo: repo,
	}
}

// GetAuthURL returns a dummy URL (should not be called in no-auth mode)
func (uc *NoAuthnUseCase) GetAuthURL(state string) string {
	return "/"
}

// HandleCallback handles OAuth callback (should not be called in no-auth mode)
func (uc *NoAuthnUseCase) HandleCallback(ctx context.Context, code string) (*auth.Token, error) {
	// In no-auth mode, this should not be called, but we create an anonymous token anyway
	token := auth.NewAnonymousUser()
	if err := uc.repo.PutToken(ctx, token); err != nil {
		return nil, goerr.Wrap(err, "failed to store anonymous token")
	}
	return token, nil
}

// ValidateToken always returns an anonymous user token
func (uc *NoAuthnUseCase) ValidateToken(ctx context.Context, tokenID auth.TokenID, tokenSecret auth.TokenSecret) (*auth.Token, error) {
	// Always return anonymous user
	return auth.NewAnonymousUser(), nil
}

// Logout does nothing in no-auth mode
func (uc *NoAuthnUseCase) Logout(ctx context.Context, tokenID auth.TokenID) error {
	// No-op in no-auth mode
	return nil
}

// IsNoAuthn returns true for NoAuthnUseCase
func (uc *NoAuthnUseCase) IsNoAuthn() bool {
	return true
}
