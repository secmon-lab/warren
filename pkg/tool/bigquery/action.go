package bigquery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/urfave/cli/v3"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

type Action struct {
	projectID     string
	credentials   string
	configFile    string
	storageBucket string
	storagePrefix string
	timeout       time.Duration
	config        *Config
}

type Config struct {
	Datasets map[string]DatasetConfig `yaml:"datasets"`
}

type DatasetConfig struct {
	Tables []TableConfig `yaml:"tables"`
}

type TableConfig struct {
	// TableID
	TableID string `yaml:"table_id"`

	// Description of the table
	Description string `yaml:"description"`

	// Columns of the table. It's not required to describe all column.
	Columns []ColumnConfig `yaml:"columns"`
}

type ColumnConfig struct {
	Name         string         `yaml:"name"`
	Description  string         `yaml:"description"`
	ValueExample string         `yaml:"value_example"`
	Type         string         `yaml:"type"`   // STRING, INTEGER, FLOAT, BOOLEAN, TIMESTAMP, DATE, TIME, DATETIME, BYTES, RECORD
	Fields       []ColumnConfig `yaml:"fields"` // for RECORD type
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

func (x *Action) processResults(_ context.Context, processor rowProcessor, limit, offset int, objectPath string) (map[string]any, error) {
	var rows []map[string]bigquery.Value
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

		rows = append(rows, row)
		rowJSON, err := json.Marshal(row)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to calculate row size")
		}
		totalSize += int64(len(rowJSON))

		if err := processor.writeRow(row); err != nil {
			return nil, err
		}

		currentRow++
	}

	return map[string]any{
		"status":     "completed",
		"gcs_path":   fmt.Sprintf("gs://%s/%s", x.storageBucket, objectPath),
		"rows":       rows,
		"total_rows": totalRows,
		"total_size": totalSize,
		"limit":      limit,
		"offset":     offset,
		"has_more":   currentRow > offset+limit,
	}, nil
}

func (x *Action) processStorageResults(ctx context.Context, reader *storage.Reader, limit, offset int, objectPath string) (map[string]any, error) {
	processor := &storageRowProcessor{
		decoder: json.NewDecoder(reader),
	}
	return x.processResults(ctx, processor, limit, offset, objectPath)
}

func (x *Action) processBigQueryResults(ctx context.Context, it *bigquery.RowIterator, writer *storage.Writer, limit, offset int, objectPath string) (map[string]any, error) {
	processor := &bigQueryRowProcessor{
		iterator: it,
		writer:   writer,
	}
	return x.processResults(ctx, processor, limit, offset, objectPath)
}

func (x *Action) Name() string {
	return "bigquery"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "bigquery-project-id",
			Usage:       "Google Cloud Project ID",
			Destination: &x.projectID,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_PROJECT_ID"),
		},
		&cli.StringFlag{
			Name:        "bigquery-credentials",
			Usage:       "Path to Google Cloud credentials JSON file (optional)",
			Destination: &x.credentials,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_CREDENTIALS"),
		},
		&cli.StringFlag{
			Name:        "bigquery-config",
			Usage:       "Path to configuration YAML file",
			Destination: &x.configFile,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_CONFIG"),
		},
		&cli.StringFlag{
			Name:        "bigquery-storage-bucket",
			Usage:       "GCS bucket name for storing query results",
			Destination: &x.storageBucket,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_STORAGE_BUCKET"),
		},
		&cli.StringFlag{
			Name:        "bigquery-storage-prefix",
			Usage:       "Prefix for GCS object path for storing query results",
			Destination: &x.storagePrefix,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_STORAGE_PREFIX"),
		},
		&cli.DurationFlag{
			Name:        "bigquery-timeout",
			Usage:       "Timeout for query execution",
			Destination: &x.timeout,
			Category:    "Tool",
			Value:       5 * time.Minute,
			Sources:     cli.EnvVars("WARREN_BIGQUERY_TIMEOUT"),
		},
	}
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "bigquery_list_dataset",
			Description: "List available BigQuery datasets and tables",
			Parameters:  map[string]*gollem.Parameter{},
		},
		{
			Name:        "bigquery_query",
			Description: "Execute a BigQuery query and return the query ID",
			Parameters: map[string]*gollem.Parameter{
				"query": {
					Type:        gollem.TypeString,
					Description: "The SQL query to execute",
				},
			},
			Required: []string{"query"},
		},
		{
			Name:        "bigquery_result",
			Description: "Get the results of a previously executed query",
			Parameters: map[string]*gollem.Parameter{
				"query_id": {
					Type:        gollem.TypeString,
					Description: "The ID of the query to get results for",
				},
				"limit": {
					Type:        gollem.TypeInteger,
					Description: "Maximum number of rows to return (default: 100)",
					Required:    []string{},
				},
				"offset": {
					Type:        gollem.TypeInteger,
					Description: "Number of rows to skip (default: 0)",
					Required:    []string{},
				},
			},
			Required: []string{"query_id"},
		},
		{
			Name:        "bigquery_schema",
			Description: "Get schema information for a specific table",
			Parameters: map[string]*gollem.Parameter{
				"dataset_id": {
					Type:        gollem.TypeString,
					Description: "The dataset ID containing the table",
				},
				"table_id": {
					Type:        gollem.TypeString,
					Description: "The table ID to get schema for",
				},
			},
			Required: []string{"dataset_id", "table_id"},
		},
	}, nil
}

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
		datasetID, ok := args["dataset_id"].(string)
		if !ok {
			return nil, goerr.New("dataset_id parameter is required")
		}
		tableID, ok := args["table_id"].(string)
		if !ok {
			return nil, goerr.New("table_id parameter is required")
		}
		return x.getTableSchema(datasetID, tableID)
	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}
}

func (x *Action) Configure(ctx context.Context) error {
	if x.projectID == "" {
		return errs.ErrActionUnavailable
	}

	if x.configFile == "" {
		return goerr.New("configuration file is required")
	}

	// Read and parse the configuration file
	data, err := os.ReadFile(x.configFile)
	if err != nil {
		return goerr.Wrap(err, "failed to read configuration file")
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return goerr.Wrap(err, "failed to parse configuration file")
	}

	x.config = &config
	return nil
}

func (x *Action) listDatasets() (map[string]any, error) {
	if x.config == nil {
		return nil, goerr.New("configuration is not loaded")
	}

	// Convert config to JSON and back to ensure all fields are included
	jsonData, err := json.Marshal(x.config)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal config to JSON")
	}

	var result map[string]any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal config from JSON")
	}

	return map[string]any{
		"datasets": result["datasets"],
	}, nil
}

type queryMetadata struct {
	JobID    string `json:"job_id"`
	Location string `json:"location"`
}

func (x *Action) executeQuery(ctx context.Context, client *bigquery.Client, query string) (map[string]any, error) {
	q := client.Query(query)

	job, err := q.Run(ctx)
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

		return x.processStorageResults(ctx, reader, limit, offset, objectPath)
	} else if !errors.Is(err, storage.ErrObjectNotExist) {
		// Return error if it's not a "file not found" error
		return nil, goerr.Wrap(err, "failed to check existing result file")
	}

	// Process new query results
	job, err := client.JobFromIDLocation(ctx, metadata.JobID, metadata.Location)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get job")
	}

	status, err := job.Wait(ctx)
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
	result, err := x.processBigQueryResults(ctx, it, writer, limit, offset, objectPath)
	if err != nil {
		return nil, err
	}

	// Write results to storage for future use
	if err := writer.Close(); err != nil {
		return nil, goerr.Wrap(err, "failed to close GCS writer")
	}

	return result, nil
}

func (x *Action) getTableSchema(datasetID, tableID string) (map[string]any, error) {
	if x.config == nil {
		return nil, goerr.New("configuration is not loaded")
	}

	datasetConfig, ok := x.config.Datasets[datasetID]
	if !ok {
		return nil, goerr.New("dataset not found in configuration", goerr.V("dataset_id", datasetID))
	}

	for _, table := range datasetConfig.Tables {
		if table.TableID == tableID {
			// Convert table config to JSON and back to ensure all fields are included
			jsonData, err := json.Marshal(table)
			if err != nil {
				return nil, goerr.Wrap(err, "failed to marshal table config to JSON")
			}

			var result map[string]any
			if err := json.Unmarshal(jsonData, &result); err != nil {
				return nil, goerr.Wrap(err, "failed to unmarshal table config from JSON")
			}

			return result, nil
		}
	}

	return nil, goerr.New("table not found in configuration",
		goerr.V("dataset_id", datasetID),
		goerr.V("table_id", tableID))
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("project_id", x.projectID),
		slog.String("credentials", x.credentials),
		slog.String("config_file", x.configFile),
		slog.String("storage_bucket", x.storageBucket),
		slog.Duration("timeout", x.timeout),
	)
}
