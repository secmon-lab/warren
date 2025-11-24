package prompt_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/service/prompt"
)

func TestGeneratePromptWithParams(t *testing.T) {
	t.Run("generates prompt with custom params", func(t *testing.T) {
		// Create temp directory for templates
		tmpDir := t.TempDir()

		// Create template file with custom params
		templateContent := `Severity: {{.severity_threshold}}
Include Context: {{.include_context}}
Alert ID: {{.Alert.ID}}`
		templatePath := filepath.Join(tmpDir, "test.tmpl")
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		gt.NoError(t, err)

		// Create prompt service
		svc, err := prompt.New(tmpDir)
		gt.NoError(t, err)

		// Create alert
		ctx := context.Background()
		a := alert.New(ctx, "test-schema", nil, alert.Metadata{})

		// Generate prompt with params
		params := map[string]any{
			"severity_threshold": "high",
			"include_context":    true,
		}
		result, err := svc.GeneratePromptWithParams(ctx, "test.tmpl", &a, params)

		gt.NoError(t, err)
		gt.True(t, strings.Contains(result, "Severity: high"))
		gt.True(t, strings.Contains(result, "Include Context: true"))
		gt.True(t, strings.Contains(result, "Alert ID: "+string(a.ID)))
	})

	t.Run("works without params", func(t *testing.T) {
		// Create temp directory for templates
		tmpDir := t.TempDir()

		// Create template file without custom params
		templateContent := `Alert ID: {{.Alert.ID}}
Alert Schema: {{.Alert.Schema}}`
		templatePath := filepath.Join(tmpDir, "simple.tmpl")
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		gt.NoError(t, err)

		// Create prompt service
		svc, err := prompt.New(tmpDir)
		gt.NoError(t, err)

		// Create alert
		ctx := context.Background()
		a := alert.New(ctx, "guardduty", nil, alert.Metadata{})

		// Generate prompt without params
		result, err := svc.GeneratePromptWithParams(ctx, "simple.tmpl", &a, nil)

		gt.NoError(t, err)
		gt.True(t, strings.Contains(result, "Alert ID: "+string(a.ID)))
		gt.True(t, strings.Contains(result, "Alert Schema: guardduty"))
	})

	t.Run("GeneratePrompt calls GeneratePromptWithParams", func(t *testing.T) {
		// Create temp directory for templates
		tmpDir := t.TempDir()

		// Create template file
		templateContent := `Alert Schema: {{.Alert.Schema}}`
		templatePath := filepath.Join(tmpDir, "basic.tmpl")
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		gt.NoError(t, err)

		// Create prompt service
		svc, err := prompt.New(tmpDir)
		gt.NoError(t, err)

		// Create alert
		ctx := context.Background()
		a := alert.New(ctx, "vpc_flow", nil, alert.Metadata{})

		// Use GeneratePrompt (without params)
		result, err := svc.GeneratePrompt(ctx, "basic.tmpl", &a)

		gt.NoError(t, err)
		gt.True(t, strings.Contains(result, "Alert Schema: vpc_flow"))
	})
}
