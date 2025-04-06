package gemini

import (
	"context"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/gemini"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

type GeminiClient struct {
	client             *genai.Client
	location           string
	defaultModel       string
	defaultContentType string
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
		o.defaultModel = model
	}
}

func WithContentType(mimeType string) Option {
	return func(o *GeminiClient) {
		o.defaultContentType = mimeType
	}
}

func New(ctx context.Context, project string, opts ...Option) (*GeminiClient, error) {
	client := &GeminiClient{
		location:           "us-central1",
		defaultModel:       "gemini-2.0-flash-exp",
		defaultContentType: "application/json",
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

	client.client = genaiClient

	return client, nil
}

func (x *GeminiClient) StartChat(options ...gemini.Option) interfaces.LLMSession {
	cfg := gemini.NewConfig(options...)

	genaiModel := x.defaultModel
	if cfg.Model() != "" {
		genaiModel = cfg.Model()
	}
	model := x.client.GenerativeModel(genaiModel)

	model.GenerationConfig.ResponseMIMEType = x.defaultContentType
	if cfg.ContentType() != "" {
		model.GenerationConfig.ResponseMIMEType = cfg.ContentType()
	}

	model.Tools = cfg.Tools()
	model.ToolConfig = cfg.ToolConfig()

	ssn := model.StartChat()
	if history := cfg.History(); history != nil {
		ssn.History = history.ToContents()
	}

	return &GeminiSession{session: ssn}
}

func (x *GeminiClient) SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error) {
	resp, err := x.client.GenerativeModel(x.defaultModel).GenerateContent(ctx, msg...)
	logging.From(ctx).Debug("GeminiClient: sent message", "resp", resp, "parts", msg)

	if err != nil {
		return nil, goerr.Wrap(err, "failed to send message")
	}

	return resp, nil
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
