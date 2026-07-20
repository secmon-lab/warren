// Package llmclient provides decorators that apply session-level policy to any
// gollem.LLMClient, independent of which provider built it.
//
// Policy lives here rather than on a concrete client type (or on a routing type
// such as the CLI's composite client) because Warren constructs LLM clients in
// more than one place: the main LLM configuration and the webfetch tool each
// build their own. Tying a policy to one construction site silently skips the
// others, so the policy is expressed as a wrapper applied wherever a client is
// born.
package llmclient

import (
	"context"

	"github.com/gollem-dev/gollem"
)

// WithPromptCache returns a client that enables provider prompt caching on every
// session it creates. When enabled is false the client is returned unchanged, so
// callers never pay for a wrapper that does nothing.
//
// Only Claude acts on the option: it marks the stable prefix (system prompt and
// tool definitions) and the conversation tail with ephemeral cache_control so
// repeated prefixes are served from Claude's prompt cache. Gemini and OpenAI
// cache automatically and ignore it, so wrapping a non-Claude client is harmless
// but pointless.
func WithPromptCache(client gollem.LLMClient, enabled bool) gollem.LLMClient {
	if !enabled || client == nil {
		return client
	}
	return &promptCacheClient{inner: client}
}

type promptCacheClient struct {
	inner gollem.LLMClient
}

// NewSession enables prompt caching for the session. The option is prepended so
// that a caller passing an explicit gollem.WithSessionPromptCache still wins.
//
// Note this only reaches callers that go through gollem.LLMClient. A gollem
// agent configured with gollem.WithPromptCache(false) cannot switch it back off,
// because the agent forwards a session option only when its own setting is true.
// Use the construction-time toggle to disable caching instead.
func (c *promptCacheClient) NewSession(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
	opts := append([]gollem.SessionOption{gollem.WithSessionPromptCache(true)}, options...)
	return c.inner.NewSession(ctx, opts...)
}

func (c *promptCacheClient) GenerateEmbedding(ctx context.Context, dimension int, input []string) ([][]float64, error) {
	return c.inner.GenerateEmbedding(ctx, dimension, input)
}
