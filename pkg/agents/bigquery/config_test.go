package bigquery_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/agents/bigquery"
)

func TestLoadConfigWithRunbooks(t *testing.T) {
	ctx := context.Background()
	configPath := filepath.Join("testdata", "config_with_runbooks.yaml")
	runbookDir := filepath.Join("testdata", "runbooks")

	// Load config with runbooks
	cfg, err := bigquery.LoadConfigWithRunbooks(ctx, configPath, []string{runbookDir})
	gt.NoError(t, err)
	gt.V(t, cfg).NotNil()

	// Verify basic config loaded
	gt.V(t, len(cfg.Tables)).Equal(1)
	gt.V(t, cfg.Tables[0].TableID).Equal("test_table")

	// Verify runbooks loaded
	gt.V(t, len(cfg.Runbooks) > 0).Equal(true)

	// Verify runbook contents
	var foundFailedLogins, foundSuspiciousAccess bool
	for _, entry := range cfg.Runbooks {
		if entry.Title == "Failed Login Investigation" {
			foundFailedLogins = true
			gt.V(t, strings.Contains(entry.Description, "failed login attempts")).Equal(true)
			gt.V(t, strings.Contains(entry.SQLContent, "login_failed")).Equal(true)
		}
		if entry.Title == "Suspicious Access Pattern" {
			foundSuspiciousAccess = true
			gt.V(t, strings.Contains(entry.Description, "unusual access patterns")).Equal(true)
			gt.V(t, strings.Contains(entry.SQLContent, "access_logs")).Equal(true)
		}
	}

	gt.V(t, foundFailedLogins).Equal(true)
	gt.V(t, foundSuspiciousAccess).Equal(true)
}

func TestLoadConfigWithAdditionalRunbookPaths(t *testing.T) {
	ctx := context.Background()
	configPath := filepath.Join("testdata", "config_with_runbooks.yaml")

	// Create a temp runbook file
	tmpDir := t.TempDir()
	tmpRunbook := filepath.Join(tmpDir, "temp_runbook.sql")
	content := `-- Title: Temp Runbook
-- Description: Temporary runbook for testing
SELECT * FROM test
`
	err := os.WriteFile(tmpRunbook, []byte(content), 0644)
	gt.NoError(t, err)

	// Load config with additional paths
	cfg, err := bigquery.LoadConfigWithRunbooks(ctx, configPath, []string{tmpDir})
	gt.NoError(t, err)

	// Verify additional runbook loaded
	var foundTempRunbook bool
	for _, entry := range cfg.Runbooks {
		if entry.Title == "Temp Runbook" {
			foundTempRunbook = true
			break
		}
	}
	gt.V(t, foundTempRunbook).Equal(true)
}

func TestLoadConfigWithoutRunbooks(t *testing.T) {
	ctx := context.Background()

	// Create a temp config without runbooks
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config_no_runbooks.yaml")
	configContent := `tables:
  - project_id: test-project
    dataset_id: test_dataset
    table_id: test_table
    description: Test table

scan_size_limit: "1GB"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	gt.NoError(t, err)

	// Load config without runbooks
	cfg, err := bigquery.LoadConfigWithRunbooks(ctx, configPath, nil)
	gt.NoError(t, err)
	gt.V(t, cfg).NotNil()
	gt.V(t, len(cfg.Runbooks)).Equal(0)
}

func TestRunbookIDCollisionHandling(t *testing.T) {
	ctx := context.Background()
	configPath := filepath.Join("testdata", "config_with_runbooks.yaml")
	runbookDir := filepath.Join("testdata", "runbooks")

	// Load config with runbooks (including subdirectory)
	cfg, err := bigquery.LoadConfigWithRunbooks(ctx, configPath, []string{runbookDir})
	gt.NoError(t, err)

	// Verify we have runbooks from both root and subdirectory
	// IDs are UUIDs, so we search by title
	var hasRootFailedLogins, hasSubdirFailedLogins bool
	for _, entry := range cfg.Runbooks {
		t.Logf("Runbook ID: %s, Title: %s", entry.ID, entry.Title)
		if entry.Title == "Failed Login Investigation" {
			hasRootFailedLogins = true
		}
		if entry.Title == "Failed Login in Subdirectory" {
			hasSubdirFailedLogins = true
		}
	}

	// Both should exist with different UUIDs
	gt.V(t, hasRootFailedLogins).Equal(true)
	gt.V(t, hasSubdirFailedLogins).Equal(true)
}
