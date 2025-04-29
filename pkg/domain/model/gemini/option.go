package gemini

import (
	"cloud.google.com/go/vertexai/genai"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
)

type Config struct {
	model       string
	tools       []*genai.Tool
	toolConfig  *genai.ToolConfig
	contentType string
	history     *session.History
}

func NewConfig(opts ...Option) *Config {
	cfg := &Config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func (c *Config) Model() string {
	return c.model
}

func (c *Config) Tools() []*genai.Tool {
	return c.tools
}

func (c *Config) ToolConfig() *genai.ToolConfig {
	return c.toolConfig
}

func (c *Config) ContentType() string {
	return c.contentType
}

func (c *Config) History() *session.History {
	return c.history
}

type Option func(*Config)

func WithModel(model string) Option {
	return func(c *Config) {
		c.model = model
	}
}

func WithContentType(contentType string) Option {
	return func(c *Config) {
		c.contentType = contentType
	}
}

func WithTools(tools []*genai.Tool) Option {
	return func(c *Config) {
		c.tools = tools
	}
}

func WithToolConfig(toolConfig *genai.ToolConfig) Option {
	return func(c *Config) {
		c.toolConfig = toolConfig
	}
}

func WithHistory(history *session.History) Option {
	return func(c *Config) {
		c.history = history
	}
}
