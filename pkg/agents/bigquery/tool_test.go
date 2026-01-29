package bigquery_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	bqagent "github.com/secmon-lab/warren/pkg/agents/bigquery"
)

// Tests for internal tool implementation

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

	tool := bqagent.ExportNewInternalTool(config, "test-project")

	specs, err := tool.Specs(ctx)
	gt.NoError(t, err)
	gt.V(t, len(specs)).Equal(2) // bigquery_query and bigquery_schema

	// Find query spec
	var querySpec *bqagent.ToolSpec
	for i := range specs {
		if specs[i].Name == "bigquery_query" {
			querySpec = &specs[i]
			break
		}
	}

	gt.V(t, querySpec).NotNil()
	gt.V(t, querySpec.Name).Equal("bigquery_query")
	gt.True(t, querySpec.Parameters["sql"].Required)
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
