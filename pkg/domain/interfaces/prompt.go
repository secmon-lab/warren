package interfaces

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/alert"
)

// PromptService handles prompt template management and generation
type PromptService interface {
	// GeneratePrompt generates a prompt from template name and alert data
	GeneratePrompt(ctx context.Context, templateName string, alert *alert.Alert) (string, error)
}
