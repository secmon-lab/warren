package config

import (
	"context"

	"github.com/gollem-dev/gollem"
)

type CompositeLLMClient struct {
	contentClient   gollem.LLMClient
	embeddingClient gollem.LLMClient
}

func NewCompositeLLMClient(contentClient, embeddingClient gollem.LLMClient) *CompositeLLMClient {
	return &CompositeLLMClient{
		contentClient:   contentClient,
		embeddingClient: embeddingClient,
	}
}

func (c *CompositeLLMClient) NewSession(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
	// Enable provider prompt caching for every session created through the content
	// client. This is the single choke point every session flows through — both
	// direct NewSession calls and the sessions gollem.New agents build internally —
	// so injecting it here turns caching on everywhere without touching each call
	// site. Only Claude acts on it (ephemeral cache_control on the stable prefix);
	// Gemini caches automatically and ignores the option. Prepended so an explicit
	// caller-supplied option would still win.
	opts := append([]gollem.SessionOption{gollem.WithSessionPromptCache(true)}, options...)
	return c.contentClient.NewSession(ctx, opts...)
}

func (c *CompositeLLMClient) GenerateEmbedding(ctx context.Context, dimension int, input []string) ([][]float64, error) {
	return c.embeddingClient.GenerateEmbedding(ctx, dimension, input)
}
