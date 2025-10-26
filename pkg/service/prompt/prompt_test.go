package prompt_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/prompt"
)

func TestPromptService(t *testing.T) {
	// Create temporary directory for test templates
	tempDir, err := os.MkdirTemp("", "warren-prompt-test-*")
	gt.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Create test template
	templateContent := `You are a security analyst. Analyze the following alert:

Title: {{.Alert.Metadata.Title}}
Description: {{.Alert.Metadata.Description}}
Schema: {{.Alert.Schema}}

Please provide a summary of this security alert.`

	templatePath := filepath.Join(tempDir, "test_template.tmpl")
	err = os.WriteFile(templatePath, []byte(templateContent), 0644)
	gt.NoError(t, err)

	ctx := context.Background()

	t.Run("successful service creation and prompt generation", func(t *testing.T) {
		service, err := prompt.New(tempDir)
		gt.NoError(t, err)
		gt.NotNil(t, service)

		testAlert := &alert.Alert{
			ID:     types.NewAlertID(),
			Schema: "test_schema",
			Metadata: alert.Metadata{
				Title:       "Test Security Alert",
				Description: "This is a test security alert description",
			},
		}

		generatedPrompt, err := service.GeneratePrompt(ctx, "test_template.tmpl", testAlert)
		gt.NoError(t, err)
		gt.S(t, generatedPrompt).Contains("Test Security Alert")
		gt.S(t, generatedPrompt).Contains("This is a test security alert description")
		gt.S(t, generatedPrompt).Contains("test_schema")
	})

	t.Run("template not found", func(t *testing.T) {
		service, err := prompt.New(tempDir)
		gt.NoError(t, err)

		testAlert := &alert.Alert{
			ID:     types.NewAlertID(),
			Schema: "test_schema",
			Metadata: alert.Metadata{
				Title: "Test Alert",
			},
		}

		_, err = service.GeneratePrompt(ctx, "nonexistent_template", testAlert)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("prompt template not found")
	})

	t.Run("empty prompt directory", func(t *testing.T) {
		service, err := prompt.New("")
		gt.NoError(t, err)
		gt.NotNil(t, service)

		testAlert := &alert.Alert{
			ID: types.NewAlertID(),
		}

		_, err = service.GeneratePrompt(ctx, "any_template", testAlert)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("prompt template not found")
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		_, err := prompt.New("/nonexistent/directory")
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("prompt directory does not exist")
	})

	t.Run("multiple template formats", func(t *testing.T) {
		// Create additional template with .txt extension
		txtContent := `Simple text template: {{.Alert.Metadata.Title}}`
		txtPath := filepath.Join(tempDir, "simple.txt")
		err = os.WriteFile(txtPath, []byte(txtContent), 0644)
		gt.NoError(t, err)

		// Create file with unsupported extension (should be ignored)
		ignoredPath := filepath.Join(tempDir, "ignored.json")
		err = os.WriteFile(ignoredPath, []byte(`{"ignored": true}`), 0644)
		gt.NoError(t, err)

		service, err := prompt.New(tempDir)
		gt.NoError(t, err)

		testAlert := &alert.Alert{
			Metadata: alert.Metadata{
				Title: "Test Alert Title",
			},
		}

		// Test .tmpl template
		tmplPrompt, err := service.GeneratePrompt(ctx, "test_template.tmpl", testAlert)
		gt.NoError(t, err)
		gt.S(t, tmplPrompt).Contains("Test Alert Title")

		// Test .txt template
		txtPrompt, err := service.GeneratePrompt(ctx, "simple.txt", testAlert)
		gt.NoError(t, err)
		gt.Equal(t, txtPrompt, "Simple text template: Test Alert Title")

		// Ignored file should not be available
		_, err = service.GeneratePrompt(ctx, "ignored", testAlert)
		gt.Error(t, err)
	})
}
