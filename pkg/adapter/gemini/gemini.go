package gemini

import (
	"context"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

type GeminiClient struct {
	model *genai.GenerativeModel
}

var _ interfaces.LLMClient = &GeminiClient{}

type clientOption struct {
	location         string
	model            string
	responseMIMEType string
}

type Option func(*clientOption)

func WithLocation(location string) Option {
	return func(o *clientOption) {
		o.location = location
	}
}

func WithModel(model string) Option {
	return func(o *clientOption) {
		o.model = model
	}
}

func WithResponseMIMEType(mimeType string) Option {
	return func(o *clientOption) {
		o.responseMIMEType = mimeType
	}
}

func New(ctx context.Context, project string, opts ...Option) (*GeminiClient, error) {
	opt := &clientOption{
		location:         "us-central1",
		model:            "gemini-2.0-flash-exp",
		responseMIMEType: "application/json",
	}
	for _, o := range opts {
		o(opt)
	}
	client, err := genai.NewClient(ctx, project, opt.location)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create vertex ai client",
			goerr.V("project", project),
			goerr.V("location", opt.location),
		)
	}

	model := client.GenerativeModel(opt.model)
	model.GenerationConfig.ResponseMIMEType = opt.responseMIMEType

	return &GeminiClient{model: model}, nil
}

func (x *GeminiClient) StartChat() interfaces.LLMSession {
	return &GeminiSession{session: x.model.StartChat()}
}

func (x *GeminiClient) SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error) {
	return x.model.GenerateContent(ctx, msg...)
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
