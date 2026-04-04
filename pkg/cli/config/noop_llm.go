package config

import (
	"context"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

// noopLLMClient is a no-op LLM client for E2E testing and development without LLM access.
type noopLLMClient struct{}

func (n *noopLLMClient) NewSession(_ context.Context, _ ...gollem.SessionOption) (gollem.Session, error) {
	return &noopSession{}, nil
}

func (n *noopLLMClient) GenerateEmbedding(_ context.Context, dimension int, input []string) ([][]float64, error) {
	result := make([][]float64, len(input))
	for i := range input {
		result[i] = make([]float64, dimension)
	}
	return result, nil
}

type noopSession struct{}

func (n *noopSession) Generate(_ context.Context, _ []gollem.Input, _ ...gollem.GenerateOption) (*gollem.Response, error) {
	return &gollem.Response{Texts: []string{"LLM is disabled"}}, nil
}

func (n *noopSession) Stream(_ context.Context, _ []gollem.Input, _ ...gollem.GenerateOption) (<-chan *gollem.Response, error) {
	ch := make(chan *gollem.Response, 1)
	ch <- &gollem.Response{Texts: []string{"LLM is disabled"}}
	close(ch)
	return ch, nil
}

func (n *noopSession) GenerateContent(_ context.Context, _ ...gollem.Input) (*gollem.Response, error) {
	return &gollem.Response{Texts: []string{"LLM is disabled"}}, nil
}

func (n *noopSession) GenerateStream(_ context.Context, _ ...gollem.Input) (<-chan *gollem.Response, error) {
	ch := make(chan *gollem.Response, 1)
	ch <- &gollem.Response{Texts: []string{"LLM is disabled"}}
	close(ch)
	return ch, nil
}

func (n *noopSession) History() (*gollem.History, error) {
	return nil, nil
}

func (n *noopSession) AppendHistory(_ *gollem.History) error {
	return nil
}

func (n *noopSession) CountToken(_ context.Context, _ ...gollem.Input) (int, error) {
	return 0, nil
}

// noopEmbeddingClient is a no-op embedding client that returns zero vectors.
type noopEmbeddingClient struct{}

func (n *noopEmbeddingClient) Embeddings(_ context.Context, texts []string, dimensionality int) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, dimensionality)
	}
	return result, nil
}

// NewNoopLLMClient returns a no-op LLM client.
func NewNoopLLMClient() gollem.LLMClient {
	return &noopLLMClient{}
}

// NewNoopEmbeddingClient returns a no-op embedding client.
func NewNoopEmbeddingClient() interfaces.EmbeddingClient {
	return &noopEmbeddingClient{}
}
