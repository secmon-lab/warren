package bigquery

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/goerr/v2"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// mockBigQueryClientFactory is a test implementation of BigQueryClientFactory
type mockBigQueryClientFactory struct {
	Client BigQueryClient
}

var _ BigQueryClientFactory = (*mockBigQueryClientFactory)(nil)

func (f *mockBigQueryClientFactory) NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (BigQueryClient, error) {
	return f.Client, nil
}

// mockBigQueryClient is a test implementation of BigQueryClient
type mockBigQueryClient struct {
	// Table metadata configuration
	TableMetadata map[string]*bigquery.TableMetadata
	// Query results configuration (queryID -> rows)
	QueryResults map[string][]map[string]any
	// Error simulation
	Errors map[string]error
	// Record of executed queries
	ExecutedQueries []string
	// Dry run results configuration
	DryRunResults map[string]*bigquery.JobStatistics
}

var _ BigQueryClient = (*mockBigQueryClient)(nil)

func newMockBigQueryClient() *mockBigQueryClient {
	return &mockBigQueryClient{
		TableMetadata:   make(map[string]*bigquery.TableMetadata),
		QueryResults:    make(map[string][]map[string]any),
		Errors:          make(map[string]error),
		ExecutedQueries: make([]string, 0),
		DryRunResults:   make(map[string]*bigquery.JobStatistics),
	}
}

func (c *mockBigQueryClient) Query(query string) BigQueryQuery {
	return &mockBigQueryQuery{
		client: c,
		query:  query,
	}
}

func (c *mockBigQueryClient) Dataset(datasetID string) BigQueryDataset {
	return &mockBigQueryDataset{
		client:    c,
		datasetID: datasetID,
	}
}

func (c *mockBigQueryClient) Close() error {
	if err, ok := c.Errors["close"]; ok {
		return err
	}
	return nil
}

// mockBigQueryQuery is a test implementation of BigQueryQuery
type mockBigQueryQuery struct {
	client *mockBigQueryClient
	query  string
	DryRun bool
}

var _ BigQueryQuery = (*mockBigQueryQuery)(nil)

func (q *mockBigQueryQuery) Run(ctx context.Context) (BigQueryJob, error) {
	if err, ok := q.client.Errors["query_run"]; ok {
		return nil, err
	}

	q.client.ExecutedQueries = append(q.client.ExecutedQueries, q.query)

	return &mockBigQueryJob{
		client: q.client,
		query:  q.query,
		dryRun: q.DryRun,
	}, nil
}

// mockBigQueryJob is a test implementation of BigQueryJob
type mockBigQueryJob struct {
	client *mockBigQueryClient
	query  string
	dryRun bool
}

var _ BigQueryJob = (*mockBigQueryJob)(nil)

func (j *mockBigQueryJob) Wait(ctx context.Context) (*bigquery.JobStatus, error) {
	if err, ok := j.client.Errors["job_wait"]; ok {
		return nil, err
	}

	status := &bigquery.JobStatus{
		State: bigquery.Done,
	}

	if j.dryRun {
		if stats, ok := j.client.DryRunResults[j.query]; ok {
			status.Statistics = stats
		} else {
			// Default dry run result
			status.Statistics = &bigquery.JobStatistics{
				TotalBytesProcessed: 1000, // 1KB
			}
		}
	}

	return status, nil
}

func (j *mockBigQueryJob) Read(ctx context.Context) (BigQueryRowIterator, error) {
	if err, ok := j.client.Errors["job_read"]; ok {
		return nil, err
	}

	if j.dryRun {
		return nil, goerr.New("cannot read from dry run job")
	}

	// Get query results
	var rows []map[string]any
	if result, ok := j.client.QueryResults[j.query]; ok {
		rows = result
	}

	return &mockBigQueryRowIterator{
		rows:  rows,
		index: 0,
	}, nil
}

func (j *mockBigQueryJob) LastStatus() *bigquery.JobStatus {
	status := &bigquery.JobStatus{
		State: bigquery.Done,
	}

	if j.dryRun {
		if stats, ok := j.client.DryRunResults[j.query]; ok {
			status.Statistics = stats
		} else {
			status.Statistics = &bigquery.JobStatistics{
				TotalBytesProcessed: 1000,
			}
		}
	}

	return status
}

func (j *mockBigQueryJob) ID() string {
	return fmt.Sprintf("mock-job-%d", len(j.client.ExecutedQueries))
}

func (j *mockBigQueryJob) Location() string {
	return "US"
}

// mockBigQueryRowIterator is a test implementation of BigQueryRowIterator
type mockBigQueryRowIterator struct {
	rows   []map[string]any
	index  int
	schema bigquery.Schema
}

var _ BigQueryRowIterator = (*mockBigQueryRowIterator)(nil)

func (r *mockBigQueryRowIterator) Next(dst interface{}) error {
	if r.index >= len(r.rows) {
		return iterator.Done
	}

	row := r.rows[r.index]
	r.index++

	// Convert appropriately based on dst type
	switch v := dst.(type) {
	case *[]bigquery.Value:
		// Convert as bigquery.Value array
		values := make([]bigquery.Value, 0, len(row))
		for _, val := range row {
			values = append(values, val)
		}
		*v = values
	default:
		// JSON-based conversion
		data, err := json.Marshal(row)
		if err != nil {
			return goerr.Wrap(err, "failed to marshal row")
		}
		if err := json.Unmarshal(data, dst); err != nil {
			return goerr.Wrap(err, "failed to unmarshal row")
		}
	}

	return nil
}

func (r *mockBigQueryRowIterator) Schema() bigquery.Schema {
	return r.schema
}

// mockBigQueryDataset is a test implementation of BigQueryDataset
type mockBigQueryDataset struct {
	client    *mockBigQueryClient
	datasetID string
}

var _ BigQueryDataset = (*mockBigQueryDataset)(nil)

func (d *mockBigQueryDataset) Table(tableID string) BigQueryTable {
	return &mockBigQueryTable{
		client:    d.client,
		datasetID: d.datasetID,
		tableID:   tableID,
	}
}

// mockBigQueryTable is a test implementation of BigQueryTable
type mockBigQueryTable struct {
	client    *mockBigQueryClient
	datasetID string
	tableID   string
}

var _ BigQueryTable = (*mockBigQueryTable)(nil)

func (t *mockBigQueryTable) Metadata(ctx context.Context) (*bigquery.TableMetadata, error) {
	key := fmt.Sprintf("%s.%s", t.datasetID, t.tableID)
	if err, ok := t.client.Errors["table_metadata"]; ok {
		return nil, err
	}

	if metadata, ok := t.client.TableMetadata[key]; ok {
		return metadata, nil
	}

	// Return default metadata
	return &bigquery.TableMetadata{
		Schema: bigquery.Schema{
			{Name: "id", Type: bigquery.StringFieldType},
			{Name: "name", Type: bigquery.StringFieldType},
			{Name: "timestamp", Type: bigquery.TimestampFieldType},
		},
	}, nil
}

// SetDryRun sets the dry run flag for the query
func (q *mockBigQueryQuery) SetDryRun(dryRun bool) {
	q.DryRun = dryRun
}
