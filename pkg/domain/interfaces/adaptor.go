package interfaces

import (
	"context"
	"io"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/model/gemini"
	"github.com/slack-go/slack"
)

type LLMClient interface {
	StartChat(options ...gemini.Option) LLMSession
	NewQuery(options ...gemini.Option) LLMQuery
	LLMInquiry
}

type LLMSession interface {
	LLMInquiry
	GetHistory() []*genai.Content
}

type LLMQuery func(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error)

type LLMInquiry interface {
	SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error)
}

type EmbeddingClient interface {
	Embeddings(ctx context.Context, texts []string, dimensionality int) ([][]float32, error)
}

type PolicyClient interface {
	Query(context.Context, string, any, any, ...opaq.QueryOption) error
	Sources() map[string]string
}

type StorageClient interface {
	PutObject(ctx context.Context, bucket, object string, r io.Reader) error
	GetObject(ctx context.Context, bucket, object string) (io.ReadCloser, error)
	Close(ctx context.Context)
}

type SlackClient interface {
	PostMessageContext(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error)
	UpdateMessageContext(ctx context.Context, channelID, timestamp string, options ...slack.MsgOption) (string, string, string, error)
	AuthTest() (*slack.AuthTestResponse, error)
	OpenView(triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error)
	UploadFileV2Context(ctx context.Context, params slack.UploadFileV2Parameters) (*slack.FileSummary, error)
}

type SlackThreadService interface {
	Reply(ctx context.Context, message string)
	NewStateFunc(ctx context.Context, message string) func(ctx context.Context, msg string)
}
