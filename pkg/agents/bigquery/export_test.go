package bigquery

import (
	"context"
	"time"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	agentmemory "github.com/secmon-lab/warren/pkg/domain/model/memory"
)

// Expose internal types and functions for testing

// NewAgentForTest creates an Agent instance for testing with direct configuration
func NewAgentForTest(config *Config, llmClient gollem.LLMClient, repo interfaces.Repository) *Agent {
	return New(llmClient, repo, config)
}

// GenerateKPTAnalysis is exported for testing
func (a *Agent) GenerateKPTAnalysis(ctx context.Context, query string, resp *gollem.ExecuteResponse, execErr error, duration time.Duration, session gollem.Session) ([]string, []string, []string, error) {
	return a.generateKPTAnalysis(ctx, query, resp, execErr, duration, session)
}

// ExportNewInternalTool creates a new internalTool instance for testing
func ExportNewInternalTool(config *Config, projectID string) gollem.ToolSet {
	return &internalTool{
		config:                    config,
		projectID:                 projectID,
		impersonateServiceAccount: config.ImpersonateServiceAccount,
	}
}

// ToolSpec is exported for testing
type ToolSpec = gollem.ToolSpec

// ExportedExtractRecords is exported for testing
func (a *Agent) ExportedExtractRecords(ctx context.Context, originalQuery string, session gollem.Session) ([]map[string]any, error) {
	return a.extractRecords(ctx, originalQuery, session)
}

// ExportedCreateMiddleware is exported for testing
func (a *Agent) ExportedCreateMiddleware() func(gollem.SubAgentHandler) gollem.SubAgentHandler {
	return a.createMiddleware()
}

// ExportedBuildSystemPrompt is exported for testing
func ExportedBuildSystemPrompt(config *Config) (string, error) {
	return buildSystemPrompt(config)
}

// ExportedNewPromptTemplate is exported for testing
func ExportedNewPromptTemplate() (*gollem.PromptTemplate, error) {
	return newPromptTemplate()
}

// ExportedFormatMemoryContext is exported for testing
func ExportedFormatMemoryContext(memories []*agentmemory.AgentMemory) string {
	return formatMemoryContext(memories)
}
