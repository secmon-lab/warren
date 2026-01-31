package slack

import (
	"context"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	agentmemory "github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/service/memory"
)

// Export for testing

// NewAgentForTest creates an Agent instance for testing with direct configuration
func NewAgentForTest(llmClient gollem.LLMClient, repo interfaces.Repository, slackClient interfaces.SlackClient) *Agent {
	return &Agent{
		llmClient: llmClient,
		repo:      repo,
		slackClient: slackClient,
		internalTool: &internalTool{
			slackClient: slackClient,
		},
		memoryService: memory.New("slack_search", llmClient, repo),
	}
}

type InternalTool = internalTool

// NewInternalToolForTest creates an internalTool for testing
func NewInternalToolForTest(slackClient interfaces.SlackClient, maxLimit int) *internalTool {
	return &internalTool{
		slackClient: slackClient,
		maxLimit:    maxLimit,
	}
}

// ExportedExtractRecords is exported for testing
func (a *Agent) ExportedExtractRecords(ctx context.Context, originalQuery string, session gollem.Session) ([]map[string]any, error) {
	return a.extractRecords(ctx, originalQuery, session)
}

// ExportedCreateMiddleware is exported for testing
func (a *Agent) ExportedCreateMiddleware() func(gollem.SubAgentHandler) gollem.SubAgentHandler {
	return a.createMiddleware()
}

// ExportedBuildSystemPrompt is exported for testing
func ExportedBuildSystemPrompt() (string, error) {
	return buildSystemPrompt()
}

// ExportedNewPromptTemplate is exported for testing
func ExportedNewPromptTemplate() (*gollem.PromptTemplate, error) {
	return newPromptTemplate()
}

// ExportedFormatMemoryContext is exported for testing
func ExportedFormatMemoryContext(memories []*agentmemory.AgentMemory) string {
	return formatMemoryContext(memories)
}
