package config

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/adapter/gemini"
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
	}
}

func (x GeminiCfg) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("model", x.model),
		slog.String("project_id", x.projectID),
		slog.String("location", x.location),
	)
}

func (x *GeminiCfg) Configure(ctx context.Context) (*gemini.GeminiClient, error) {
	client, err := gemini.New(ctx, x.projectID,
		gemini.WithLocation(x.location),
		gemini.WithModel(x.model),
		gemini.WithResponseMIMEType("application/json"),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create vertex ai client")
	}

	return client, nil
}
