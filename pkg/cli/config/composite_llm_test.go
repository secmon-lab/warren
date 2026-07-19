package config_test

import (
	"context"
	"testing"

	"github.com/gollem-dev/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config"
)

// recordingLLMClient captures the SessionOptions handed to NewSession and the
// arguments handed to GenerateEmbedding so tests can assert what the composite
// client forwards to each wrapped client.
type recordingLLMClient struct {
	gotOptions []gollem.SessionOption

	embeddingDimension int
	embeddingInput     []string
	embeddingResult    [][]float64
}

func (c *recordingLLMClient) NewSession(_ context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
	c.gotOptions = options
	return nil, nil
}

func (c *recordingLLMClient) GenerateEmbedding(_ context.Context, dimension int, input []string) ([][]float64, error) {
	c.embeddingDimension = dimension
	c.embeddingInput = input
	return c.embeddingResult, nil
}

func TestCompositeLLMClientNewSessionEnablesPromptCache(t *testing.T) {
	content := &recordingLLMClient{}
	embedding := &recordingLLMClient{}
	client := config.NewCompositeLLMClient(content, embedding)

	caller := gollem.WithSessionSystemPrompt("system")
	_, err := client.NewSession(context.Background(), caller)
	gt.NoError(t, err)

	// The content client must receive the caller's options plus the injected
	// prompt-cache option, and PromptCache must resolve to true.
	cfg := gollem.NewSessionConfig(content.gotOptions...)
	gt.True(t, cfg.PromptCache())
	gt.Equal(t, cfg.SystemPrompt(), "system")

	// Embedding client must not be touched by NewSession.
	gt.Equal(t, len(embedding.gotOptions), 0)
}

func TestCompositeLLMClientNewSessionCallerCanOverridePromptCache(t *testing.T) {
	content := &recordingLLMClient{}
	client := config.NewCompositeLLMClient(content, &recordingLLMClient{})

	// A caller that explicitly disables prompt cache must win over the injected
	// default because the injection is prepended.
	_, err := client.NewSession(context.Background(), gollem.WithSessionPromptCache(false))
	gt.NoError(t, err)

	cfg := gollem.NewSessionConfig(content.gotOptions...)
	gt.False(t, cfg.PromptCache())
}

func TestCompositeLLMClientGenerateEmbeddingDelegates(t *testing.T) {
	content := &recordingLLMClient{}
	embedding := &recordingLLMClient{embeddingResult: [][]float64{{0.1, 0.2}}}
	client := config.NewCompositeLLMClient(content, embedding)

	result, err := client.GenerateEmbedding(context.Background(), 2, []string{"hello"})
	gt.NoError(t, err)
	gt.Equal(t, result, [][]float64{{0.1, 0.2}})

	// Embedding goes to the embedding client, never the content client.
	gt.Equal(t, embedding.embeddingDimension, 2)
	gt.Equal(t, embedding.embeddingInput, []string{"hello"})
	gt.Nil(t, content.embeddingInput)
}
