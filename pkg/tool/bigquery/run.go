package bigquery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.projectID == "" {
		return nil, goerr.New("BigQuery project ID is required")
	}

	var opts []option.ClientOption
	if x.credentials != "" {
		opts = append(opts, option.WithCredentialsFile(x.credentials))
	}

	client, err := bigquery.NewClient(ctx, x.projectID, opts...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create BigQuery client")
	}
	defer safe.Close(ctx, client)

	switch name {
	case "bigquery_list_dataset":
		return x.listDatasets()

	case "bigquery_query":
		query, ok := args["query"].(string)
		if !ok {
			return nil, goerr.New("query parameter is required")
		}
		return x.executeQuery(ctx, client, query)

	case "bigquery_result":
		queryID, ok := args["query_id"].(string)
		if !ok {
			return nil, goerr.New("query_id parameter is required")
		}
		limit := 100
		if l, ok := args["limit"].(int); ok {
			limit = l
		}
		offset := 0
		if o, ok := args["offset"].(int); ok {
			offset = o
		}
		return x.getQueryResults(ctx, client, queryID, limit, offset)

	case "bigquery_schema":
		projectID, ok := args["project_id"].(string)
		if !ok {
			return nil, goerr.New("project_id parameter is required")
		}
		datasetID, ok := args["dataset_id"].(string)
		if !ok {
			return nil, goerr.New("dataset_id parameter is required")
		}
		tableID, ok := args["table_id"].(string)
		if !ok {
			return nil, goerr.New("table_id parameter is required")
		}
		return x.getTableSchema(ctx, projectID, datasetID, tableID)

	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}
}

type rowProcessor interface {
	processRow() (map[string]bigquery.Value, error)
	writeRow(row map[string]bigquery.Value) error
}

type storageRowProcessor struct {
	decoder *json.Decoder
	writer  *storage.Writer
}

func (p *storageRowProcessor) processRow() (map[string]bigquery.Value, error) {
	var row map[string]bigquery.Value
	if !p.decoder.More() {
		return nil, iterator.Done
	}
	if err := p.decoder.Decode(&row); err != nil {
		return nil, goerr.Wrap(err, "failed to decode row from JSON")
	}
	return row, nil
}

func (p *storageRowProcessor) writeRow(row map[string]bigquery.Value) error {
	if p.writer != nil {
		if err := json.NewEncoder(p.writer).Encode(row); err != nil {
			return goerr.Wrap(err, "failed to encode row to JSON")
		}
	}
	return nil
}

type bigQueryRowProcessor struct {
	iterator *bigquery.RowIterator
	writer   *storage.Writer
}

func (p *bigQueryRowProcessor) processRow() (map[string]bigquery.Value, error) {
	var row []bigquery.Value
	if err := p.iterator.Next(&row); err != nil {
		if err == iterator.Done {
			return nil, iterator.Done
		}
		return nil, goerr.Wrap(err, "failed to iterate results")
	}

	rowMap := make(map[string]bigquery.Value)
	for i, field := range p.iterator.Schema {
		rowMap[field.Name] = row[i]
	}
	return rowMap, nil
}

func (p *bigQueryRowProcessor) writeRow(row map[string]bigquery.Value) error {
	if p.writer != nil {
		if err := json.NewEncoder(p.writer).Encode(row); err != nil {
			return goerr.Wrap(err, "failed to encode row to JSON")
		}
	}
	return nil
}

func (x *Action) toResultStoragePath(queryID string) string {
	return fmt.Sprintf("%sbigquery/%s/data.json", x.storagePrefix, queryID)
}

func (x *Action) toMetadataStoragePath(queryID string) string {
	return fmt.Sprintf("%sbigquery/%s/metadata.json", x.storagePrefix, queryID)
}

func (x *Action) processResults(_ context.Context, processor rowProcessor, limit, offset int) (map[string]any, error) {
	var rows []map[string]any
	var totalSize int64
	var totalRows int
	currentRow := 0

	for {
		row, err := processor.processRow()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		totalRows++

		// Skip rows before offset
		if currentRow < offset {
			currentRow++
			continue
		}

		// Stop if we've reached the limit
		if len(rows) >= limit {
			continue
		}

		rowJSON, err := json.Marshal(row)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to calculate row size")
		}
		var rowMap map[string]any
		if err := json.Unmarshal(rowJSON, &rowMap); err != nil {
			return nil, goerr.Wrap(err, "failed to unmarshal row")
		}
		rows = append(rows, rowMap)

		totalSize += int64(len(rowJSON))

		if err := processor.writeRow(row); err != nil {
			return nil, err
		}

		currentRow++
	}

	return map[string]any{
		"rows":       rows,
		"total_rows": totalRows,
		"total_size": totalSize,
		"limit":      limit,
		"offset":     offset,
		"has_more":   currentRow > offset+limit,
	}, nil
}

func (x *Action) processStorageResults(ctx context.Context, reader *storage.Reader, limit, offset int) (map[string]any, error) {
	processor := &storageRowProcessor{
		decoder: json.NewDecoder(reader),
	}
	return x.processResults(ctx, processor, limit, offset)
}

func (x *Action) processBigQueryResults(ctx context.Context, it *bigquery.RowIterator, writer *storage.Writer, limit, offset int) (map[string]any, error) {
	processor := &bigQueryRowProcessor{
		iterator: it,
		writer:   writer,
	}
	return x.processResults(ctx, processor, limit, offset)
}

func (x *Action) listDatasets() (map[string]any, error) {
	if len(x.configs) == 0 {
		return nil, goerr.New("configuration is not loaded")
	}

	// Convert config to JSON and back to ensure all fields are included
	jsonData, err := json.Marshal(x.configs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal config to JSON")
	}

	var result []map[string]any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal config from JSON")
	}

	return map[string]any{
		"config": result,
	}, nil
}

type queryMetadata struct {
	JobID    string `json:"job_id"`
	Location string `json:"location"`
}

func (x *Action) executeQuery(ctx context.Context, client *bigquery.Client, query string) (map[string]any, error) {
	q := client.Query(query)

	// Perform dry run to check scan size
	q.DryRun = true

	job, err := q.Run(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to dry run query")
	}

	status, err := job.Wait(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to wait for dry run job")
	}
	if err := status.Err(); err != nil {
		return nil, goerr.Wrap(err, "dry run job failed")
	}
	if uint64(status.Statistics.TotalBytesProcessed) > x.scanLimit {
		return nil, goerr.New("query scan size exceeds limit",
			goerr.V("scan_size", status.Statistics.TotalBytesProcessed),
			goerr.V("scan_limit", x.scanLimit))
	}

	// Execute the actual query
	q.DryRun = false
	job, err = q.Run(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to run query")
	}

	storageClient, err := x.newStorageClient(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create storage client")
	}
	defer safe.Close(ctx, storageClient)

	queryID := uuid.New().String()
	metadataPath := x.toMetadataStoragePath(queryID)

	metadata := queryMetadata{
		JobID:    job.ID(),
		Location: job.Location(),
	}

	writer := storageClient.Bucket(x.storageBucket).Object(metadataPath).NewWriter(ctx)
	defer safe.Close(ctx, writer)

	if err := json.NewEncoder(writer).Encode(metadata); err != nil {
		return nil, goerr.Wrap(err, "failed to write metadata to GCS")
	}

	return map[string]any{
		"query_id": queryID,
	}, nil
}

func (x *Action) newStorageClient(ctx context.Context) (*storage.Client, error) {
	var opts []option.ClientOption
	if x.credentials != "" {
		opts = append(opts, option.WithCredentialsFile(x.credentials))
	}
	return storage.NewClient(ctx, opts...)
}

func (x *Action) getQueryResults(ctx context.Context, client *bigquery.Client, queryID string, limit, offset int) (map[string]any, error) {
	// First, check if the result file already exists
	storageClient, err := x.newStorageClient(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create storage client")
	}
	defer safe.Close(ctx, storageClient)

	metadataPath := x.toMetadataStoragePath(queryID)
	metadataReader, err := storageClient.Bucket(x.storageBucket).Object(metadataPath).NewReader(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create metadata reader")
	}
	defer safe.Close(ctx, metadataReader)

	var metadata queryMetadata
	if err := json.NewDecoder(metadataReader).Decode(&metadata); err != nil {
		return nil, goerr.Wrap(err, "failed to decode metadata")
	}

	objectPath := x.toResultStoragePath(queryID)
	object := storageClient.Bucket(x.storageBucket).Object(objectPath)

	// Check if the result file already exists
	_, err = object.Attrs(ctx)
	if err == nil {
		// If the file exists, read and process it with pagination
		reader, err := object.NewReader(ctx)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to create GCS reader")
		}
		defer safe.Close(ctx, reader)

		return x.processStorageResults(ctx, reader, limit, offset)
	} else if !errors.Is(err, storage.ErrObjectNotExist) {
		// Return error if it's not a "file not found" error
		return nil, goerr.Wrap(err, "failed to check existing result file")
	}

	// Process new query results
	job, err := client.JobFromIDLocation(ctx, metadata.JobID, metadata.Location)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get job")
	}

	// Wait for the job to complete
	waitCtx, cancel := context.WithTimeout(ctx, x.timeout)
	defer cancel()
	status, err := job.Wait(waitCtx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to wait for job")
	}

	if err := status.Err(); err != nil {
		return nil, goerr.Wrap(err, "job failed")
	}

	it, err := job.Read(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read query results")
	}

	// Create a temporary buffer to store results
	writer := object.NewWriter(ctx)
	defer safe.Close(ctx, writer)

	// Process results and write to storage
	result, err := x.processBigQueryResults(ctx, it, writer, limit, offset)
	if err != nil {
		return nil, err
	}

	// Write results to storage for future use
	if err := writer.Close(); err != nil {
		return nil, goerr.Wrap(err, "failed to close GCS writer")
	}

	return result, nil
}

func (x *Action) getTableSchema(ctx context.Context, projectID, datasetID, tableID string) (map[string]any, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create BigQuery client")
	}
	defer safe.Close(ctx, client)

	table := client.Dataset(datasetID).Table(tableID)
	metadata, err := table.Metadata(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get table metadata", goerr.V("dataset_id", datasetID), goerr.V("table_id", tableID))
	}

	// Convert schema to JSON
	schemaJSON, err := json.Marshal(metadata.Schema)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal schema to JSON")
	}

	var result []map[string]any
	if err := json.Unmarshal(schemaJSON, &result); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal schema from JSON")
	}

	return map[string]any{
		"schema": result,
	}, nil
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("project_id", x.projectID),
		slog.String("credentials", x.credentials),
		slog.Any("config_files", x.configFiles),
		slog.String("storage_bucket", x.storageBucket),
		slog.Duration("timeout", x.timeout),
	)
}
