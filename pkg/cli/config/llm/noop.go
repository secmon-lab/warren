package llm

import (
	"context"

	"github.com/m-mizutani/gollem"
)

// noopLLMClient is a no-op LLM client used by the "noop" provider. It exists
// so that test environments (e2e, local development without GCP credentials)
// can boot warren via a regular TOML config without special-casing in CLI code.
//
// The LLM is REQUIRED in production; "noop" is only for non-LLM integration
// paths such as Playwright UI tests.
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

func (n *noopSession) History() (*gollem.History, error) { return nil, nil }

func (n *noopSession) AppendHistory(_ *gollem.History) error { return nil }

func (n *noopSession) CountToken(_ context.Context, _ ...gollem.Input) (int, error) { return 0, nil }
