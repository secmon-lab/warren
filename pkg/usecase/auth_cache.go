package usecase

import (
	"context"
	"crypto/subtle"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
)

// authCacheExpireDuration defines the cache expiration duration (10 minutes)
const authCacheExpireDuration = 10 * time.Minute

// cachedToken represents a cached token with its cache timestamp
type cachedToken struct {
	Token    *auth.Token
	CachedAt time.Time
}

// IsExpired checks if the cached entry has expired
func (c *cachedToken) IsExpired() bool {
	return time.Now().After(c.CachedAt.Add(authCacheExpireDuration))
}

// authCache provides in-memory caching for authentication tokens
type authCache struct {
	mu    sync.RWMutex
	cache map[auth.TokenID]*cachedToken
}

// newAuthCache creates a new authentication cache
func newAuthCache() *authCache {
	return &authCache{
		cache: make(map[auth.TokenID]*cachedToken),
	}
}

// get retrieves a token from cache if it exists and is not expired
func (ac *authCache) get(tokenID auth.TokenID) (*auth.Token, bool) {
	ac.mu.RLock()
	cachedToken, exists := ac.cache[tokenID]
	ac.mu.RUnlock()

	if !exists {
		return nil, false
	}

	// Check if cache entry has expired
	if cachedToken.IsExpired() {
		// Remove expired entry with write lock
		ac.mu.Lock()
		delete(ac.cache, tokenID)
		ac.mu.Unlock()
		return nil, false
	}

	return cachedToken.Token, true
}

// put stores a token in the cache
func (ac *authCache) put(tokenID auth.TokenID, token *auth.Token) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.cache[tokenID] = &cachedToken{
		Token:    token,
		CachedAt: time.Now(),
	}
}

// remove removes a token from the cache
func (ac *authCache) remove(tokenID auth.TokenID) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	delete(ac.cache, tokenID)
}

// clear removes all tokens from the cache
func (ac *authCache) clear() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.cache = make(map[auth.TokenID]*cachedToken)
}

// validateTokenWithCache validates the token using cache when possible
func (uc *AuthUseCase) validateTokenWithCache(ctx context.Context, tokenID auth.TokenID, tokenSecret auth.TokenSecret) (*auth.Token, error) {
	// Try to get token from cache first
	if cachedToken, found := uc.cache.get(tokenID); found {
		// Even if cached, still validate the secret and expiration
		if subtle.ConstantTimeCompare([]byte(cachedToken.Secret), []byte(tokenSecret)) != 1 {
			// Invalid secret - remove from cache
			uc.cache.remove(tokenID)
			return nil, goerr.New("invalid token secret")
		}

		if cachedToken.IsExpired() {
			// Token expired - remove from cache
			uc.cache.remove(tokenID)
			return nil, goerr.New("token expired")
		}

		return cachedToken, nil
	}

	// Not in cache, get from repository
	token, err := uc.repo.GetToken(ctx, tokenID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get token")
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(token.Secret), []byte(tokenSecret)) != 1 {
		return nil, goerr.New("invalid token secret")
	}

	if token.IsExpired() {
		return nil, goerr.New("token expired")
	}

	// Store in cache for future use
	uc.cache.put(tokenID, token)

	return token, nil
}
