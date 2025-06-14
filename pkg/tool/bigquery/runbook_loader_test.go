package bigquery

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/bigquery"
)

func TestRunbookLoader_LoadFromFile(t *testing.T) {
	ctx := context.Background()

	// Create a temporary SQL file with metadata
	tmpDir := t.TempDir()
	sqlFile := filepath.Join(tmpDir, "test_query.sql")

	sqlContent := `-- Title: Find suspicious login activities
-- Description: This query identifies login activities from unusual locations
-- that might indicate potential security threats or compromised accounts.
-- It looks for IP addresses that have never been seen before for a user.

SELECT 
    user_id,
    ip_address,
    login_time,
    COUNT(*) as login_count
FROM security_logs.user_logins
WHERE DATE(login_time) >= DATE_SUB(CURRENT_DATE(), INTERVAL 7 DAY)
    AND ip_address NOT IN (
        SELECT DISTINCT ip_address 
        FROM security_logs.user_logins 
        WHERE DATE(login_time) < DATE_SUB(CURRENT_DATE(), INTERVAL 30 DAY)
    )
GROUP BY user_id, ip_address, DATE(login_time)
ORDER BY login_time DESC
LIMIT 100;`

	err := os.WriteFile(sqlFile, []byte(sqlContent), 0644)
	gt.NoError(t, err)

	// Create loader and load the file
	loader := NewRunbookLoader([]string{sqlFile})
	entries, err := loader.LoadRunbooks(ctx)
	gt.NoError(t, err)

	// Verify results
	gt.Array(t, entries).Length(1)

	entry := entries[0]
	gt.Value(t, entry.Title).Equal("Find suspicious login activities")
	gt.S(t, entry.Description).Contains("This query identifies login activities from unusual locations")
	gt.S(t, entry.SQLContent).Contains("SELECT")
	gt.S(t, entry.SQLContent).Contains("security_logs.user_logins")
}

func TestRunbookLoader_LoadFromDirectory(t *testing.T) {
	ctx := context.Background()

	// Create a temporary directory with multiple SQL files
	tmpDir := t.TempDir()

	// Create first SQL file
	sqlFile1 := filepath.Join(tmpDir, "query1.sql")
	sqlContent1 := `-- Title: Count total users
-- Description: Simple query to count all users
SELECT COUNT(*) FROM users;`
	err := os.WriteFile(sqlFile1, []byte(sqlContent1), 0644)
	gt.NoError(t, err)

	// Create second SQL file
	sqlFile2 := filepath.Join(tmpDir, "query2.sql")
	sqlContent2 := `-- Title: Recent activities
-- Description: Get recent user activities
SELECT * FROM activities WHERE created_at >= CURRENT_DATE() - INTERVAL 1 DAY;`
	err = os.WriteFile(sqlFile2, []byte(sqlContent2), 0644)
	gt.NoError(t, err)

	// Create a non-SQL file (should be ignored)
	txtFile := filepath.Join(tmpDir, "readme.txt")
	err = os.WriteFile(txtFile, []byte("This is not a SQL file"), 0644)
	gt.NoError(t, err)

	// Create loader and load the directory
	loader := NewRunbookLoader([]string{tmpDir})
	entries, err := loader.LoadRunbooks(ctx)
	gt.NoError(t, err)

	// Verify results - should have 2 SQL files, ignoring the txt file
	gt.Array(t, entries).Length(2)

	// Verify entries (order might vary)
	titles := make(map[string]bool)
	for _, entry := range entries {
		titles[entry.Title] = true
		gt.Value(t, entry.SQLContent).NotEqual("")
	}

	gt.Value(t, titles["Count total users"]).Equal(true)
	gt.Value(t, titles["Recent activities"]).Equal(true)
}

func TestRunbookLoader_ExtractTitleAndDescription(t *testing.T) {
	loader := &RunbookLoader{}

	testCases := []struct {
		name        string
		content     string
		expectTitle string
		expectDesc  string
	}{
		{
			name: "full metadata",
			content: `-- Title: Test Query
-- Description: This is a test query that does something
-- useful for security analysis
SELECT * FROM test_table;`,
			expectTitle: "Test Query",
			expectDesc:  "This is a test query that does something useful for security analysis",
		},
		{
			name: "title only",
			content: `-- Title: Simple Query
SELECT COUNT(*) FROM users;`,
			expectTitle: "Simple Query",
			expectDesc:  "",
		},
		{
			name: "description only",
			content: `-- Description: Query without title
SELECT * FROM logs;`,
			expectTitle: "",
			expectDesc:  "Query without title",
		},
		{
			name: "no metadata",
			content: `-- Just a comment
SELECT * FROM table;`,
			expectTitle: "",
			expectDesc:  "",
		},
		{
			name: "case insensitive",
			content: `-- title: Lowercase Title
-- description: lowercase description
SELECT 1;`,
			expectTitle: "Lowercase Title",
			expectDesc:  "lowercase description",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			title, desc := loader.extractTitleAndDescription(tc.content)
			gt.Value(t, title).Equal(tc.expectTitle)
			gt.Value(t, desc).Equal(tc.expectDesc)
		})
	}
}

func TestLoadRunbooks_Integration(t *testing.T) {
	ctx := context.Background()

	// Create a temporary directory with SQL files
	tempDir := t.TempDir()

	// Create sample SQL files
	sqlFiles := map[string]string{
		"user_activity.sql": `-- Title: User Activity Analysis
-- Description: Analyze user activity patterns for security monitoring

SELECT 
    user_id,
    COUNT(*) as activity_count,
    MIN(timestamp) as first_activity,
    MAX(timestamp) as last_activity
FROM security_logs
WHERE timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 DAY)
GROUP BY user_id
ORDER BY activity_count DESC;`,

		"failed_logins.sql": `-- Title: Failed Login Detection
-- Description: Detect patterns of failed login attempts that might indicate brute force attacks

SELECT 
    source_ip,
    user_id,
    COUNT(*) as failed_attempts,
    MIN(timestamp) as first_attempt,
    MAX(timestamp) as last_attempt
FROM auth_logs
WHERE status = 'FAILED'
  AND timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 HOUR)
GROUP BY source_ip, user_id
HAVING COUNT(*) >= 5
ORDER BY failed_attempts DESC;`,

		"network_anomaly.sql": `-- Title: Network Traffic Anomaly Detection
-- Description: Identify unusual network traffic patterns that could indicate malicious activity

WITH traffic_stats AS (
  SELECT 
    source_ip,
    destination_ip,
    COUNT(*) as connection_count,
    SUM(bytes_transferred) as total_bytes
  FROM network_logs
  WHERE timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 HOUR)
  GROUP BY source_ip, destination_ip
)
SELECT *
FROM traffic_stats
WHERE connection_count > 1000 OR total_bytes > 1000000000;`,
	}

	for filename, content := range sqlFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		gt.NoError(t, err)
	}

	// Create runbook loader
	loader := NewRunbookLoader([]string{tempDir})

	// Load runbooks
	entries, err := loader.LoadRunbooks(ctx)
	gt.NoError(t, err)
	gt.Array(t, entries).Length(3)

	// Verify that all entries have correct metadata
	titleMap := make(map[string]*bigquery.RunbookEntry)
	for _, entry := range entries {
		titleMap[entry.Title] = entry
	}

	// Check user activity entry
	userActivityEntry := titleMap["User Activity Analysis"]
	gt.NotEqual(t, userActivityEntry, nil)
	gt.S(t, userActivityEntry.Description).Contains("security monitoring")
	gt.S(t, userActivityEntry.SQLContent).Contains("FROM security_logs")

	// Check failed logins entry
	failedLoginsEntry := titleMap["Failed Login Detection"]
	gt.NotEqual(t, failedLoginsEntry, nil)
	gt.S(t, failedLoginsEntry.Description).Contains("brute force attacks")
	gt.S(t, failedLoginsEntry.SQLContent).Contains("WHERE status = 'FAILED'")

	// Check network anomaly entry
	networkAnomalyEntry := titleMap["Network Traffic Anomaly Detection"]
	gt.NotEqual(t, networkAnomalyEntry, nil)
	gt.S(t, networkAnomalyEntry.Description).Contains("malicious activity")
	gt.S(t, networkAnomalyEntry.SQLContent).Contains("network_logs")

	// Verify all entries have unique IDs
	idSet := make(map[string]bool)
	for _, entry := range entries {
		gt.Value(t, idSet[entry.ID.String()]).Equal(false) // Should not be duplicate
		idSet[entry.ID.String()] = true
		gt.S(t, entry.ID.String()).NotEqual("") // ID should not be empty
	}
}
