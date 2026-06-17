package falcon

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

// TokenProviderForTest wraps tokenProvider for testing purposes.
type TokenProviderForTest struct {
	tp *tokenProvider
}

// NewTokenProviderForTest creates a tokenProvider wrapper for testing.
func NewTokenProviderForTest(clientID, clientSecret, baseURL string) *TokenProviderForTest {
	return &TokenProviderForTest{
		tp: newTokenProvider(clientID, clientSecret, baseURL),
	}
}

// GetToken returns a valid bearer token.
func (t *TokenProviderForTest) GetToken(ctx context.Context) (string, error) {
	return t.tp.getToken(ctx)
}

// ClearToken invalidates the cached token.
func (t *TokenProviderForTest) ClearToken() {
	t.tp.clearToken()
}

// InternalToolForTest wraps internalTool for testing purposes.
type InternalToolForTest struct {
	tool *internalTool
}

// NewInternalToolForTest creates an internalTool wrapper for testing without
// storage (event search returns the first page directly).
func NewInternalToolForTest(clientID, clientSecret, baseURL string) *InternalToolForTest {
	tp := newTokenProvider(clientID, clientSecret, baseURL)
	return &InternalToolForTest{
		tool: newInternalTool(tp, baseURL, nil, ""),
	}
}

// NewInternalToolForTestWithStorage creates an internalTool wrapper backed by
// the given storage client, enabling result-set snapshotting and pagination.
func NewInternalToolForTestWithStorage(clientID, clientSecret, baseURL string, storage interfaces.StorageClient, prefix string) *InternalToolForTest {
	tp := newTokenProvider(clientID, clientSecret, baseURL)
	return &InternalToolForTest{
		tool: newInternalTool(tp, baseURL, storage, prefix),
	}
}

// Run executes a tool by name with the given arguments.
func (t *InternalToolForTest) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	return t.tool.Run(ctx, name, args)
}

// SpecCount returns the number of tool specs.
func (t *InternalToolForTest) SpecCount(ctx context.Context) (int, error) {
	specs, err := t.tool.Specs(ctx)
	if err != nil {
		return 0, err
	}
	return len(specs), nil
}

// ParseLimit exposes parseLimit for testing.
func ParseLimit(args map[string]any) int { return parseLimit(args) }

// ParseOffset exposes parseOffset for testing.
func ParseOffset(args map[string]any) int { return parseOffset(args) }

// Paginate exposes paginate for testing.
func Paginate(events []any, offset, limit int) ([]any, int, bool) {
	return paginate(events, offset, limit)
}
