package bigquery_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	bq "github.com/secmon-lab/warren/pkg/tool/bigquery"
)

func TestLoadBulkConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `tables:
  - table_id: "project1.dataset1.table1"
    description: "Test table 1 description"
  - table_id: "project2.dataset2.table2" 
    description: "Test table 2 description"
`

	gt.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Test loading the config
	config, err := bq.LoadBulkConfig(configPath)
	gt.NoError(t, err)
	gt.NotNil(t, config)
	gt.Equal(t, len(config.Tables), 2)

	// Verify first table
	gt.Equal(t, config.Tables[0].TableID, "project1.dataset1.table1")
	gt.Equal(t, config.Tables[0].Description, "Test table 1 description")

	// Verify second table
	gt.Equal(t, config.Tables[1].TableID, "project2.dataset2.table2")
	gt.Equal(t, config.Tables[1].Description, "Test table 2 description")
}

func TestLoadBulkConfigFileNotFound(t *testing.T) {
	_, err := bq.LoadBulkConfig("nonexistent-file.yaml")
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("failed to read bulk config file")
}

func TestLoadBulkConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid-config.yaml")

	// Create truly invalid YAML with unmatched brackets and indentation errors
	invalidContent := `tables:
  - table_id: "project1.dataset1.table1"
    description: "Test table 1"
  - table_id: [
      invalid: yaml
    content: }
`

	gt.NoError(t, os.WriteFile(configPath, []byte(invalidContent), 0644))

	_, err := bq.LoadBulkConfig(configPath)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("failed to parse bulk config file")
}

func TestParseTableIDFromBulkConfig(t *testing.T) {
	var cfg bq.GenerateConfigConfig

	// Test valid table ID
	err := bq.ParseTableID("my-project.my_dataset.my_table", &cfg)
	gt.NoError(t, err)
	// Note: These fields are not exported, so we'll just test that parsing succeeds

	// Test invalid table ID format
	err = bq.ParseTableID("invalid.format", &cfg)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("table ID must be in format 'project_id.dataset_id.table_id'")
}
