package bigquery_test

import (
	"context"
	"encoding/json"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/gt"
	bq "github.com/secmon-lab/warren/pkg/tool/bigquery"
)

func TestMockBigQueryClient(t *testing.T) {
	ctx := context.Background()

	// Create mock client
	mockClient := bq.NewMockBigQueryClient()

	// Set up table metadata
	mockClient.TableMetadata["test.users"] = &bigquery.TableMetadata{
		Schema: bigquery.Schema{
			{Name: "user_id", Type: bigquery.StringFieldType, Description: "User identifier"},
			{Name: "email", Type: bigquery.StringFieldType, Description: "User email address"},
			{Name: "login_time", Type: bigquery.TimestampFieldType, Description: "Last login timestamp"},
			{Name: "ip_address", Type: bigquery.StringFieldType, Description: "Source IP address"},
		},
	}

	// Set up query results
	mockClient.QueryResults["SELECT * FROM test.users LIMIT 5"] = []map[string]any{
		{"user_id": "user1", "email": "user1@example.com", "login_time": "2024-01-01T00:00:00Z", "ip_address": "192.168.1.1"},
		{"user_id": "user2", "email": "user2@example.com", "login_time": "2024-01-01T01:00:00Z", "ip_address": "192.168.1.2"},
	}

	// Test dataset and table access
	dataset := mockClient.Dataset("test")
	table := dataset.Table("users")

	metadata, err := table.Metadata(ctx)
	gt.NoError(t, err)
	gt.A(t, metadata.Schema).Length(4)
	gt.Equal(t, metadata.Schema[0].Name, "user_id")
	gt.Equal(t, metadata.Schema[0].Description, "User identifier")

	// Test query execution
	query := mockClient.Query("SELECT * FROM test.users LIMIT 5")
	job, err := query.Run(ctx)
	gt.NoError(t, err)

	status, err := job.Wait(ctx)
	gt.NoError(t, err)
	gt.Equal(t, status.State, bigquery.Done)

	iter, err := job.Read(ctx)
	gt.NoError(t, err)

	var rows []map[string]any
	for {
		var row map[string]any
		err := iter.Next(&row)
		if err != nil {
			break
		}
		rows = append(rows, row)
	}

	gt.A(t, rows).Length(2)
	gt.Equal(t, rows[0]["user_id"], "user1")
	gt.Equal(t, rows[1]["user_id"], "user2")

	// Verify executed queries
	gt.A(t, mockClient.ExecutedQueries).Length(1)
	gt.Equal(t, mockClient.ExecutedQueries[0], "SELECT * FROM test.users LIMIT 5")
}

func TestMockBigQueryClientDryRun(t *testing.T) {
	ctx := context.Background()

	mockClient := bq.NewMockBigQueryClient()

	// Set up dry run result
	mockClient.DryRunResults["SELECT COUNT(*) FROM large_table"] = &bigquery.JobStatistics{
		TotalBytesProcessed: 1000000000, // 1GB
	}

	query := mockClient.Query("SELECT COUNT(*) FROM large_table")
	query.SetDryRun(true)

	job, err := query.Run(ctx)
	gt.NoError(t, err)

	status, err := job.Wait(ctx)
	gt.NoError(t, err)
	gt.Equal(t, status.State, bigquery.Done)
	gt.Equal(t, status.Statistics.TotalBytesProcessed, int64(1000000000))

	// Dry run job should not be readable
	_, err = job.Read(ctx)
	gt.Error(t, err)
}

func TestMockBigQueryClientFactory(t *testing.T) {
	ctx := context.Background()

	mockClient := bq.NewMockBigQueryClient()
	factory := &bq.MockBigQueryClientFactory{
		Client: mockClient,
	}

	client, err := factory.NewClient(ctx, "test-project")
	gt.NoError(t, err)
	gt.NotNil(t, client)
}

func TestDefaultBigQueryClientFactory(t *testing.T) {
	// This test would require actual BigQuery credentials
	// Skip if not running integration tests
	t.Skip("Requires actual BigQuery credentials for integration testing")
}

func TestConfigSchemaGeneration(t *testing.T) {
	// Test that the config schema generation works correctly
	schema := bq.GenerateConfigSchema()
	gt.True(t, len(schema) > 0)

	// Verify that the schema contains expected fields
	gt.S(t, schema).Contains("dataset_id")
	gt.S(t, schema).Contains("table_id")
	gt.S(t, schema).Contains("description")
	gt.S(t, schema).Contains("columns")
	gt.S(t, schema).Contains("partitioning")

	// Verify that nested structures are included
	gt.S(t, schema).Contains("value_example")
	gt.S(t, schema).Contains("fields")
	gt.S(t, schema).Contains("time_unit")

	// Verify it's valid JSON
	var parsed map[string]any
	err := json.Unmarshal([]byte(schema), &parsed)
	gt.NoError(t, err)

	t.Logf("Generated schema: %s", schema)
}
