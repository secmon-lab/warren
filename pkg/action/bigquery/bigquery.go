package bigquery

import (
	"context"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/goerr/v2"
	"google.golang.org/api/iterator"
)

type BigQueryClient interface {
	GetMetadata(ctx context.Context, datasetID, tableID string) (*bigquery.TableMetadata, error)
	Query(ctx context.Context, query string, out func(v map[string]bigquery.Value) error) error
	DryRun(ctx context.Context, query string) (*bigquery.JobStatus, error)
	Close() error
}

type BigQueryClientFactory func(ctx context.Context, projectID string) (BigQueryClient, error)

type BigQueryClientImpl struct {
	client *bigquery.Client
}

func newBigQueryClient(ctx context.Context, projectID string) (BigQueryClient, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create bigquery client")
	}
	return &BigQueryClientImpl{client: client}, nil
}

func (x *BigQueryClientImpl) Query(ctx context.Context, query string, out func(v map[string]bigquery.Value) error) error {
	iter, err := x.client.Query(query).Read(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to run bigquery query")
	}

	for {
		var v map[string]bigquery.Value
		if err := iter.Next(&v); err != nil {
			if err == iterator.Done {
				break
			}
			return goerr.Wrap(err, "failed to read bigquery query")
		}
		if err := out(v); err != nil {
			return goerr.Wrap(err, "failed to write bigquery query")
		}
	}

	return nil
}

func (x *BigQueryClientImpl) DryRun(ctx context.Context, query string) (*bigquery.JobStatus, error) {
	q := x.client.Query(query)
	q.DryRun = true

	job, err := q.Run(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to run bigquery query")
	}

	return job.LastStatus(), nil
}

func (x *BigQueryClientImpl) GetMetadata(ctx context.Context, datasetID, tableID string) (*bigquery.TableMetadata, error) {
	table := x.client.Dataset(datasetID).Table(tableID)
	meta, err := table.Metadata(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get bigquery table metadata")
	}
	return meta, nil
}

func (x *BigQueryClientImpl) Close() error {
	return x.client.Close()
}
