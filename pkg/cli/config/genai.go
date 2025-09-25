package config

import (
	"log/slog"

	"github.com/urfave/cli/v3"
)

// GenAI represents configuration for GenAI functionality including prompt templates
type GenAI struct {
	promptDir string
}

// Flags returns CLI flags for GenAI configuration
func (x *GenAI) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "prompt-dir",
			Usage:       "Directory containing prompt template files",
			Category:    "GenAI",
			Sources:     cli.EnvVars("WARREN_PROMPT_DIR"),
			Destination: &x.promptDir,
		},
	}
}

// LogValue returns a structured log value for GenAI configuration
func (x GenAI) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("prompt_dir", x.promptDir),
	)
}

// GetPromptDir returns the configured prompt directory path
func (x *GenAI) GetPromptDir() string {
	return x.promptDir
}

// IsConfigured returns true if prompt directory is configured
func (x *GenAI) IsConfigured() bool {
	return x.promptDir != ""
}
