package bigquery_test

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/gt"
	bq "github.com/secmon-lab/warren/pkg/action/bigquery"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func TestBigQueryClient(t *testing.T) {
	vars := test.NewEnvVars(t, "TEST_BIGQUERY_PROJECT_ID", "TEST_BIGQUERY_DATASET_ID", "TEST_BIGQUERY_TABLE_ID")

	ctx := context.Background()
	client, err := bq.NewBigQueryClient(ctx, vars.Get("TEST_BIGQUERY_PROJECT_ID"))
	gt.NoError(t, err)
	defer client.Close()

	t.Run("get metadata", func(t *testing.T) {
		meta, err := client.GetMetadata(ctx, vars.Get("TEST_BIGQUERY_DATASET_ID"), vars.Get("TEST_BIGQUERY_TABLE_ID"))
		gt.NoError(t, err)
		gt.Equal(t, meta.Schema, bigquery.Schema{
			{
				Name:     "name",
				Required: false,
				Type:     bigquery.StringFieldType,
			},
		})
	})

	q := fmt.Sprintf("SELECT * FROM `%s.%s.%s` LIMIT 1000", vars.Get("TEST_BIGQUERY_PROJECT_ID"), vars.Get("TEST_BIGQUERY_DATASET_ID"), vars.Get("TEST_BIGQUERY_TABLE_ID"))

	t.Run("dry run", func(t *testing.T) {
		job, err := client.DryRun(ctx, q)
		gt.NoError(t, err)
		gt.N(t, job.Statistics.TotalBytesProcessed).Greater(0)
	})

	t.Run("query", func(t *testing.T) {
		var rows []map[string]bigquery.Value
		out := func(v map[string]bigquery.Value) error {
			rows = append(rows, v)
			return nil
		}
		err := client.Query(ctx, q, out)
		gt.NoError(t, err)
		gt.A(t, rows).Longer(0)
	})
}
