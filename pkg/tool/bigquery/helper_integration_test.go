package bigquery

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/gt"
	"gopkg.in/yaml.v3"
)

func TestGenerateConfigTool_BigQueryQuery(t *testing.T) {
	ctx := context.Background()

	// Create mock BigQuery client
	mockClient := newMockBigQueryClient()

	// Set up dry run result (under limit)
	mockClient.DryRunResults["SELECT COUNT(*) FROM test.table"] = &bigquery.JobStatistics{
		TotalBytesProcessed: 1000, // 1KB
	}

	// Set up query results
	mockClient.QueryResults["SELECT COUNT(*) FROM test.table"] = []map[string]any{
		{"count": int64(12345)},
	}

	tool := &generateConfigTool{
		scanLimitStr:   "1GB",
		scanLimit:      1024 * 1024 * 1024,
		bigqueryClient: mockClient,
		outputPath:     "/tmp/test.yaml",
	}

	// Test bigquery_query tool
	result, err := tool.Run(ctx, "bigquery_query", map[string]any{
		"query": "SELECT COUNT(*) FROM test.table",
	})

	gt.NoError(t, err)
	gt.NotNil(t, result["query_id"])
	gt.Equal(t, result["total_rows"], 1)

	// Verify query was executed (dry run + actual execution)
	gt.A(t, mockClient.ExecutedQueries).Length(2)
	gt.Equal(t, mockClient.ExecutedQueries[0], "SELECT COUNT(*) FROM test.table")
	gt.Equal(t, mockClient.ExecutedQueries[1], "SELECT COUNT(*) FROM test.table")
}

func TestGenerateConfigTool_ScanLimitExceeded(t *testing.T) {
	ctx := context.Background()

	// Create mock BigQuery client
	mockClient := newMockBigQueryClient()

	// Set up dry run result (exceeds limit)
	mockClient.DryRunResults["SELECT * FROM large_table"] = &bigquery.JobStatistics{
		TotalBytesProcessed: 2 * 1024 * 1024 * 1024, // 2GB
	}

	tool := &generateConfigTool{
		scanLimitStr:   "1GB",
		scanLimit:      1024 * 1024 * 1024, // 1GB limit
		bigqueryClient: mockClient,
		outputPath:     "/tmp/test.yaml",
	}

	// Test bigquery_query tool with scan limit exceeded
	_, err := tool.Run(ctx, "bigquery_query", map[string]any{
		"query": "SELECT * FROM large_table",
	})

	gt.Error(t, err)
	gt.True(t, err.Error() != "")
}

func TestGenerateConfigTool_ConfigOutput(t *testing.T) {
	ctx := context.Background()

	// Create temporary output file
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_config.yaml")

	// Create mock BigQuery client
	mockClient := newMockBigQueryClient()

	// Set up proper table metadata with the fields we're testing
	mockClient.TableMetadata["security.auth_logs"] = &bigquery.TableMetadata{
		Schema: bigquery.Schema{
			{Name: "timestamp", Type: bigquery.TimestampFieldType, Description: "Event timestamp"},
			{Name: "user_id", Type: bigquery.StringFieldType, Description: "User identifier"},
		},
	}

	tool := &generateConfigTool{
		scanLimitStr:   "1GB",
		scanLimit:      1024 * 1024 * 1024,
		bigqueryClient: mockClient,
		outputPath:     outputPath,
		tableDatasetID: "security",
		tableTableID:   "auth_logs",
	}

	// Test generate_config_output tool call
	result, err := tool.Run(ctx, "generate_config_output", map[string]any{
		"config": map[string]any{
			"dataset_id":  "security",
			"table_id":    "auth_logs",
			"description": "Authentication and authorization events for security monitoring",
			"columns": []map[string]any{
				{
					"name":          "timestamp",
					"type":          "TIMESTAMP",
					"description":   "Event timestamp for temporal analysis and correlation",
					"value_example": "2024-01-01T00:00:00Z",
				},
				{
					"name":          "user_id",
					"type":          "STRING",
					"description":   "User identifier for tracking user activities and anomalies",
					"value_example": "user123",
				},
			},
		},
	})

	// The tool returns gollem.ErrExitConversation on success, which is expected
	gt.True(t, err != nil && err.Error() == "exit conversation")
	gt.Equal(t, result["status"], "success")

	// Verify YAML file was created
	_, err = os.Stat(outputPath)
	gt.NoError(t, err)

	// Read and verify YAML content
	yamlData, err := os.ReadFile(outputPath)
	gt.NoError(t, err)

	var config Config
	err = yaml.Unmarshal(yamlData, &config)
	gt.NoError(t, err)

	gt.Equal(t, config.DatasetID, "security")
	gt.Equal(t, config.TableID, "auth_logs")
	gt.Equal(t, config.Description, "Authentication and authorization events for security monitoring")
	gt.A(t, config.Columns).Length(2)
	gt.Equal(t, config.Columns[0].Name, "timestamp")
	gt.Equal(t, config.Columns[0].Type, "TIMESTAMP")
	gt.Equal(t, config.Columns[1].Name, "user_id")
	gt.Equal(t, config.Columns[1].Type, "STRING")
}
