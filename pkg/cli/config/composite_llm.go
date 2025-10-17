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
