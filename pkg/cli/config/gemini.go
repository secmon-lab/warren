package config

import (
	"context"
	"log/slog"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/urfave/cli/v3"
)

type GeminiCfg struct {
	model     string
	projectID string
	location  string
}

func (x *GeminiCfg) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "gemini-model",
			Usage:       "Gemini model",
			Destination: &x.model,
			Category:    "Gemini",
			Value:       "gemini-flash-experimental",
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
	}
}

func (x GeminiCfg) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("model", x.model),
		slog.String("project_id", x.projectID),
		slog.String("location", x.location),
	)
}

func (x *GeminiCfg) Configure(ctx context.Context) (*genai.GenerativeModel, error) {
	client, err := genai.NewClient(ctx, x.projectID, x.location)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create vertex ai client")
	}

	model := client.GenerativeModel(x.model)
	return model, nil
}
