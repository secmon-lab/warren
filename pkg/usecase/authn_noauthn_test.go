package usecase_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase"
)

func TestNoAuthnUseCase(t *testing.T) {
	ctx := context.Background()

	t.Run("GetAuthURL returns root path", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.NewNoAuthnUseCase(repo)

		url := uc.GetAuthURL("test-state")
		gt.Equal(t, url, "/")
	})

	t.Run("HandleCallback returns anonymous user", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.NewNoAuthnUseCase(repo)

		token, err := uc.HandleCallback(ctx, "test-code")
		gt.NoError(t, err)
		gt.NotNil(t, token)
		gt.True(t, token.IsAnonymous())
		gt.Equal(t, token.Sub, auth.AnonymousUserID)
		gt.Equal(t, token.Name, auth.AnonymousUserName)
		gt.Equal(t, token.Email, auth.AnonymousUserEmail)

		// Verify token was stored
		storedToken, err := repo.GetToken(ctx, token.ID)
		gt.NoError(t, err)
		gt.Equal(t, storedToken.Sub, auth.AnonymousUserID)
	})

	t.Run("ValidateToken always returns anonymous user", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.NewNoAuthnUseCase(repo)

		// Even with invalid token ID and secret
		token, err := uc.ValidateToken(ctx, auth.TokenID("invalid"), auth.TokenSecret("invalid"))
		gt.NoError(t, err)
		gt.NotNil(t, token)
		gt.True(t, token.IsAnonymous())
		gt.Equal(t, token.Sub, auth.AnonymousUserID)
	})

	t.Run("Logout does nothing", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.NewNoAuthnUseCase(repo)

		// Store a token first
		token := auth.NewToken("U12345", "test@example.com", "Test User")
		err := repo.PutToken(ctx, token)
		gt.NoError(t, err)

		// Logout should not error
		err = uc.Logout(ctx, token.ID)
		gt.NoError(t, err)

		// Token should still exist (no-op)
		storedToken, err := repo.GetToken(ctx, token.ID)
		gt.NoError(t, err)
		gt.NotNil(t, storedToken)
	})
}
