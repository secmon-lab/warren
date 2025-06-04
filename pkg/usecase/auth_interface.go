package usecase

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/auth"
)

// AuthUseCaseInterface defines the interface for authentication use cases
type AuthUseCaseInterface interface {
	GetAuthURL(state string) string
	HandleCallback(ctx context.Context, code string) (*auth.Token, error)
	ValidateToken(ctx context.Context, tokenID auth.TokenID, tokenSecret auth.TokenSecret) (*auth.Token, error)
	Logout(ctx context.Context, tokenID auth.TokenID) error
}
