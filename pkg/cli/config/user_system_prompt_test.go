package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config"
)

func TestUserSystemPrompt(t *testing.T) {
	t.Run("returns empty string when no file path is set", func(t *testing.T) {
		cfg := config.NewUserSystemPromptWithPath("")
		result, err := cfg.Configure()
		gt.NoError(t, err)
		gt.V(t, result).Equal("")
	})

	t.Run("reads file content successfully", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "prompt.md")
		content := "## Environment\nProduction environment for AWS security monitoring."
		err := os.WriteFile(filePath, []byte(content), 0o644)
		gt.NoError(t, err)

		cfg := config.NewUserSystemPromptWithPath(filePath)
		result, err := cfg.Configure()
		gt.NoError(t, err)
		gt.V(t, result).Equal(content)
	})

	t.Run("returns error when file does not exist", func(t *testing.T) {
		cfg := config.NewUserSystemPromptWithPath("/nonexistent/path/prompt.md")
		_, err := cfg.Configure()
		gt.Error(t, err)
	})
}
