package config

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/llm/claude"
	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/urfave/cli/v3"
)

// llmEmbeddingAdapter adapts gollem.LLMClient to interfaces.EmbeddingClient
type llmEmbeddingAdapter struct {
	client gollem.LLMClient
}

// Embeddings implements interfaces.EmbeddingClient.Embeddings
func (a *llmEmbeddingAdapter) Embeddings(ctx context.Context, texts []string, dimensionality int) ([][]float32, error) {
	embeddings, err := a.client.GenerateEmbedding(ctx, dimensionality, texts)
	if err != nil {
		return nil, err
	}

	// Convert from [][]float64 to [][]float32
	result := make([][]float32, len(embeddings))
	for i, embedding := range embeddings {
		result[i] = make([]float32, len(embedding))
		for j, val := range embedding {
			result[i][j] = float32(val)
		}
	}

	return result, nil
}

type LLMCfg struct {
	// Claude configuration
	claudeModel     string
	claudeProjectID string
	claudeLocation  string

	// Gemini configuration
	geminiModel          string
	geminiEmbeddingModel string
	geminiProjectID      string
	geminiLocation       string
}

func (x *LLMCfg) Flags() []cli.Flag {
	return []cli.Flag{
		// Claude flags
		&cli.StringFlag{
			Name:        "claude-model",
			Usage:       "Claude model name",
			Sources:     cli.EnvVars("WARREN_CLAUDE_MODEL"),
			Value:       "claude-sonnet-4@20250514",
			Destination: &x.claudeModel,
			Category:    "Claude",
		},
		&cli.StringFlag{
			Name:        "claude-project-id",
			Usage:       "Google Cloud Project ID for Claude Vertex AI",
			Sources:     cli.EnvVars("WARREN_CLAUDE_PROJECT_ID"),
			Destination: &x.claudeProjectID,
			Category:    "Claude",
		},
		&cli.StringFlag{
			Name:        "claude-location",
			Usage:       "Google Cloud location for Claude Vertex AI",
			Sources:     cli.EnvVars("WARREN_CLAUDE_LOCATION"),
			Value:       "us-east5",
			Destination: &x.claudeLocation,
			Category:    "Claude",
		},
		// Gemini flags
		&cli.StringFlag{
			Name:        "gemini-model",
			Usage:       "Gemini model",
			Destination: &x.geminiModel,
			Category:    "Gemini",
			Value:       "gemini-2.5-flash",
			Sources:     cli.EnvVars("WARREN_GEMINI_MODEL"),
		},
		&cli.StringFlag{
			Name:        "gemini-project-id",
			Usage:       "GCP Project ID for Vertex AI",
			Required:    true,
			Destination: &x.geminiProjectID,
			Category:    "Gemini",
			Sources:     cli.EnvVars("WARREN_GEMINI_PROJECT_ID"),
		},
		&cli.StringFlag{
			Name:        "gemini-location",
			Usage:       "GCP Location for Vertex AI",
			Value:       "us-central1",
			Destination: &x.geminiLocation,
			Category:    "Gemini",
			Sources:     cli.EnvVars("WARREN_GEMINI_LOCATION"),
		},
		&cli.StringFlag{
			Name:        "gemini-embedding-model",
			Usage:       "Gemini embedding model",
			Destination: &x.geminiEmbeddingModel,
			Category:    "Embedding model",
			Sources:     cli.EnvVars("WARREN_GEMINI_EMBEDDING_MODEL"),
		},
	}
}

func (x LLMCfg) LogValue() slog.Value {
	attrs := []slog.Attr{}

	// Add Claude info if configured
	if x.claudeProjectID != "" {
		attrs = append(attrs,
			slog.String("claude_model", x.claudeModel),
			slog.String("claude_project_id", x.claudeProjectID),
			slog.String("claude_location", x.claudeLocation),
		)
	}

	// Add Gemini info
	attrs = append(attrs,
		slog.String("gemini_model", x.geminiModel),
		slog.String("gemini_project_id", x.geminiProjectID),
		slog.String("gemini_location", x.geminiLocation),
	)

	return slog.GroupValue(attrs...)
}

// Configure creates and returns an LLM client, preferring Claude if configured
func (x *LLMCfg) Configure(ctx context.Context) (gollem.LLMClient, error) {
	// Prefer Claude if project ID is configured
	if x.claudeProjectID != "" {
		return x.configureClaude(ctx)
	}

	// Fall back to Gemini
	return x.configureGemini(ctx)
}

func (x *LLMCfg) configureClaude(ctx context.Context) (gollem.LLMClient, error) {
	options := []claude.VertexOption{
		claude.WithVertexModel(x.claudeModel),
	}

	client, err := claude.NewWithVertex(ctx, x.claudeLocation, x.claudeProjectID, options...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create Claude Vertex AI client",
			goerr.V("projectID", x.claudeProjectID),
			goerr.V("location", x.claudeLocation),
			goerr.V("model", x.claudeModel))
	}

	return client, nil
}

func (x *LLMCfg) configureGemini(ctx context.Context) (gollem.LLMClient, error) {
	options := []gemini.Option{
		gemini.WithModel(x.geminiModel),
	}
	if x.geminiEmbeddingModel != "" {
		options = append(options, gemini.WithEmbeddingModel(x.geminiEmbeddingModel))
	}

	client, err := gemini.New(ctx, x.geminiProjectID, x.geminiLocation, options...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create Gemini client")
	}

	return client, nil
}

// IsClaudeConfigured returns true if Claude configuration is available
func (x *LLMCfg) IsClaudeConfigured() bool {
	return x.claudeProjectID != ""
}

// IsGeminiConfigured returns true if Gemini configuration is available
func (x *LLMCfg) IsGeminiConfigured() bool {
	return x.geminiProjectID != ""
}

// GetActiveProvider returns the name of the active LLM provider
func (x *LLMCfg) GetActiveProvider() string {
	if x.IsClaudeConfigured() {
		return "claude"
	}
	if x.IsGeminiConfigured() {
		return "gemini"
	}
	return "none"
}

// ConfigureEmbeddingClient creates and returns an embedding client adapter from the configured LLM client
func (x *LLMCfg) ConfigureEmbeddingClient(ctx context.Context) (interfaces.EmbeddingClient, error) {
	llmClient, err := x.Configure(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to configure LLM client for embedding")
	}
	return &llmEmbeddingAdapter{client: llmClient}, nil
}
