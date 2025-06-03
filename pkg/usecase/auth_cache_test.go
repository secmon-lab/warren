package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthCache_Basic(t *testing.T) {
	cache := usecase.NewTestAuthCache()

	// Create a test token
	token := auth.NewToken("test-sub", "test@example.com", "Test User")

	// Test Put and Get
	cache.TestPut(token.ID, token)
	retrievedToken, found := cache.TestGet(token.ID)
	assert.True(t, found)
	assert.Equal(t, token, retrievedToken)

	// Test non-existent token
	nonExistentID := auth.NewTokenID()
	_, found = cache.TestGet(nonExistentID)
	assert.False(t, found)

	// Test Remove
	cache.TestRemove(token.ID)
	_, found = cache.TestGet(token.ID)
	assert.False(t, found)
}

func TestAuthCache_Expiration(t *testing.T) {
	cache := usecase.NewTestAuthCache()
	token := auth.NewToken("test-sub", "test@example.com", "Test User")

	// Create a cached token with past timestamp to simulate expiration
	cachedToken := usecase.NewTestCachedToken(token, time.Now().Add(-usecase.AuthCacheExpireDuration-time.Minute))

	cache.TestSetCachedToken(token.ID, cachedToken)

	// Should not return expired token
	_, found := cache.TestGet(token.ID)
	assert.False(t, found)

	// And should be cleaned up from cache
	exists := cache.TestHasCachedToken(token.ID)
	assert.False(t, exists)
}

func TestAuthCache_Clear(t *testing.T) {
	cache := usecase.NewTestAuthCache()

	// Add multiple tokens
	token1 := auth.NewToken("sub1", "test1@example.com", "User 1")
	token2 := auth.NewToken("sub2", "test2@example.com", "User 2")

	cache.TestPut(token1.ID, token1)
	cache.TestPut(token2.ID, token2)

	// Verify tokens are there
	_, found1 := cache.TestGet(token1.ID)
	_, found2 := cache.TestGet(token2.ID)
	assert.True(t, found1)
	assert.True(t, found2)

	// Clear cache
	cache.TestClear()

	// Verify tokens are gone
	_, found1 = cache.TestGet(token1.ID)
	_, found2 = cache.TestGet(token2.ID)
	assert.False(t, found1)
	assert.False(t, found2)
}

func TestAuthUseCase_ValidateTokenWithCache(t *testing.T) {
	repo := repository.NewMemory()
	slackSvc := mockSlackService(t)
	authUC := usecase.NewAuthUseCase(repo, slackSvc, "client-id", "client-secret", "callback-url")
	ctx := context.Background()

	// Create and store a token
	token := auth.NewToken("test-sub", "test@example.com", "Test User")
	err := repo.PutToken(ctx, token)
	require.NoError(t, err)

	// First validation should fetch from repository and cache
	validatedToken, err := authUC.ValidateToken(ctx, token.ID, token.Secret)
	require.NoError(t, err)
	assert.Equal(t, token, validatedToken)

	// Second validation should use cache
	validatedToken2, err := authUC.ValidateToken(ctx, token.ID, token.Secret)
	require.NoError(t, err)
	assert.Equal(t, token, validatedToken2)

	// Verify token is in cache
	cache := authUC.TestGetCache()
	cachedToken, found := cache.TestGet(token.ID)
	assert.True(t, found)
	assert.Equal(t, token, cachedToken)
}

func TestAuthUseCase_ValidateTokenWithCache_InvalidSecret(t *testing.T) {
	repo := repository.NewMemory()
	slackSvc := mockSlackService(t)
	authUC := usecase.NewAuthUseCase(repo, slackSvc, "client-id", "client-secret", "callback-url")
	ctx := context.Background()

	// Create and store a token
	token := auth.NewToken("test-sub", "test@example.com", "Test User")
	err := repo.PutToken(ctx, token)
	require.NoError(t, err)

	// First validation with correct secret to cache the token
	_, err = authUC.ValidateToken(ctx, token.ID, token.Secret)
	require.NoError(t, err)

	// Try with invalid secret - should fail and remove from cache
	_, err = authUC.ValidateToken(ctx, token.ID, auth.TokenSecret("invalid-secret"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token secret")

	// Verify token is removed from cache
	cache := authUC.TestGetCache()
	_, found := cache.TestGet(token.ID)
	assert.False(t, found)
}

func TestAuthUseCase_Logout_RemovesFromCache(t *testing.T) {
	repo := repository.NewMemory()
	slackSvc := mockSlackService(t)
	authUC := usecase.NewAuthUseCase(repo, slackSvc, "client-id", "client-secret", "callback-url")
	ctx := context.Background()

	// Create and store a token
	token := auth.NewToken("test-sub", "test@example.com", "Test User")
	err := repo.PutToken(ctx, token)
	require.NoError(t, err)

	// Validate to cache the token
	_, err = authUC.ValidateToken(ctx, token.ID, token.Secret)
	require.NoError(t, err)

	// Verify token is in cache
	cache := authUC.TestGetCache()
	_, found := cache.TestGet(token.ID)
	assert.True(t, found)

	// Logout
	err = authUC.Logout(ctx, token.ID)
	require.NoError(t, err)

	// Verify token is removed from cache
	_, found = cache.TestGet(token.ID)
	assert.False(t, found)
}
