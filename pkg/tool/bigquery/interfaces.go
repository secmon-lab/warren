package bigquery

import (
	"context"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"
)

// BigQueryClient is an interface for BigQuery client
type BigQueryClient interface {
	Query(query string) BigQueryQuery
	Dataset(datasetID string) BigQueryDataset
	Close() error
}

// BigQueryQuery is an interface for BigQuery query
type BigQueryQuery interface {
	Run(ctx context.Context) (BigQueryJob, error)
	SetDryRun(dryRun bool)
}

// BigQueryJob is an interface for BigQuery job
type BigQueryJob interface {
	Wait(ctx context.Context) (*bigquery.JobStatus, error)
	Read(ctx context.Context) (BigQueryRowIterator, error)
	LastStatus() *bigquery.JobStatus
	ID() string
	Location() string
}

// BigQueryRowIterator is an interface for BigQuery row iterator
type BigQueryRowIterator interface {
	Next(dst interface{}) error
	Schema() bigquery.Schema
}

// BigQueryDataset is an interface for BigQuery dataset
type BigQueryDataset interface {
	Table(tableID string) BigQueryTable
}

// BigQueryTable is an interface for BigQuery table
type BigQueryTable interface {
	Metadata(ctx context.Context) (*bigquery.TableMetadata, error)
}

// BigQueryClientFactory is an interface for creating BigQuery clients
type BigQueryClientFactory interface {
	NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (BigQueryClient, error)
}

// DefaultBigQueryClientFactory is the default implementation of BigQueryClientFactory
type DefaultBigQueryClientFactory struct{}

var _ BigQueryClientFactory = (*DefaultBigQueryClientFactory)(nil)

func (f *DefaultBigQueryClientFactory) NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (BigQueryClient, error) {
	client, err := bigquery.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, err
	}
	return &DefaultBigQueryClient{client: client}, nil
}

// DefaultBigQueryClient is the default implementation of BigQueryClient
type DefaultBigQueryClient struct {
	client *bigquery.Client
}

var _ BigQueryClient = (*DefaultBigQueryClient)(nil)

func (c *DefaultBigQueryClient) Query(query string) BigQueryQuery {
	return &DefaultBigQueryQuery{query: c.client.Query(query)}
}

func (c *DefaultBigQueryClient) Dataset(datasetID string) BigQueryDataset {
	return &DefaultBigQueryDataset{dataset: c.client.Dataset(datasetID)}
}

func (c *DefaultBigQueryClient) Close() error {
	return c.client.Close()
}

// DefaultBigQueryQuery is the default implementation of BigQueryQuery
type DefaultBigQueryQuery struct {
	query *bigquery.Query
}

var _ BigQueryQuery = (*DefaultBigQueryQuery)(nil)

func (q *DefaultBigQueryQuery) Run(ctx context.Context) (BigQueryJob, error) {
	job, err := q.query.Run(ctx)
	if err != nil {
		return nil, err
	}
	return &DefaultBigQueryJob{job: job}, nil
}

func (q *DefaultBigQueryQuery) SetDryRun(dryRun bool) {
	q.query.DryRun = dryRun
}

// DefaultBigQueryJob is the default implementation of BigQueryJob
type DefaultBigQueryJob struct {
	job *bigquery.Job
}

var _ BigQueryJob = (*DefaultBigQueryJob)(nil)

func (j *DefaultBigQueryJob) Wait(ctx context.Context) (*bigquery.JobStatus, error) {
	return j.job.Wait(ctx)
}

func (j *DefaultBigQueryJob) Read(ctx context.Context) (BigQueryRowIterator, error) {
	iter, err := j.job.Read(ctx)
	if err != nil {
		return nil, err
	}
	return &DefaultBigQueryRowIterator{iter: iter}, nil
}

func (j *DefaultBigQueryJob) LastStatus() *bigquery.JobStatus {
	return j.job.LastStatus()
}

func (j *DefaultBigQueryJob) ID() string {
	return j.job.ID()
}

func (j *DefaultBigQueryJob) Location() string {
	return j.job.Location()
}

// DefaultBigQueryRowIterator is the default implementation of BigQueryRowIterator
type DefaultBigQueryRowIterator struct {
	iter *bigquery.RowIterator
}

var _ BigQueryRowIterator = (*DefaultBigQueryRowIterator)(nil)

func (r *DefaultBigQueryRowIterator) Next(dst interface{}) error {
	return r.iter.Next(dst)
}

func (r *DefaultBigQueryRowIterator) Schema() bigquery.Schema {
	return r.iter.Schema
}

// DefaultBigQueryDataset is the default implementation of BigQueryDataset
type DefaultBigQueryDataset struct {
	dataset *bigquery.Dataset
}

var _ BigQueryDataset = (*DefaultBigQueryDataset)(nil)

func (d *DefaultBigQueryDataset) Table(tableID string) BigQueryTable {
	return &DefaultBigQueryTable{table: d.dataset.Table(tableID)}
}

// DefaultBigQueryTable is the default implementation of BigQueryTable
type DefaultBigQueryTable struct {
	table *bigquery.Table
}

var _ BigQueryTable = (*DefaultBigQueryTable)(nil)

func (t *DefaultBigQueryTable) Metadata(ctx context.Context) (*bigquery.TableMetadata, error) {
	return t.table.Metadata(ctx)
}
