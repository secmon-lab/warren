package bigquery_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	extbq "github.com/gollem-dev/tools/bigquery"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/bigquery"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

// TestBigQueryOptionsAppended verifies that each optional flag's Action appends
// the matching external option carrying the provided value, by applying the
// accumulated options to a fresh external ToolSet and reading its unexported
// fields.
func TestBigQueryOptionsAppended(t *testing.T) {
	var action bigquery.Action
	cmd := cli.Command{
		Name:   "bigquery",
		Flags:  action.Flags(),
		Action: func(context.Context, *cli.Command) error { return nil },
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{
		"bigquery",
		"--bigquery-project-id", "proj",
		"--bigquery-credentials", "/tmp/creds.json",
		"--bigquery-impersonate-service-account", "sa@example.iam.gserviceaccount.com",
		"--bigquery-config", "/cfg/a.yaml",
		"--bigquery-config", "/cfg/b.yaml",
		"--bigquery-runbook-path", "/rb",
		"--bigquery-storage-bucket", "my-bucket",
		"--bigquery-storage-prefix", "results/",
		"--bigquery-timeout", "3m",
		"--bigquery-scan-limit", "5GB",
	}))

	var ts extbq.ToolSet
	for _, o := range action.Opts() {
		o(&ts)
	}
	gt.Value(t, test.PrivateField(t, &ts, "credentials")).Equal("/tmp/creds.json")
	gt.Value(t, test.PrivateField(t, &ts, "impersonateServiceAccount")).Equal("sa@example.iam.gserviceaccount.com")
	gt.Value(t, test.PrivateField(t, &ts, "configFiles")).Equal([]string{"/cfg/a.yaml", "/cfg/b.yaml"})
	gt.Value(t, test.PrivateField(t, &ts, "runbookPaths")).Equal([]string{"/rb"})
	gt.Value(t, test.PrivateField(t, &ts, "storageBucket")).Equal("my-bucket")
	gt.Value(t, test.PrivateField(t, &ts, "storagePrefix")).Equal("results/")
	gt.Value(t, test.PrivateField(t, &ts, "timeout")).Equal(3 * time.Minute)
	gt.Value(t, test.PrivateField(t, &ts, "scanLimitStr")).Equal("5GB")
}

// runConfigure parses the given CLI args into a fresh Action and runs Configure.
func runConfigure(t *testing.T, args []string, fn func(ctx context.Context, action *bigquery.Action)) {
	t.Helper()
	var action bigquery.Action
	cmd := cli.Command{
		Name:  "bigquery",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			fn(ctx, &action)
			return nil
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), append([]string{"bigquery"}, args...)))
}

func TestBigQueryUnavailableWithoutProject(t *testing.T) {
	t.Setenv("WARREN_BIGQUERY_PROJECT_ID", "")
	t.Setenv("WARREN_BIGQUERY_CREDENTIALS", "")
	runConfigure(t, nil, func(ctx context.Context, action *bigquery.Action) {
		gt.Equal(t, action.Configure(ctx), errutil.ErrActionUnavailable)
	})
}

func TestBigQueryUnavailableWithoutConfig(t *testing.T) {
	runConfigure(t, []string{"--bigquery-project-id", "test-project"}, func(ctx context.Context, action *bigquery.Action) {
		gt.Equal(t, action.Configure(ctx), errutil.ErrActionUnavailable)
	})
}

func writeBQConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `dataset_id: "test_dataset"
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
partitioning:
  field: "timestamp"
  type: "time"
  time_unit: "daily"
`
	gt.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func TestBigQuerySpecsDelegation(t *testing.T) {
	configFile := writeBQConfig(t)
	runConfigure(t, []string{"--bigquery-project-id", "test-project", "--bigquery-config", configFile},
		func(ctx context.Context, action *bigquery.Action) {
			gt.NoError(t, action.Configure(ctx))
			specs, err := action.Specs(ctx)
			gt.NoError(t, err)
			gt.A(t, specs).Length(6)

			names := map[string]bool{}
			for _, s := range specs {
				names[s.Name] = true
			}
			for _, want := range []string{
				"bigquery_list_dataset",
				"bigquery_query",
				"bigquery_result",
				"bigquery_table_summary",
				"bigquery_schema",
				"get_runbook_entry",
			} {
				gt.Value(t, names[want]).Equal(true)
			}
		})
}

func TestBigQuerySpecsBeforeConfigure(t *testing.T) {
	var action bigquery.Action
	_, err := action.Specs(context.Background())
	gt.Error(t, err)
}

func TestBigQueryPrompt(t *testing.T) {
	configFile := writeBQConfig(t)
	runConfigure(t, []string{"--bigquery-project-id", "test-project", "--bigquery-config", configFile},
		func(ctx context.Context, action *bigquery.Action) {
			gt.NoError(t, action.Configure(ctx))

			prompt, err := action.Prompt(ctx)
			gt.NoError(t, err)

			gt.S(t, prompt).Contains("Available BigQuery Tables")
			gt.S(t, prompt).Contains("test_dataset")
			gt.S(t, prompt).Contains("test_table")
			gt.S(t, prompt).Contains("Test table for security events")

			// Detailed column information is intentionally omitted to save tokens.
			gt.S(t, prompt).NotContains("src_ip")

			gt.S(t, prompt).Contains("For detailed column information and schema, use the `bigquery_table_summary` tool")
		})
}

func TestBigQueryPromptEmpty(t *testing.T) {
	var action bigquery.Action
	prompt, err := action.Prompt(context.Background())
	gt.NoError(t, err)
	gt.Value(t, prompt).Equal("")
}
