package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/usecase"
)

func TestReadAlertData(t *testing.T) {
	t.Run("reads from stdin", func(t *testing.T) {
		// Save original stdin
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		// Create test data
		testData := map[string]any{
			"test":  "stdin data",
			"value": 456,
		}
		jsonData, err := json.Marshal(testData)
		gt.NoError(t, err)

		// Create pipe and replace stdin
		r, w, err := os.Pipe()
		gt.NoError(t, err)
		os.Stdin = r

		// Write test data to pipe in goroutine
		go func() {
			defer func() {
				_ = w.Close()
			}()
			_, _ = w.Write(jsonData)
		}()

		// Read alert data from stdin (empty string means stdin)
		result, err := cli.ReadAlertDataForTest("")
		gt.NoError(t, err)

		// Verify result
		resultMap, ok := result.(map[string]any)
		gt.True(t, ok)
		gt.Equal(t, resultMap["test"], "stdin data")
		gt.Value(t, resultMap["value"]).Equal(float64(456)) // JSON unmarshal converts numbers to float64
	})

	t.Run("reads from file", func(t *testing.T) {
		// Create temporary directory
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test-alert.json")

		// Write test data
		testData := map[string]any{
			"test":  "file data",
			"value": 789,
			"nested": map[string]any{
				"key": "nested value",
			},
		}
		jsonData, err := json.Marshal(testData)
		gt.NoError(t, err)

		err = os.WriteFile(tmpFile, jsonData, 0600)
		gt.NoError(t, err)

		// Read alert data from file
		result, err := cli.ReadAlertDataForTest(tmpFile)
		gt.NoError(t, err)

		// Verify result
		resultMap, ok := result.(map[string]any)
		gt.True(t, ok)
		gt.Equal(t, resultMap["test"], "file data")
		gt.Value(t, resultMap["value"]).Equal(float64(789))

		nestedMap, ok := resultMap["nested"].(map[string]any)
		gt.True(t, ok)
		gt.Equal(t, nestedMap["key"], "nested value")
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := cli.ReadAlertDataForTest("/non/existent/file.json")
		gt.Error(t, err)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "invalid.json")

		err := os.WriteFile(tmpFile, []byte("{invalid json}"), 0600)
		gt.NoError(t, err)

		_, err = cli.ReadAlertDataForTest(tmpFile)
		gt.Error(t, err)
	})
}

func TestDisplayPipelineResult(t *testing.T) {
	t.Run("displays empty result", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		defer func() { os.Stdout = oldStdout }()

		r, w, err := os.Pipe()
		gt.NoError(t, err)
		os.Stdout = w

		// Display empty result
		results := []*usecase.AlertPipelineResult{}

		err = cli.DisplayPipelineResultForTest(results)
		gt.NoError(t, err)

		// Close writer and read output
		_ = w.Close()
		var buf bytes.Buffer
		_, err = buf.ReadFrom(r)
		gt.NoError(t, err)

		// Verify output contains "No alerts generated"
		output := buf.String()
		gt.True(t, len(output) > 0)
		gt.True(t, output == "No alerts generated\n")
	})

	t.Run("displays result with alert data", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		defer func() { os.Stdout = oldStdout }()

		r, w, err := os.Pipe()
		gt.NoError(t, err)
		os.Stdout = w

		// Create test result
		testAlert := &alert.Alert{
			Schema: types.AlertSchema("test"),
			Data: map[string]any{
				"test": "alert data",
			},
			Metadata: alert.Metadata{
				Title:       "Test Alert",
				Description: "Test Description",
				TitleSource: types.SourcePolicy,
			},
		}

		enrichResults := policy.EnrichResults{
			"task1": map[string]any{
				"result": "enriched data",
			},
		}

		commitResult := &policy.CommitPolicyResult{
			Title:       "Updated Title",
			Description: "Updated Description",
			Channel:     "test-channel",
		}

		results := []*usecase.AlertPipelineResult{
			{
				Alert:        testAlert,
				EnrichResult: enrichResults,
				CommitResult: commitResult,
			},
		}

		err = cli.DisplayPipelineResultForTest(results)
		gt.NoError(t, err)

		// Close writer and read output
		_ = w.Close()
		var buf bytes.Buffer
		_, err = buf.ReadFrom(r)
		gt.NoError(t, err)

		// Verify human-readable output
		output := buf.String()
		gt.True(t, len(output) > 0)

		// Check for key sections
		gt.True(t, strings.Contains(output, "📋 ALERT"))
		gt.True(t, strings.Contains(output, "✅ COMMIT RESULT"))

		// Check for alert data
		gt.True(t, strings.Contains(output, "Schema:      test"))
		gt.True(t, strings.Contains(output, "Title:       Test Alert"))
		gt.True(t, strings.Contains(output, "Description: Test Description"))

		// Check for commit result
		gt.True(t, strings.Contains(output, "Title:       Updated Title"))
		gt.True(t, strings.Contains(output, "Description: Updated Description"))
		gt.True(t, strings.Contains(output, "Channel:     test-channel"))
	})

	t.Run("displays result with multiple alerts", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		defer func() { os.Stdout = oldStdout }()

		r, w, err := os.Pipe()
		gt.NoError(t, err)
		os.Stdout = w

		// Create test results with multiple alerts
		results := []*usecase.AlertPipelineResult{
			{
				Alert: &alert.Alert{
					Schema: types.AlertSchema("test"),
					Metadata: alert.Metadata{
						Title: "Alert 1",
					},
				},
				EnrichResult: policy.EnrichResults{},
				CommitResult: &policy.CommitPolicyResult{},
			},
			{
				Alert: &alert.Alert{
					Schema: types.AlertSchema("test"),
					Metadata: alert.Metadata{
						Title: "Alert 2",
					},
				},
				EnrichResult: policy.EnrichResults{},
				CommitResult: &policy.CommitPolicyResult{},
			},
		}

		err = cli.DisplayPipelineResultForTest(results)
		gt.NoError(t, err)

		// Close writer and read output
		_ = w.Close()
		var buf bytes.Buffer
		_, err = buf.ReadFrom(r)
		gt.NoError(t, err)

		// Verify human-readable output
		output := buf.String()
		gt.True(t, len(output) > 0)

		// Check for separator between alerts
		gt.True(t, strings.Contains(output, "═══════════════════════════════════════════════════════════════════════════"))

		// Check for both alerts
		gt.True(t, strings.Contains(output, "Title:       Alert 1"))
		gt.True(t, strings.Contains(output, "Title:       Alert 2"))
	})
}
