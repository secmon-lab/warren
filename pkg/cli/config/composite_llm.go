package config

import (
	"context"

	"github.com/m-mizutani/gollem"
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
	return c.contentClient.NewSession(ctx, options...)
}

func (c *CompositeLLMClient) GenerateEmbedding(ctx context.Context, dimension int, input []string) ([][]float64, error) {
	return c.embeddingClient.GenerateEmbedding(ctx, dimension, input)
}

func (c *CompositeLLMClient) CountTokens(ctx context.Context, history *gollem.History) (int, error) {
	return c.contentClient.CountTokens(ctx, history)
}

func (c *CompositeLLMClient) IsCompatibleHistory(ctx context.Context, history *gollem.History) error {
	return c.contentClient.IsCompatibleHistory(ctx, history)
}
