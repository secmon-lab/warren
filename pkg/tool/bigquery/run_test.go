package bigquery_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/tool/bigquery"
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
			gt.Equal(t, action.Configure(ctx), errs.ErrActionUnavailable)
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
	gt.A(t, specs).Length(4)

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
		case "bigquery_schema":
			gt.Map(t, spec.Parameters).HasKey("dataset_id")
			gt.Map(t, spec.Parameters).HasKey("table_id")
			gt.Value(t, spec.Parameters["dataset_id"].Type).Equal("string")
			gt.Value(t, spec.Parameters["table_id"].Type).Equal("string")
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
			gt.Map(t, runResult).HasKey("rows").Required()
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
				rows1 := gt.Cast[[]map[string]any](t, result1["rows"])

				result2, err := action.Run(ctx, "bigquery_result", map[string]any{
					"query_id": queryID,
					"limit":    1.0,
					"offset":   0.0,
				})
				gt.NoError(t, err)
				gt.NotEqual(t, result2, nil)

				rows2 := gt.Cast[[]map[string]any](t, result2["rows"])
				gt.Equal(t, rows1, rows2)

				result3, err := action.Run(ctx, "bigquery_result", map[string]any{
					"query_id": queryID,
					"limit":    1.0,
					"offset":   1.0,
				})
				gt.NoError(t, err)
				gt.NotEqual(t, result3, nil)
				rows3 := gt.Cast[[]map[string]any](t, result3["rows"])
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
