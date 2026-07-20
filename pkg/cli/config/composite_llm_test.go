package config_test

import (
	"context"
	"testing"

	"github.com/gollem-dev/gollem"
	"github.com/gollem-dev/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config"
)

// TestCompositeLLMClientRoutes covers the composite's whole job: content
// generation goes to one client and embeddings to the other.
func TestCompositeLLMClientRoutes(t *testing.T) {
	t.Run("NewSession goes to the content client", func(t *testing.T) {
		wantSession := &mock.SessionMock{}
		contentCalled := false
		content := &mock.LLMClientMock{
			NewSessionFunc: func(_ context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
				contentCalled = true
				// Caller options must be forwarded untouched.
				cfg := gollem.NewSessionConfig(options...)
				gt.Equal(t, cfg.SystemPrompt(), "system")
				return wantSession, nil
			},
		}
		embedding := &mock.LLMClientMock{
			NewSessionFunc: func(_ context.Context, _ ...gollem.SessionOption) (gollem.Session, error) {
				t.Error("embedding client must not receive NewSession")
				return nil, nil
			},
		}

		client := config.NewCompositeLLMClient(content, embedding)
		got, err := client.NewSession(context.Background(), gollem.WithSessionSystemPrompt("system"))
		gt.NoError(t, err)
		gt.True(t, contentCalled)
		gt.Equal(t, got, gollem.Session(wantSession))
	})

	t.Run("GenerateEmbedding goes to the embedding client", func(t *testing.T) {
		content := &mock.LLMClientMock{
			GenerateEmbeddingFunc: func(_ context.Context, _ int, _ []string) ([][]float64, error) {
				t.Error("content client must not receive GenerateEmbedding")
				return nil, nil
			},
		}
		var gotDimension int
		var gotInput []string
		embedding := &mock.LLMClientMock{
			GenerateEmbeddingFunc: func(_ context.Context, dimension int, input []string) ([][]float64, error) {
				gotDimension = dimension
				gotInput = input
				return [][]float64{{0.1, 0.2}}, nil
			},
		}

		client := config.NewCompositeLLMClient(content, embedding)
		result, err := client.GenerateEmbedding(context.Background(), 2, []string{"hello"})
		gt.NoError(t, err)
		gt.Equal(t, result, [][]float64{{0.1, 0.2}})
		gt.Equal(t, gotDimension, 2)
		gt.Equal(t, gotInput, []string{"hello"})
	})
}
