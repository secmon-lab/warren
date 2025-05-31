package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase"
)

func TestAuthUseCase_GetAuthURL(t *testing.T) {
	repo := repository.NewMemory()
	authUC := usecase.NewAuthUseCase(repo, "test-client-id", "test-client-secret", "http://localhost:3000/api/auth/callback")

	authURL := authUC.GetAuthURL("test-state")

	gt.S(t, authURL).Contains("slack.com/openid/connect/authorize")
	gt.S(t, authURL).Contains("client_id=test-client-id")
	gt.S(t, authURL).Contains("redirect_uri=http%3A%2F%2Flocalhost%3A3000%2Fapi%2Fauth%2Fcallback")
	gt.S(t, authURL).Contains("response_type=code")
	gt.S(t, authURL).Contains("state=test-state")
	gt.S(t, authURL).Contains("scope=openid%2Cemail%2Cprofile")
}

func TestAuthUseCase_ValidateToken(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	authUC := usecase.NewAuthUseCase(repo, "test-client-id", "test-client-secret", "http://localhost:3000/api/auth/callback")

	// Create and store a test token
	token := auth.NewToken("test-sub", "test@example.com", "Test User")
	gt.NoError(t, repo.PutToken(ctx, token))

	// Test valid token
	validatedToken, err := authUC.ValidateToken(ctx, token.ID, token.Secret)
	gt.NoError(t, err)
	gt.Value(t, validatedToken.ID).Equal(token.ID)
	gt.Value(t, validatedToken.Sub).Equal("test-sub")
	gt.Value(t, validatedToken.Email).Equal("test@example.com")
	gt.Value(t, validatedToken.Name).Equal("Test User")

	// Test invalid token secret
	_, err = authUC.ValidateToken(ctx, token.ID, auth.TokenSecret("invalid-secret"))
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "invalid token secret"))

	// Test non-existent token
	nonExistentID := auth.NewTokenID()
	_, err = authUC.ValidateToken(ctx, nonExistentID, token.Secret)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "token not found"))
}

func TestAuthUseCase_Logout(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	authUC := usecase.NewAuthUseCase(repo, "test-client-id", "test-client-secret", "http://localhost:3000/api/auth/callback")

	// Create and store a test token
	token := auth.NewToken("test-sub", "test@example.com", "Test User")
	gt.NoError(t, repo.PutToken(ctx, token))

	// Verify token exists
	_, err := repo.GetToken(ctx, token.ID)
	gt.NoError(t, err)

	// Logout (delete token)
	gt.NoError(t, authUC.Logout(ctx, token.ID))

	// Verify token is deleted
	_, err = repo.GetToken(ctx, token.ID)
	gt.Error(t, err)
}

func TestAuthUseCase_TokenExpiration(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	authUC := usecase.NewAuthUseCase(repo, "test-client-id", "test-client-secret", "http://localhost:3000/api/auth/callback")

	// Create an expired token
	token := auth.NewToken("test-sub", "test@example.com", "Test User")
	// Manually set expiration to past
	token.ExpiresAt = token.CreatedAt.Add(-1 * 60 * 60) // 1 hour ago
	gt.NoError(t, repo.PutToken(ctx, token))

	// Test that expired token is rejected
	_, err := authUC.ValidateToken(ctx, token.ID, token.Secret)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "token expired"))
}
