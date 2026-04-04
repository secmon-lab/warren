package slack

import (
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

// Export for testing

type InternalTool = internalTool

// NewInternalToolForTest creates an internalTool for testing
func NewInternalToolForTest(slackClient interfaces.SlackClient, maxLimit int) *internalTool {
	return &internalTool{
		slackClient: slackClient,
		maxLimit:    maxLimit,
	}
}

// NewToolSetForTest creates a toolSet instance for testing
func NewToolSetForTest(slackClient interfaces.SlackClient) *toolSet {
	return &toolSet{
		tool: &internalTool{slackClient: slackClient},
	}
}

// ExportedBuildSystemPrompt is exported for testing
func ExportedBuildSystemPrompt() (string, error) {
	return buildSystemPrompt()
}

// ExportedNewPromptTemplate is exported for testing
func ExportedNewPromptTemplate() (*gollem.PromptTemplate, error) {
	return newPromptTemplate()
}
