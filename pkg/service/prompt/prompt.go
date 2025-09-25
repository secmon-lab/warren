package prompt

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

// Service implements the PromptService interface
type Service struct {
	templates map[string]*template.Template
}

// New creates a new prompt service with preloaded templates from the specified directory
func New(promptDir string) (interfaces.PromptService, error) {
	service := &Service{
		templates: make(map[string]*template.Template),
	}

	if promptDir != "" {
		if err := service.loadTemplates(promptDir); err != nil {
			return nil, goerr.Wrap(err, "failed to load prompt templates", goerr.V("prompt_dir", promptDir))
		}
	}

	return service, nil
}

// loadTemplates loads all template files from the specified directory
func (s *Service) loadTemplates(promptDir string) error {
	if _, err := os.Stat(promptDir); os.IsNotExist(err) {
		return goerr.New("prompt directory does not exist", goerr.V("prompt_dir", promptDir))
	}

	var loadedFiles []string

	err := filepath.Walk(promptDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return goerr.Wrap(err, "failed to walk prompt directory", goerr.V("path", path))
		}

		if info.IsDir() {
			return nil
		}

		// Only process .txt and .tmpl files
		ext := filepath.Ext(path)
		if ext != ".txt" && ext != ".tmpl" && ext != ".md" {
			return nil
		}

		// Use filename as template name (with extension)
		name := filepath.Base(path)

		tmpl, err := template.ParseFiles(path)
		if err != nil {
			return goerr.Wrap(err, "failed to parse template", goerr.V("path", path), goerr.V("name", name))
		}

		s.templates[name] = tmpl
		loadedFiles = append(loadedFiles, filepath.Base(path))
		return nil
	})

	if err != nil {
		return err
	}

	// Log the loaded prompt files
	if len(loadedFiles) > 0 {
		logging.Default().Info("loaded prompt templates",
			"prompt_dir", promptDir,
			"files", loadedFiles,
			"count", len(loadedFiles))
	} else {
		logging.Default().Warn("no prompt template files found",
			"prompt_dir", promptDir,
			"supported_extensions", []string{".txt", ".tmpl", ".md"})
	}

	return nil
}

// GeneratePrompt generates a prompt from template name and alert data
func (s *Service) GeneratePrompt(ctx context.Context, templateName string, alert *alert.Alert) (string, error) {
	logger := logging.From(ctx)

	tmpl, exists := s.templates[templateName]
	if !exists {
		return "", goerr.New("prompt template not found", goerr.V("template_name", templateName))
	}

	// Execute template with alert data
	var buf bytes.Buffer
	templateData := map[string]interface{}{
		"Alert": alert,
	}

	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", goerr.Wrap(err, "failed to execute prompt template",
			goerr.V("template_name", templateName),
			goerr.V("alert_id", alert.ID))
	}

	prompt := strings.TrimSpace(buf.String())

	logger.Debug("generated prompt from template",
		"template_name", templateName,
		"alert_id", alert.ID,
		"prompt_length", len(prompt))

	return prompt, nil
}
