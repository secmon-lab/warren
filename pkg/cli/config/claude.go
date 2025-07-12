package config

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem/llm/claude"
	"github.com/urfave/cli/v3"
)

type ClaudeCfg struct {
	model     string
	ProjectID string
	location  string
}

func (x *ClaudeCfg) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "claude-model",
			Usage:       "Claude model name",
			Sources:     cli.EnvVars("WARREN_CLAUDE_MODEL"),
			Value:       "claude-sonnet-4@20250514",
			Destination: &x.model,
			Category:    "Claude",
		},
		&cli.StringFlag{
			Name:        "claude-project-id",
			Usage:       "Google Cloud Project ID for Claude Vertex AI",
			Sources:     cli.EnvVars("WARREN_CLAUDE_PROJECT_ID"),
			Destination: &x.ProjectID,
			Category:    "Claude",
		},
		&cli.StringFlag{
			Name:        "claude-location",
			Usage:       "Google Cloud location for Claude Vertex AI",
			Sources:     cli.EnvVars("WARREN_CLAUDE_LOCATION"),
			Value:       "us-central1",
			Destination: &x.location,
			Category:    "Claude",
		},
	}
}

func (x *ClaudeCfg) Configure(ctx context.Context) (*claude.VertexClient, error) {
	if x.ProjectID == "" {
		return nil, goerr.New("claude-project-id is required")
	}

	options := []claude.VertexOption{
		claude.WithVertexModel(x.model),
	}

	client, err := claude.NewWithVertex(ctx, x.location, x.ProjectID, options...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create Claude Vertex AI client",
			goerr.V("projectID", x.ProjectID),
			goerr.V("location", x.location),
			goerr.V("model", x.model))
	}

	return client, nil
}
