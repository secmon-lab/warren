package config

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/cli/config/llm"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/urfave/cli/v3"
)

// LLMConfigFile holds the path to the LLM TOML configuration file.
type LLMConfigFile struct {
	path string
}

// Flags returns the CLI flag for --llm-config.
func (x *LLMConfigFile) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "llm-config",
			Usage:       "Path to the LLM TOML configuration file (required)",
			Sources:     cli.EnvVars("WARREN_LLM_CONFIG"),
			Destination: &x.path,
			Category:    "LLM",
		},
	}
}

// LogValue redacts the path is sensitive? No — it's just a path. Show it.
func (x LLMConfigFile) LogValue() slog.Value {
	return slog.GroupValue(slog.String("path", x.path))
}

// Path returns the configured path.
func (x *LLMConfigFile) Path() string { return x.path }

// Load loads the registry and runs HealthCheck. Both must succeed for the
// caller to proceed; misconfiguration fails fast at startup.
func (x *LLMConfigFile) Load(ctx context.Context) (*llm.Registry, error) {
	if x.path == "" {
		return nil, goerr.New("--llm-config is required (or set WARREN_LLM_CONFIG)")
	}
	reg, err := llm.Load(ctx, x.path)
	if err != nil {
		return nil, err
	}
	if err := reg.HealthCheck(ctx); err != nil {
		return nil, err
	}
	return reg, nil
}

// EmbeddingClientAdapter wraps a gollem embedding client into the
// interfaces.EmbeddingClient expected by warren's repository layer.
type embeddingClientAdapter struct {
	registry *llm.Registry
}

func NewEmbeddingClientAdapter(reg *llm.Registry) interfaces.EmbeddingClient {
	return &embeddingClientAdapter{registry: reg}
}

func (a *embeddingClientAdapter) Embeddings(ctx context.Context, texts []string, dimensionality int) ([][]float32, error) {
	embeddings, err := a.registry.Embedding().GenerateEmbedding(ctx, dimensionality, texts)
	if err != nil {
		return nil, err
	}
	result := make([][]float32, len(embeddings))
	for i, embedding := range embeddings {
		result[i] = make([]float32, len(embedding))
		for j, val := range embedding {
			result[i][j] = float32(val)
		}
	}
	return result, nil
}
