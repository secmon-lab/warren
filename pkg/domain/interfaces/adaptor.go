package interfaces

import (
	"context"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/opaq"
	"github.com/slack-go/slack"
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

type SlackClient interface {
	PostMessageContext(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error)
	UpdateMessageContext(ctx context.Context, channelID, timestamp string, options ...slack.MsgOption) (string, string, string, error)
	AuthTest() (*slack.AuthTestResponse, error)
	OpenView(triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error)
	UploadFileV2Context(ctx context.Context, params slack.UploadFileV2Parameters) (*slack.FileSummary, error)
}
