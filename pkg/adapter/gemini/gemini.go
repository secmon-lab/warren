package gemini

import (
	"context"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/gemini"
)

type GeminiClient struct {
	client           *genai.Client
	location         string
	model            string
	responseMIMEType string
}

var _ interfaces.LLMClient = &GeminiClient{}

type Option func(*GeminiClient)

func WithLocation(location string) Option {
	return func(o *GeminiClient) {
		o.location = location
	}
}

func WithModel(model string) Option {
	return func(o *GeminiClient) {
		o.model = model
	}
}

func WithResponseMIMEType(mimeType string) Option {
	return func(o *GeminiClient) {
		o.responseMIMEType = mimeType
	}
}

func New(ctx context.Context, project string, opts ...Option) (*GeminiClient, error) {
	client := &GeminiClient{
		location:         "us-central1",
		model:            "gemini-2.0-flash-exp",
		responseMIMEType: "application/json",
	}
	for _, o := range opts {
		o(client)
	}
	genaiClient, err := genai.NewClient(ctx, project, client.location)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create vertex ai client",
			goerr.V("project", project),
			goerr.V("location", client.location),
		)
	}

	return &GeminiClient{client: genaiClient}, nil
}

func (x *GeminiClient) StartChat(options ...gemini.Option) interfaces.LLMSession {
	cfg := gemini.NewConfig(options...)

	model := x.client.GenerativeModel(x.model)
	model.GenerationConfig.ResponseMIMEType = x.responseMIMEType
	model.Tools = cfg.Tools()
	model.ToolConfig = cfg.ToolConfig()

	return &GeminiSession{session: model.StartChat()}
}

func (x *GeminiClient) SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error) {
	return x.client.GenerativeModel(x.model).GenerateContent(ctx, msg...)
}

type GeminiSession struct {
	session *genai.ChatSession
}

var _ interfaces.LLMSession = &GeminiSession{}

func (x *GeminiSession) SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error) {
	return x.session.SendMessage(ctx, msg...)
}

func (x *GeminiSession) SetHistory(history ...*genai.Content) {
	x.session.History = history
}

func (x *GeminiSession) GetHistory() []*genai.Content {
	return x.session.History
}
