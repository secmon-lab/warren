package interfaces

import (
	"context"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/opaq"
)

type LLMClient interface {
	StartChat() LLMSession
	SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error)
}

type EmbeddingClient interface {
	Embeddings(ctx context.Context, texts []string, dimensionality int) ([][]float32, error)
}

type LLMSession interface {
	SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error)
}

type PolicyClient interface {
	Query(context.Context, string, any, any, ...opaq.QueryOption) error
	Sources() map[string]string
}
