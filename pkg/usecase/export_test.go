package usecase

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
)

// Export private types and functions for testing

type TestAuthCache = authCache
type TestCachedToken = cachedToken

const AuthCacheExpireDuration = authCacheExpireDuration

// NewTestAuthCache creates a new authentication cache for testing
func NewTestAuthCache() *TestAuthCache {
	return newAuthCache()
}

// NewTestCachedToken creates a cached token for testing
func NewTestCachedToken(token *auth.Token, cachedAt time.Time) *TestCachedToken {
	return &cachedToken{
		Token:    token,
		CachedAt: cachedAt,
	}
}

// TestAuthCache_Get exposes the private get method for testing
func (ac *TestAuthCache) TestGet(tokenID auth.TokenID) (*auth.Token, bool) {
	return ac.get(tokenID)
}

// TestAuthCache_Put exposes the private put method for testing
func (ac *TestAuthCache) TestPut(tokenID auth.TokenID, token *auth.Token) {
	ac.put(tokenID, token)
}

// TestAuthCache_Remove exposes the private remove method for testing
func (ac *TestAuthCache) TestRemove(tokenID auth.TokenID) {
	ac.remove(tokenID)
}

// TestAuthCache_Clear exposes the private clear method for testing
func (ac *TestAuthCache) TestClear() {
	ac.clear()
}

// TestAuthCache_SetCachedToken allows direct manipulation of cache for testing
func (ac *TestAuthCache) TestSetCachedToken(tokenID auth.TokenID, cachedToken *TestCachedToken) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.cache[tokenID] = cachedToken
}

// TestAuthCache_HasCachedToken checks if a token exists in cache for testing
func (ac *TestAuthCache) TestHasCachedToken(tokenID auth.TokenID) bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	_, exists := ac.cache[tokenID]
	return exists
}

// TestAuthUseCase_GetCache exposes the private cache field for testing
func (uc *AuthUseCase) TestGetCache() *TestAuthCache {
	return uc.cache
}

var ToolCallToText = toolCallToText

// GenerateInitialTicketComment exports the private generateInitialTicketComment method for testing
func (uc *UseCases) GenerateInitialTicketCommentForTest(ctx context.Context, ticketData *ticket.Ticket, alerts alert.Alerts) (string, error) {
	return uc.generateInitialTicketComment(ctx, ticketData, alerts)
}

// GenAI related exports for testing

// ProcessGenAI exports the private processGenAI method for testing
func (uc *UseCases) ProcessGenAI(ctx context.Context, alert *alert.Alert) (any, error) {
	return uc.processGenAI(ctx, alert)
}

// HandleNotice exports the private handleNotice method for testing
func (uc *UseCases) HandleNotice(ctx context.Context, alert *alert.Alert, channels []string) error {
	return uc.handleNotice(ctx, alert, channels)
}

// EvaluateAction exports the private evaluateAction function for testing
func EvaluateAction(ctx context.Context, policyClient interfaces.PolicyClient, alert *alert.Alert, llmResponse any) (*action.PolicyResult, error) {
	return evaluateAction(ctx, policyClient, alert, llmResponse)
}
