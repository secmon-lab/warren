package config

import (
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/urfave/cli/v3"
)

// UserSystemPrompt holds CLI configuration for user system prompt file.
type UserSystemPrompt struct {
	filePath string
}

// Flags returns CLI flags for user system prompt configuration.
func (x *UserSystemPrompt) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "user-system-prompt",
			Usage:       "Path to a user system prompt file (markdown format, h2+ headings recommended)",
			Destination: &x.filePath,
			Sources:     cli.EnvVars("WARREN_USER_SYSTEM_PROMPT"),
		},
	}
}

// Configure reads the user system prompt file and returns its content.
// Returns empty string if no file path is configured.
func (x *UserSystemPrompt) Configure() (string, error) {
	if x.filePath == "" {
		return "", nil
	}

	data, err := os.ReadFile(x.filePath)
	if err != nil {
		return "", goerr.Wrap(err, "failed to read user system prompt file",
			goerr.V("path", x.filePath),
		)
	}

	return string(data), nil
}
