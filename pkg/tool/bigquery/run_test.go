package bigquery_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/bigquery"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/urfave/cli/v3"
)

func TestBigQuery(t *testing.T) {
	testCases := []struct {
		name     string
		funcName string
		args     map[string]any
		wantErr  bool
	}{
		{
			name:     "list datasets without config",
			funcName: "bigquery_list_dataset",
			args:     map[string]any{},
			wantErr:  true,
		},
		{
			name:     "execute query without config",
			funcName: "bigquery_query",
			args: map[string]any{
				"query": "SELECT 1",
			},
			wantErr: true,
		},
		{
			name:     "get results without config",
			funcName: "bigquery_result",
			args: map[string]any{
				"query_id": "test-query-id",
			},
			wantErr: true,
		},
		{
			name:     "get table summary without config",
			funcName: "bigquery_table_summary",
			args:     map[string]any{},
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var action bigquery.Action
			cmd := cli.Command{
				Name:  "bigquery",
				Flags: action.Flags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					resp, err := action.Run(ctx, tc.funcName, tc.args)
					if tc.wantErr {
						gt.Error(t, err)
						return nil
					}

					gt.NoError(t, err)
					gt.NotEqual(t, resp, nil)
					return nil
				},
			}

			gt.NoError(t, cmd.Run(t.Context(), []string{
				"bigquery",
			}))
		})
	}
}

func TestBigQuery_Enabled(t *testing.T) {
	var action bigquery.Action

	cmd := cli.Command{
		Name:  "bigquery",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.Equal(t, action.Configure(ctx), errutil.ErrActionUnavailable)
			return nil
		},
	}

	t.Setenv("WARREN_BIGQUERY_PROJECT_ID", "")
	t.Setenv("WARREN_BIGQUERY_CREDENTIALS", "")
	gt.NoError(t, cmd.Run(t.Context(), []string{
		"bigquery",
	}))
}

func TestBigQuery_Specs(t *testing.T) {
	var action bigquery.Action
	specs, err := action.Specs(context.Background())
	gt.NoError(t, err)
	gt.A(t, specs).Length(6)

	// Check specifications of each tool
	for _, spec := range specs {
		switch spec.Name {
		case "bigquery_list_dataset":
			gt.Value(t, len(spec.Parameters)).Equal(0)
		case "bigquery_query":
			gt.Map(t, spec.Parameters).HasKey("query")
			gt.Value(t, spec.Parameters["query"].Type).Equal("string")
		case "bigquery_result":
			gt.Map(t, spec.Parameters).HasKey("query_id")
			gt.Value(t, spec.Parameters["query_id"].Type).Equal("string")
		case "bigquery_table_summary":
			gt.Map(t, spec.Parameters).HasKey("project_id")
			gt.Map(t, spec.Parameters).HasKey("dataset_id")
			gt.Map(t, spec.Parameters).HasKey("table_id")
			// All parameters are optional
			gt.False(t, spec.Parameters["project_id"].Required)
			gt.False(t, spec.Parameters["dataset_id"].Required)
			gt.False(t, spec.Parameters["table_id"].Required)
		case "bigquery_schema":
			gt.Map(t, spec.Parameters).HasKey("dataset_id")
			gt.Map(t, spec.Parameters).HasKey("table_id")
			gt.Value(t, spec.Parameters["dataset_id"].Type).Equal("string")
			gt.Value(t, spec.Parameters["table_id"].Type).Equal("string")
		case "bigquery_runbook_search":
			gt.Map(t, spec.Parameters).HasKey("query")
			gt.Map(t, spec.Parameters).HasKey("limit")
			gt.Value(t, spec.Parameters["query"].Type).Equal("string")
			gt.Value(t, spec.Parameters["limit"].Type).Equal("integer")
			gt.True(t, spec.Parameters["query"].Required)
		}
	}
}

func TestBigQuery_Query(t *testing.T) {
	var action bigquery.Action

	projectID := os.Getenv("TEST_BIGQUERY_PROJECT_ID")
	query := os.Getenv("TEST_BIGQUERY_QUERY")
	storageBucket := os.Getenv("TEST_BIGQUERY_STORAGE_BUCKET")
	storagePrefix := os.Getenv("TEST_BIGQUERY_STORAGE_PREFIX")

	if projectID == "" || query == "" || storageBucket == "" {
		t.Skip("TEST_BIGQUERY_PROJECT_ID, TEST_BIGQUERY_QUERY, TEST_BIGQUERY_STORAGE_BUCKET are not set")
	}

	cmd := cli.Command{
		Name:  "bigquery",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))

			// Execute query
			runResult, err := action.Run(ctx, "bigquery_query", map[string]any{"query": query})
			gt.NoError(t, err)
			gt.NotEqual(t, runResult, nil)
			gt.Map(t, runResult).HasKey("query_id").Required()
			queryID := gt.Cast[string](t, runResult["query_id"])
			gt.NotEqual(t, queryID, "")

			// Wait for a moment to ensure the job is completed
			time.Sleep(2 * time.Second)

			// Get query results
			runResult, err = action.Run(ctx, "bigquery_result", map[string]any{
				"query_id": queryID,
				"limit":    100.0,
				"offset":   0.0,
			})
			gt.NoError(t, err)
			gt.NotEqual(t, runResult, nil)
			gt.Map(t, runResult).HasKey("rows_json").Required()
			gt.Map(t, runResult).HasKey("total_rows").Required()
			gt.Map(t, runResult).HasKey("total_size").Required()
			gt.Map(t, runResult).HasKey("has_more").Required()

			totalRows := gt.Cast[int](t, runResult["total_rows"])
			if totalRows > 1 {
				result1, err := action.Run(ctx, "bigquery_result", map[string]any{
					"query_id": queryID,
					"limit":    1.0,
					"offset":   0.0,
				})
				gt.NoError(t, err).Required()
				gt.NotEqual(t, result1, nil)
				// Parse rows_json to get actual rows
				rowsJSON1 := gt.Cast[string](t, result1["rows_json"])
				var rows1 []map[string]any
				err = json.Unmarshal([]byte(rowsJSON1), &rows1)
				gt.NoError(t, err)

				result2, err := action.Run(ctx, "bigquery_result", map[string]any{
					"query_id": queryID,
					"limit":    1.0,
					"offset":   0.0,
				})
				gt.NoError(t, err)
				gt.NotEqual(t, result2, nil)

				// Parse rows_json to get actual rows
				rowsJSON2 := gt.Cast[string](t, result2["rows_json"])
				var rows2 []map[string]any
				err = json.Unmarshal([]byte(rowsJSON2), &rows2)
				gt.NoError(t, err)
				gt.Equal(t, rows1, rows2)

				result3, err := action.Run(ctx, "bigquery_result", map[string]any{
					"query_id": queryID,
					"limit":    1.0,
					"offset":   1.0,
				})
				gt.NoError(t, err)
				gt.NotEqual(t, result3, nil)
				// Parse rows_json to get actual rows
				rowsJSON3 := gt.Cast[string](t, result3["rows_json"])
				var rows3 []map[string]any
				err = json.Unmarshal([]byte(rowsJSON3), &rows3)
				gt.NoError(t, err)
				gt.NotEqual(t, rows1, rows3)
			}
			return nil
		},
	}

	gt.NoError(t, cmd.Run(t.Context(), []string{
		"bigquery",
		"--bigquery-project-id", projectID,
		"--bigquery-config", "testdata/config.yml",
		"--bigquery-storage-bucket", storageBucket,
		"--bigquery-storage-prefix", storagePrefix,
	}))
}

func TestBigQuery_WithEnvVars(t *testing.T) {
	// Get values from environment variables
	projectID := os.Getenv("TEST_BIGQUERY_PROJECT_ID")
	datasetID := os.Getenv("TEST_BIGQUERY_DATASET_ID")
	tableID := os.Getenv("TEST_BIGQUERY_TABLE_ID")
	query := os.Getenv("TEST_BIGQUERY_QUERY")

	// Skip test if required environment variables are not set
	if projectID == "" || datasetID == "" || tableID == "" || query == "" {
		t.Skip("TEST_BIGQUERY_PROJECT_ID, TEST_BIGQUERY_DATASET_ID, TEST_BIGQUERY_TABLE_ID, TEST_BIGQUERY_QUERY are not set")
	}

	testCases := []struct {
		name     string
		funcName string
		args     map[string]any
		envVars  map[string]string
		wantErr  bool
	}{
		{
			name:     "list datasets with config",
			funcName: "bigquery_list_dataset",
			args:     map[string]any{},
			envVars: map[string]string{
				"WARREN_BIGQUERY_PROJECT_ID": projectID,
				"WARREN_BIGQUERY_CONFIG":     "testdata/config.yml",
			},
			wantErr: false,
		},
		{
			name:     "get schema with config",
			funcName: "bigquery_schema",
			args: map[string]any{
				"project_id": projectID,
				"dataset_id": datasetID,
				"table_id":   tableID,
			},
			envVars: map[string]string{
				"WARREN_BIGQUERY_PROJECT_ID": projectID,
				"WARREN_BIGQUERY_CONFIG":     "testdata/config.yml",
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			var action bigquery.Action
			cmd := cli.Command{
				Name:  "bigquery",
				Flags: action.Flags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					gt.NoError(t, action.Configure(ctx))

					resp, err := action.Run(ctx, tc.funcName, tc.args)
					if tc.wantErr {
						gt.Error(t, err)
						return nil
					}

					gt.NoError(t, err)
					gt.NotEqual(t, resp, nil)
					return nil
				},
			}

			gt.NoError(t, cmd.Run(context.Background(), []string{
				"bigquery",
			}))
		})
	}
}

func TestBigQuery_Prompt(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.yaml")

	// Create a test configuration file
	configContent := `dataset_id: "test_dataset"
table_id: "test_table"
description: "Test table for security events"
columns:
  - name: "timestamp"
    description: "Event timestamp"
    value_example: "2023-01-01 00:00:00"
    type: "TIMESTAMP"
  - name: "src_ip"
    description: "Source IP address"
    value_example: "192.168.1.1"
    type: "STRING"
  - name: "event_type"
    description: "Type of security event"
    value_example: "login_failure"
    type: "STRING"
partitioning:
  field: "timestamp"
  type: "time"
  time_unit: "daily"
`

	err := os.WriteFile(configFile, []byte(configContent), 0600)
	gt.NoError(t, err)

	var action bigquery.Action
	cmd := cli.Command{
		Name:  "bigquery",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))

			// Test Prompt method
			prompt, err := action.Prompt(ctx)
			gt.NoError(t, err)

			// Verify the prompt contains expected information
			gt.S(t, prompt).Contains("Available BigQuery Tables")
			gt.S(t, prompt).Contains("test_dataset")
			gt.S(t, prompt).Contains("test_table")
			gt.S(t, prompt).Contains("Test table for security events")

			// Verify detailed information is NOT included (to save tokens)
			gt.S(t, prompt).NotContains("Partitioning")
			gt.S(t, prompt).NotContains("Available Columns")
			gt.S(t, prompt).NotContains("src_ip")
			gt.S(t, prompt).NotContains("event_type")

			// Verify exploratory strategy is included
			gt.S(t, prompt).Contains("BigQuery Exploratory Investigation Strategy")
			gt.S(t, prompt).Contains("Verify Field Values")

			// Verify the helper comment is included
			gt.S(t, prompt).Contains("For detailed column information and schema, use the `bigquery_table_summary` tool")

			return nil
		},
	}

	gt.NoError(t, cmd.Run(t.Context(), []string{
		"bigquery",
		"--bigquery-project-id", "test-project",
		"--bigquery-config", configFile,
	}))
}

func TestBigQuery_PromptEmpty(t *testing.T) {
	var action bigquery.Action

	// Test Prompt method without configuration
	prompt, err := action.Prompt(context.Background())
	gt.NoError(t, err)
	gt.Value(t, prompt).Equal("")
}

func TestBigQuery_TableSummary(t *testing.T) {
	var action bigquery.Action

	cmd := cli.Command{
		Name:  "bigquery",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))

			// Test getting all table summaries
			runResult, err := action.Run(ctx, "bigquery_table_summary", map[string]any{})
			gt.NoError(t, err)
			gt.NotEqual(t, runResult, nil)
			gt.Map(t, runResult).HasKey("tables").Required()
			gt.Map(t, runResult).HasKey("total").Required()

			tables := gt.Cast[[]map[string]any](t, runResult["tables"])
			gt.Array(t, tables).Length(1) // Should have one table from testdata

			table := tables[0]
			gt.Map(t, table).HasKey("project_id").Required()
			gt.Map(t, table).HasKey("dataset_id").Required()
			gt.Map(t, table).HasKey("table_id").Required()
			gt.Map(t, table).HasKey("description").Required()
			gt.Map(t, table).HasKey("columns").Required()

			gt.Value(t, table["dataset_id"]).Equal("test_dataset")
			gt.Value(t, table["table_id"]).Equal("test_table")
			gt.Value(t, table["description"]).Equal("Test table for BigQuery actions")

			columns := gt.Cast[[]map[string]any](t, table["columns"])
			gt.Array(t, columns).Length(6) // id, timestamp, src_ip, event_type, value, metadata

			// Check column structure
			for _, col := range columns {
				gt.Map(t, col).HasKey("name").Required()
				gt.Map(t, col).HasKey("type").Required()

				if col["name"] == "metadata" {
					gt.Map(t, col).HasKey("has_nested_fields").Required()
					gt.Map(t, col).HasKey("nested_fields_count").Required()
					gt.Value(t, col["has_nested_fields"]).Equal(true)
					gt.Value(t, col["nested_fields_count"]).Equal(2)
				}
			}

			// Test filtering by dataset
			runResult, err = action.Run(ctx, "bigquery_table_summary", map[string]any{
				"dataset_id": "test_dataset",
			})
			gt.NoError(t, err)
			tables = gt.Cast[[]map[string]any](t, runResult["tables"])
			gt.Array(t, tables).Length(1)

			// Test filtering by non-existent dataset
			runResult, err = action.Run(ctx, "bigquery_table_summary", map[string]any{
				"dataset_id": "non_existent",
			})
			gt.NoError(t, err)
			tables = gt.Cast[[]map[string]any](t, runResult["tables"])
			gt.Array(t, tables).Length(0)

			// Test filtering by table
			runResult, err = action.Run(ctx, "bigquery_table_summary", map[string]any{
				"table_id": "test_table",
			})
			gt.NoError(t, err)
			tables = gt.Cast[[]map[string]any](t, runResult["tables"])
			gt.Array(t, tables).Length(1)

			return nil
		},
	}

	gt.NoError(t, cmd.Run(t.Context(), []string{
		"bigquery",
		"--bigquery-project-id", "test-project",
		"--bigquery-config", "testdata/config.yml",
	}))
}
