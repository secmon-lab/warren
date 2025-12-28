package bigquery_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	bqagent "github.com/secmon-lab/warren/pkg/agents/bigquery"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

// TestAgent_InterfaceTool verifies that Agent implements interfaces.Tool
func TestAgent_InterfaceTool(t *testing.T) {
	ctx := context.Background()

	config := &bqagent.Config{
		Tables: []bqagent.TableConfig{
			{
				ProjectID:   "test-project",
				DatasetID:   "test-dataset",
				TableID:     "test-table",
				Description: "Test table for security logs",
			},
			{
				ProjectID:   "test-project",
				DatasetID:   "test-dataset",
				TableID:     "another-table",
				Description: "Another table for auth logs",
			},
		},
		ScanSizeLimit: 1024 * 1024 * 1024, // 1GB
		QueryTimeout:  5 * time.Minute,
	}

	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	// Memory service is created inside agent Init()

	agent := bqagent.NewAgent(config, llmClient, repo)

	// Verify interfaces.Tool interface implementation
	var _ interfaces.Tool = agent

	// Test Name()
	gt.V(t, agent.Name()).Equal("bigquery")

	// Test Configure() - should succeed when enabled
	gt.NoError(t, agent.Configure(ctx))

	// Test LogValue()
	logValue := agent.LogValue()
	gt.V(t, logValue).NotNil()

	// Test Helper()
	gt.V(t, agent.Helper()).Nil()

	// Test Prompt()
	prompt, err := agent.Prompt(ctx)
	gt.NoError(t, err)
	gt.V(t, prompt).NotEqual("")
	gt.True(t, len(prompt) > 0)

	// Verify prompt contains table information
	gt.True(t, strings.Contains(prompt, "BigQuery Agent"))
	gt.True(t, strings.Contains(prompt, "query_bigquery"))
	gt.True(t, strings.Contains(prompt, "test-project.test-dataset.test-table"))
	gt.True(t, strings.Contains(prompt, "Test table for security logs"))
	gt.True(t, strings.Contains(prompt, "test-project.test-dataset.another-table"))
	gt.True(t, strings.Contains(prompt, "Another table for auth logs"))
	gt.True(t, strings.Contains(prompt, "GB"))   // Scan limit
	gt.True(t, strings.Contains(prompt, "5m0s")) // Query timeout

	// Verify prompt contains new guidelines
	gt.True(t, strings.Contains(prompt, "MUST check table schemas"))
	gt.True(t, strings.Contains(prompt, "raw data records"))

	// Verify prompt contains usage instructions
	gt.True(t, strings.Contains(prompt, "Do NOT specify table names"))
	gt.True(t, strings.Contains(prompt, "WHAT information you need"))
}

// TestAgent_InterfaceTool_Disabled verifies behavior when agent is not configured
func TestAgent_InterfaceTool_Disabled(t *testing.T) {
	ctx := context.Background()

	// Create agent without config (disabled state)
	agent := bqagent.New()

	// Verify interfaces.Tool interface implementation
	var _ interfaces.Tool = agent

	// Test Configure() - should return ErrActionUnavailable when disabled
	err := agent.Configure(ctx)
	gt.Error(t, err)
	gt.Equal(t, err, errutil.ErrActionUnavailable)

	// Test Prompt() - should return empty string when disabled
	prompt, err := agent.Prompt(ctx)
	gt.NoError(t, err)
	gt.Equal(t, prompt, "")

	// Test LogValue() - should indicate disabled state
	logValue := agent.LogValue()
	gt.V(t, logValue).NotNil()
}
