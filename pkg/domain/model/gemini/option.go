package gemini

import "cloud.google.com/go/vertexai/genai"

type Config struct {
	tools      []*genai.Tool
	toolConfig *genai.ToolConfig
}

func NewConfig(opts ...Option) *Config {
	cfg := &Config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func (c *Config) Tools() []*genai.Tool {
	return c.tools
}

func (c *Config) ToolConfig() *genai.ToolConfig {
	return c.toolConfig
}

type Option func(*Config)

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
