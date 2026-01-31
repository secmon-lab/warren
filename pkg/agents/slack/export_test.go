package slack

import (
	"context"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	agentmemory "github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/service/memory"
)

// Export for testing

// NewAgentForTest creates an agent instance for testing with direct configuration
func NewAgentForTest(llmClient gollem.LLMClient, repo interfaces.Repository, slackClient interfaces.SlackClient) *agent {
	return &agent{
		llmClient:   llmClient,
		repo:        repo,
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
func (a *agent) ExportedExtractRecords(ctx context.Context, originalQuery string, session gollem.Session) ([]map[string]any, error) {
	return a.extractRecords(ctx, originalQuery, session)
}

// ExportedCreateMiddleware is exported for testing
func (a *agent) ExportedCreateMiddleware() func(gollem.SubAgentHandler) gollem.SubAgentHandler {
	return a.createMiddleware()
}

// Name is exported for testing
func (a *agent) Name() string {
	return a.name()
}

// Description is exported for testing
func (a *agent) Description() string {
	return a.description()
}

// SubAgent is exported for testing
func (a *agent) SubAgent() (*gollem.SubAgent, error) {
	return a.subAgent()
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
