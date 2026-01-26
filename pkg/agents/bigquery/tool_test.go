package bigquery_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	bqagent "github.com/secmon-lab/warren/pkg/agents/bigquery"
	"github.com/secmon-lab/warren/pkg/repository"
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

	agent := bqagent.NewAgent(config, llmClient, repo)

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

	agent := bqagent.NewAgent(config, llmClient, repo)

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

	agent := bqagent.NewAgent(config, llmClient, repo)

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

	agent := bqagent.NewAgent(config, llmClient, repo)

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

			agent := bqagent.NewAgent(tt.config, llmClient, repo)
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

	agent := bqagent.NewAgent(config, llmClient, repo)

	specs, err := agent.Specs(ctx)
	gt.NoError(t, err)
	gt.V(t, len(specs)).Equal(1)

	// The agent description should mention available tables
	description := specs[0].Description
	gt.V(t, description).NotEqual("")
	// Description should be non-empty and provide guidance
}

func TestInternalTool_GetRunbook(t *testing.T) {
	ctx := context.Background()

	t.Run("retrieve runbook by ID", func(t *testing.T) {
		// Load config with runbooks
		configPath := "testdata/config_with_runbooks.yaml"
		runbookDir := "testdata/runbooks"
		cfg, err := bqagent.LoadConfigWithRunbooks(ctx, configPath, []string{runbookDir})
		gt.NoError(t, err)

		// Find runbook ID by title (IDs are UUIDs)
		var targetID string
		for id, entry := range cfg.Runbooks {
			if entry.Title == "Failed Login Investigation" {
				targetID = id.String()
				break
			}
		}
		gt.V(t, targetID != "").Equal(true)

		// Call get_runbook through internal tool
		tool := bqagent.ExportNewInternalTool(cfg, "test-project")
		result, err := tool.Run(ctx, "get_runbook", map[string]any{
			"runbook_id": targetID,
		})
		gt.NoError(t, err)
		gt.V(t, result).NotNil()

		// Verify result structure
		gt.V(t, result["runbook_id"]).Equal(targetID)
		gt.V(t, result["title"]).Equal("Failed Login Investigation")
		gt.V(t, result["description"]).NotNil()
		gt.V(t, result["sql"]).NotNil()

		// Verify SQL content matches expected content
		sqlContent, ok := result["sql"].(string)
		gt.V(t, ok).Equal(true)
		gt.V(t, len(sqlContent) > 0).Equal(true)
		gt.S(t, sqlContent).Contains("-- Title: Failed Login Investigation")
		gt.S(t, sqlContent).Contains("SELECT")
		gt.S(t, sqlContent).Contains("FROM `project.dataset.auth_logs`")
		gt.S(t, sqlContent).Contains("event_type = 'login_failed'")
	})

	t.Run("runbook not found", func(t *testing.T) {
		configPath := "testdata/config_with_runbooks.yaml"
		runbookDir := "testdata/runbooks"
		cfg, err := bqagent.LoadConfigWithRunbooks(ctx, configPath, []string{runbookDir})
		gt.NoError(t, err)

		tool := bqagent.ExportNewInternalTool(cfg, "test-project")
		_, err = tool.Run(ctx, "get_runbook", map[string]any{
			"runbook_id": "nonexistent",
		})
		gt.Error(t, err)
	})

	t.Run("missing runbook_id parameter", func(t *testing.T) {
		configPath := "testdata/config_with_runbooks.yaml"
		runbookDir := "testdata/runbooks"
		cfg, err := bqagent.LoadConfigWithRunbooks(ctx, configPath, []string{runbookDir})
		gt.NoError(t, err)

		tool := bqagent.ExportNewInternalTool(cfg, "test-project")
		_, err = tool.Run(ctx, "get_runbook", map[string]any{})
		gt.Error(t, err)
	})
}

func TestInternalTool_Specs_WithRunbooks(t *testing.T) {
	ctx := context.Background()

	t.Run("specs include get_runbook when runbooks configured", func(t *testing.T) {
		configPath := "testdata/config_with_runbooks.yaml"
		runbookDir := "testdata/runbooks"
		cfg, err := bqagent.LoadConfigWithRunbooks(ctx, configPath, []string{runbookDir})
		gt.NoError(t, err)

		tool := bqagent.ExportNewInternalTool(cfg, "test-project")
		specs, err := tool.Specs(ctx)
		gt.NoError(t, err)

		// Should have bigquery_query, bigquery_schema, and get_runbook
		gt.V(t, len(specs)).Equal(3)

		// Find get_runbook spec
		var getRunbookSpec *bqagent.ToolSpec
		for i := range specs {
			if specs[i].Name == "get_runbook" {
				getRunbookSpec = &specs[i]
				break
			}
		}

		gt.V(t, getRunbookSpec).NotNil()
		gt.V(t, getRunbookSpec.Name).Equal("get_runbook")
		gt.True(t, getRunbookSpec.Parameters["runbook_id"].Required)
	})

	t.Run("specs exclude get_runbook when no runbooks", func(t *testing.T) {
		configPath := "testdata/config_with_runbooks.yaml"
		cfg, err := bqagent.LoadConfig(configPath)
		gt.NoError(t, err)

		tool := bqagent.ExportNewInternalTool(cfg, "test-project")
		specs, err := tool.Specs(ctx)
		gt.NoError(t, err)

		// Should only have bigquery_query and bigquery_schema
		gt.V(t, len(specs)).Equal(2)

		// Verify get_runbook is not present
		for _, spec := range specs {
			gt.V(t, spec.Name).NotEqual("get_runbook")
		}
	})
}
