package config

import (
	"log/slog"

	"github.com/secmon-lab/warren/pkg/adapter/embedding"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/urfave/cli/v3"
)

type EmbeddingCfg struct {
	projectID string
	location  string
	model     string
}

func (x *EmbeddingCfg) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "embedding-project-id",
			Usage:       "Embedding project ID",
			Destination: &x.projectID,
			Category:    "Embedding",
			Sources:     cli.EnvVars("WARREN_EMBEDDING_PROJECT_ID"),
		},
		&cli.StringFlag{
			Name:        "embedding-location",
			Usage:       "Embedding location",
			Destination: &x.location,
			Category:    "Embedding",
			Value:       "us-central1",
			Sources:     cli.EnvVars("WARREN_EMBEDDING_LOCATION"),
		},
		&cli.StringFlag{
			Name:        "embedding-model",
			Usage:       "Embedding model",
			Destination: &x.model,
			Category:    "Embedding",
			Value:       "text-embedding-004",
			Sources:     cli.EnvVars("WARREN_EMBEDDING_MODEL"),
		},
	}
}

func (x EmbeddingCfg) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("project_id", x.projectID),
		slog.String("location", x.location),
		slog.String("model", x.model),
	)
}

func (x *EmbeddingCfg) Configure() interfaces.EmbeddingClient {
	if x.projectID == "" {
		return nil
	}

	client := embedding.NewGemini(x.projectID,
		embedding.WithLocation(x.location),
		embedding.WithModelName(x.model),
	)
	return client
}
