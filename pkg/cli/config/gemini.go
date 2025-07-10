package config

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/urfave/cli/v3"
)

type GeminiCfg struct {
	model          string
	embeddingModel string
	projectID      string
	location       string
}

func (x *GeminiCfg) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "gemini-model",
			Usage:       "Gemini model",
			Destination: &x.model,
			Category:    "Gemini",
			Value:       "gemini-2.0-flash",
		},
		&cli.StringFlag{
			Name:        "gemini-project-id",
			Usage:       "GCP Project ID for Vertex AI",
			Required:    true,
			Destination: &x.projectID,
			Category:    "Gemini",
			Sources:     cli.EnvVars("WARREN_GEMINI_PROJECT_ID"),
		},
		&cli.StringFlag{
			Name:        "gemini-location",
			Usage:       "GCP Location for Vertex AI",
			Value:       "us-central1",
			Destination: &x.location,
			Category:    "Gemini",
			Sources:     cli.EnvVars("WARREN_GEMINI_LOCATION"),
		},
		&cli.StringFlag{
			Name:        "gemini-embedding-model",
			Usage:       "Gemini embedding model",
			Destination: &x.embeddingModel,
			Category:    "Embedding model",
			Sources:     cli.EnvVars("WARREN_GEMINI_EMBEDDING_MODEL"),
		},
	}
}

func (x GeminiCfg) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("model", x.model),
		slog.String("project_id", x.projectID),
		slog.String("location", x.location),
	)
}

func (x *GeminiCfg) Configure(ctx context.Context) (*gemini.Client, error) {
	options := []gemini.Option{
		gemini.WithModel(x.model),
		// Temporarily use default settings like gollem test to isolate the issue
		// gemini.WithTemperature(0.1),     // Lower temperature for consistency
		// gemini.WithTopK(40),             // Controlled diversity
		// gemini.WithTopP(0.95),           // Nucleus sampling
		// gemini.WithMaxTokens(8192),      // Prevent oversized responses
	}
	if x.embeddingModel != "" {
		options = append(options, gemini.WithEmbeddingModel(x.embeddingModel))
	}
	client, err := gemini.New(ctx, x.projectID, x.location, options...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create vertex ai client")
	}

	return client, nil
}
