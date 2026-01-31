package bigquery

import (
	"context"
	"time"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	agentmemory "github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/service/memory"
)

// Expose internal types and functions for testing

// NewAgentForTest creates an agent instance for testing with direct configuration
func NewAgentForTest(config *Config, llmClient gollem.LLMClient, repo interfaces.Repository, projectID string, impersonateServiceAccount string) *agent {
	return &agent{
		config:    config,
		llmClient: llmClient,
		repo:      repo,
		internalTool: &internalTool{
			config:                    config,
			projectID:                 projectID,
			impersonateServiceAccount: impersonateServiceAccount,
		},
		memoryService: memory.New("bigquery", llmClient, repo),
	}
}

// GenerateKPTAnalysis is exported for testing
func (a *agent) GenerateKPTAnalysis(ctx context.Context, query string, resp *gollem.ExecuteResponse, execErr error, duration time.Duration, session gollem.Session) ([]string, []string, []string, error) {
	return a.generateKPTAnalysis(ctx, query, resp, execErr, duration, session)
}

// ExportNewInternalTool creates a new internalTool instance for testing
func ExportNewInternalTool(config *Config, projectID string, impersonateServiceAccount string) gollem.ToolSet {
	return &internalTool{
		config:                    config,
		projectID:                 projectID,
		impersonateServiceAccount: impersonateServiceAccount,
	}
}

// ToolSpec is exported for testing
type ToolSpec = gollem.ToolSpec

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
