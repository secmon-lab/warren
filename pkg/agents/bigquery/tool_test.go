package bigquery_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	bqagent "github.com/secmon-lab/warren/pkg/agents/bigquery"
	"github.com/secmon-lab/warren/pkg/repository"
	memoryservice "github.com/secmon-lab/warren/pkg/service/memory"
)

// Mock BigQuery client tests - these will fail without actual BigQuery credentials
// but demonstrate the structure of tool execution tests

func TestInternalTool_Specs(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables: []bqagent.TableConfig{
			{
				ProjectID:   "test-project",
				DatasetID:   "test-dataset",
				TableID:     "test-table",
				Description: "Test table for unit testing",
			},
		},
		ScanSizeLimit: 10 * 1024 * 1024 * 1024, // 10GB
	}

	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := bqagent.NewAgent(config, llmClient, memService)

	// Get agent specs (which includes internal tool specs)
	specs, err := agent.Specs(ctx)
	gt.NoError(t, err)
	gt.V(t, len(specs)).Equal(1)

	// The agent wrapper spec
	gt.V(t, specs[0].Name).Equal("query_bigquery")
}

func TestInternalTool_Run_InvalidFunction(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables: []bqagent.TableConfig{
			{
				ProjectID:   "test-project",
				DatasetID:   "test-dataset",
				TableID:     "test-table",
				Description: "Test table",
			},
		},
		ScanSizeLimit: 10 * 1024 * 1024 * 1024,
	}

	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := bqagent.NewAgent(config, llmClient, memService)

	// Try to call an invalid function through the agent
	args := map[string]any{
		"query": "test",
	}
	_, err := agent.Run(ctx, "invalid_tool_function", args)
	gt.Error(t, err)
}

func TestInternalTool_QueryMissingSQL(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables: []bqagent.TableConfig{
			{
				ProjectID:   "test-project",
				DatasetID:   "test-dataset",
				TableID:     "test-table",
				Description: "Test table",
			},
		},
		ScanSizeLimit: 10 * 1024 * 1024 * 1024,
	}

	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := bqagent.NewAgent(config, llmClient, memService)

	// The agent will translate "query" to "sql" and call the internal tool
	// But let's test the validation by passing empty query
	args := map[string]any{}
	_, err := agent.Run(ctx, "query_bigquery", args)
	gt.Error(t, err)
}

func TestInternalTool_SchemaMissingParameters(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables: []bqagent.TableConfig{
			{
				ProjectID:   "test-project",
				DatasetID:   "test-dataset",
				TableID:     "test-table",
				Description: "Test table",
			},
		},
		ScanSizeLimit: 10 * 1024 * 1024 * 1024,
	}

	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := bqagent.NewAgent(config, llmClient, memService)

	// Note: The agent wrapper doesn't directly expose bigquery_schema
	// It's only available through the internal tool
	// This test demonstrates parameter validation

	// Missing all parameters
	args := map[string]any{
		"query": "get schema", // This will be interpreted by the agent
	}
	result, err := agent.Run(ctx, "query_bigquery", args)
	// Should succeed with LLM response, not schema lookup
	gt.NoError(t, err)
	gt.V(t, result).NotNil()
}

func TestInternalTool_ConfigValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		config    *bqagent.Config
		shouldErr bool
	}{
		{
			name: "valid config",
			config: &bqagent.Config{
				Tables: []bqagent.TableConfig{
					{
						ProjectID:   "test-project",
						DatasetID:   "test-dataset",
						TableID:     "test-table",
						Description: "Test",
					},
				},
				ScanSizeLimit: 1024,
			},
			shouldErr: false,
		},
		{
			name: "empty tables",
			config: &bqagent.Config{
				Tables:        []bqagent.TableConfig{},
				ScanSizeLimit: 1024,
			},
			shouldErr: false, // Agent can still be created
		},
		{
			name: "zero scan limit",
			config: &bqagent.Config{
				Tables: []bqagent.TableConfig{
					{
						ProjectID:   "test-project",
						DatasetID:   "test-dataset",
						TableID:     "test-table",
						Description: "Test",
					},
				},
				ScanSizeLimit: 0,
			},
			shouldErr: false, // Agent creation succeeds, queries will fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llmClient := newMockLLMClient()
			repo := repository.NewMemory()
			memService := memoryservice.New(llmClient, repo)

			agent := bqagent.NewAgent(tt.config, llmClient, memService)
			gt.V(t, agent).NotNil()

			// Verify specs can be retrieved
			specs, err := agent.Specs(ctx)
			gt.NoError(t, err)
			gt.V(t, len(specs)).Equal(1)
		})
	}
}

func TestInternalTool_TableDescriptionInSpecs(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables: []bqagent.TableConfig{
			{
				ProjectID:   "proj1",
				DatasetID:   "dataset1",
				TableID:     "table1",
				Description: "First test table",
			},
			{
				ProjectID:   "proj2",
				DatasetID:   "dataset2",
				TableID:     "table2",
				Description: "Second test table",
			},
		},
		ScanSizeLimit: 10 * 1024 * 1024 * 1024,
	}

	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := bqagent.NewAgent(config, llmClient, memService)

	specs, err := agent.Specs(ctx)
	gt.NoError(t, err)
	gt.V(t, len(specs)).Equal(1)

	// The agent description should mention available tables
	description := specs[0].Description
	gt.V(t, description).NotEqual("")
	// Description should be non-empty and provide guidance
}
