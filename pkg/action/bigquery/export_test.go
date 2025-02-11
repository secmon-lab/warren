package bigquery

import (
	"context"

	"cloud.google.com/go/bigquery"
)

func (x *Action) ByteLimit() int64 {
	return x.byteLimit
}

func (x *Action) LimitRows() int64 {
	return x.cfg.Limit.Rows
}

func (x *Action) Tables() []tableConfig {
	return x.cfg.Tables
}

func (x *Action) SetBQClientFactory(factory BigQueryClientFactory) {
	x.bqFactory = factory
}

func NewBigQueryClient(ctx context.Context, projectID string) (BigQueryClient, error) {
	return newBigQueryClient(ctx, projectID)
}

func GenerateQuery(fullTableID string, schema bigquery.Schema, limit int64) (string, error) {
	return generateQuery(fullTableID, schema, limit)
}
