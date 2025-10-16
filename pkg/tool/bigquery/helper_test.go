package bigquery_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/gt"
	bq "github.com/secmon-lab/warren/pkg/tool/bigquery"
	"gopkg.in/yaml.v3"
)

func TestParseTableID(t *testing.T) {
	tests := []struct {
		name      string
		tableID   string
		wantError bool
		wantProj  string
		wantDS    string
		wantTable string
	}{
		{
			name:      "valid table ID",
			tableID:   "my-project.my_dataset.my_table",
			wantError: false,
			wantProj:  "my-project",
			wantDS:    "my_dataset",
			wantTable: "my_table",
		},
		{
			name:      "invalid format - too few parts",
			tableID:   "my-project.my_dataset",
			wantError: true,
		},
		{
			name:      "invalid format - too many parts",
			tableID:   "my-project.my_dataset.my_table.extra",
			wantError: true,
		},
		{
			name:      "empty string",
			tableID:   "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg bq.GenerateConfigInput
			err := bq.ParseTableID(tt.tableID, &cfg)

			if tt.wantError {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
				gt.Equal(t, cfg.TableProjectID, tt.wantProj)
				gt.Equal(t, cfg.TableDatasetID, tt.wantDS)
				gt.Equal(t, cfg.TableTableID, tt.wantTable)
			}
		})
	}
}

func TestGenerateOutputPath(t *testing.T) {
	tests := []struct {
		name      string
		cfg       bq.GenerateConfigInput
		wantPath  string
		wantError bool
		contains  string
	}{
		{
			name: "default template",
			cfg: bq.GenerateConfigInput{
				TableProjectID: "proj",
				TableDatasetID: "dataset",
				TableTableID:   "table",
				OutputDir:      "/tmp",
				OutputFile:     "{{ .project_id }}.{{ .dataset_id }}.{{ .table_id }}.yaml",
			},
			wantPath: "/tmp/proj.dataset.table.yaml",
		},
		{
			name: "custom filename",
			cfg: bq.GenerateConfigInput{
				TableProjectID: "proj",
				TableDatasetID: "dataset",
				TableTableID:   "table",
				OutputDir:      "/tmp",
				OutputFile:     "custom.yaml",
			},
			wantPath: "/tmp/custom.yaml",
		},
		{
			name: "empty output file",
			cfg: bq.GenerateConfigInput{
				TableProjectID: "proj",
				TableDatasetID: "dataset",
				TableTableID:   "table",
				OutputDir:      "/tmp",
				OutputFile:     "",
			},
			wantPath: "/tmp/proj.dataset.table.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := bq.GenerateOutputPath(tt.cfg)

			if tt.wantError {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
				gt.Equal(t, path, tt.wantPath)
			}
		})
	}
}

func TestLoadBulkConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `tables:
  - table_id: project.dataset.table1
    description: First table
  - table_id: project.dataset.table2
    description: Second table
`
		err := os.WriteFile(configPath, []byte(configContent), 0600)
		gt.NoError(t, err)

		config, err := bq.LoadBulkConfig(configPath)
		gt.NoError(t, err)
		gt.A(t, config.Tables).Length(2)
		gt.Equal(t, config.Tables[0].TableID, "project.dataset.table1")
		gt.Equal(t, config.Tables[0].Description, "First table")
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := bq.LoadBulkConfig("/nonexistent/config.yaml")
		gt.Error(t, err)
	})

	t.Run("invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0600)
		gt.NoError(t, err)

		_, err = bq.LoadBulkConfig(configPath)
		gt.Error(t, err)
	})
}

func TestGenerateConfigWithActivitySchema(t *testing.T) {
	if os.Getenv("TEST_BIGQUERY_TOOL_HELPER") == "" {
		t.Skip("TEST_BIGQUERY_TOOL_HELPER not set, skipping integration test")
	}

	ctx := context.Background()

	// Load activity schema
	// Create simple test schema for testing
	simpleSchema := bigquery.Schema{
		{Name: "id", Type: bigquery.StringFieldType, Description: "Event ID"},
		{Name: "name", Type: bigquery.StringFieldType, Description: "Event name"},
		{Name: "timestamp", Type: bigquery.TimestampFieldType, Description: "Event timestamp"},
	}

	// Create mock BigQuery client
	mockClient := bq.NewMockBigQueryClient()
	mockClient.TableMetadata["test.activity"] = &bigquery.TableMetadata{
		Schema: simpleSchema,
	}

	// Mock some query results
	mockClient.QueryResults["SELECT * FROM `test.dataset.activity` LIMIT 10"] = []map[string]any{
		{"timestamp": "2024-01-01T00:00:00Z", "event_name": "test_event"},
	}

	// Create mock factory
	factory := &bq.MockBigQueryClientFactory{
		Client: mockClient,
	}

	// Create temp output directory
	tmpDir := t.TempDir()

	// Get GCP project ID from environment variable
	geminiProjectID := os.Getenv("TEST_BIGQUERY_TOOL_HELPER_PROJECT")
	if geminiProjectID == "" {
		t.Skip("TEST_BIGQUERY_TOOL_HELPER_PROJECT not set, skipping integration test")
	}

	// Setup config - use actual Gemini project
	cfg := bq.GenerateConfigInput{
		GeminiProjectID:  geminiProjectID,
		GeminiLocation:   "us-central1",
		BQProjectID:      "test-project",
		TableProjectID:   "test",
		TableDatasetID:   "dataset",
		TableTableID:     "activity",
		TableDescription: "Activity log table with user events and authentication data",
		ScanLimit:        1024 * 1024 * 1024, // 1GB
		ScanLimitStr:     "1GB",
		OutputDir:        tmpDir,
		OutputFile:       "activity.yaml",
	}

	// Generate config using actual Gemini LLM
	err := bq.GenerateConfigWithFactory(ctx, cfg, factory)
	gt.NoError(t, err)

	// Verify output file
	outputPath := filepath.Join(tmpDir, "activity.yaml")
	_, err = os.Stat(outputPath)
	gt.NoError(t, err)

	// Parse and validate YAML
	yamlData, err := os.ReadFile(outputPath)
	gt.NoError(t, err)

	var config bq.Config
	err = yaml.Unmarshal(yamlData, &config)
	gt.NoError(t, err)

	// Verify basic structure
	gt.Equal(t, config.DatasetID, "dataset")
	gt.Equal(t, config.TableID, "activity")
	gt.True(t, len(config.Description) > 0)
	gt.True(t, len(config.Columns) > 0)

	// Verify columns have required fields
	for _, col := range config.Columns {
		gt.True(t, len(col.Name) > 0)
		gt.True(t, len(col.Type) > 0)
		gt.True(t, len(col.Description) > 0)

		// Verify field names exist in schema
		found := false
		for _, schemaField := range simpleSchema {
			if schemaField.Name == col.Name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("column %s not found in schema", col.Name)
		}

		// Verify RECORD types have nested fields if applicable
		if col.Type == "RECORD" {
			for _, schemaField := range simpleSchema {
				if schemaField.Name == col.Name && len(schemaField.Schema) > 0 {
					if len(col.Fields) == 0 {
						t.Errorf("RECORD column %s should have nested fields", col.Name)
					}
				}
			}
		}
	}

	t.Logf("Generated config with %d columns", len(config.Columns))
	t.Logf("Output YAML:\n%s", string(yamlData))
}
