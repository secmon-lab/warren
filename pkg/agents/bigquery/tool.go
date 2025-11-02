package bigquery

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// internalTool implements the actual BigQuery operations
type internalTool struct {
	config    *Config
	projectID string
}

// ID implements gollem.ToolSet
func (t *internalTool) ID() string {
	return "bigquery_agent"
}

// Specs implements gollem.ToolSet
func (t *internalTool) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	// Build table descriptions for the prompt
	tableDescriptions := ""
	for i, table := range t.config.Tables {
		tableDescriptions += fmt.Sprintf("\n%d. %s.%s.%s: %s",
			i+1, table.ProjectID, table.DatasetID, table.TableID, table.Description)
	}

	return []gollem.ToolSpec{
		{
			Name: "bigquery_query",
			Description: fmt.Sprintf(`Execute a BigQuery SQL query to retrieve data.
Available tables:%s

Important guidelines:
- Always specify the full table name as project.dataset.table
- Use LIMIT to restrict the number of rows returned
- Be mindful of scan size limits (current limit: %s)
- For time-based queries, use proper date/time functions`,
				tableDescriptions,
				humanize.Bytes(t.config.ScanSizeLimit),
			),
			Parameters: map[string]*gollem.Parameter{
				"sql": {
					Type:        gollem.TypeString,
					Description: "The SQL query to execute. Must be a valid BigQuery SQL query.",
				},
				"description": {
					Type:        gollem.TypeString,
					Description: "Brief description of what this query is trying to achieve",
				},
			},
			Required: []string{"sql"},
		},
		{
			Name:        "bigquery_schema",
			Description: "Get detailed schema information for a specific BigQuery table",
			Parameters: map[string]*gollem.Parameter{
				"project_id": {
					Type:        gollem.TypeString,
					Description: "The project ID of the table",
				},
				"dataset_id": {
					Type:        gollem.TypeString,
					Description: "The dataset ID of the table",
				},
				"table_id": {
					Type:        gollem.TypeString,
					Description: "The table ID to get schema for",
				},
			},
			Required: []string{"project_id", "dataset_id", "table_id"},
		},
	}, nil
}

// Run implements gollem.ToolSet
func (t *internalTool) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	switch name {
	case "bigquery_query":
		return t.executeQuery(ctx, args)
	case "bigquery_schema":
		return t.getTableSchema(ctx, args)
	default:
		return nil, goerr.New("unknown function", goerr.V("name", name))
	}
}

func (t *internalTool) executeQuery(ctx context.Context, args map[string]any) (map[string]any, error) {
	sql, ok := args["sql"].(string)
	if !ok {
		return nil, goerr.New("sql parameter is required")
	}

	// Trace: Query to be executed
	msg.Trace(ctx, "📝 Executing query: %s", sql)

	// Determine project ID: CLI flag takes precedence over config
	projectID := t.projectID
	if projectID == "" {
		// Fallback to first table's project ID from config
		if len(t.config.Tables) == 0 {
			return nil, goerr.New("no project ID specified and no tables configured")
		}
		projectID = t.config.Tables[0].ProjectID
	}

	// Create BigQuery client
	client, err := t.newBigQueryClient(ctx, projectID)
	if err != nil {
		msg.Trace(ctx, "❌ Failed to create BigQuery client: %v", err)
		return nil, goerr.Wrap(err, "failed to create BigQuery client")
	}
	defer func() {
		if err := client.Close(); err != nil {
			msg.Trace(ctx, "⚠️ Failed to close BigQuery client: %v", err)
		}
	}()

	// Create query
	q := client.Query(sql)

	// Perform dry run to check scan size
	q.DryRun = true
	job, err := q.Run(ctx)
	if err != nil {
		msg.Trace(ctx, "❌ Dry run failed: %v", err)
		return nil, goerr.Wrap(err, "failed to dry run query")
	}

	totalBytes := job.LastStatus().Statistics.TotalBytesProcessed
	if totalBytes < 0 {
		msg.Trace(ctx, "❌ Invalid negative bytes processed: %d", totalBytes)
		return nil, goerr.New("invalid negative bytes processed",
			goerr.V("bytes_processed", totalBytes))
	}

	// Trace: Scan size check
	msg.Trace(ctx, "📊 Scan size: %s (limit: %s)",
		humanize.Bytes(uint64(totalBytes)),
		humanize.Bytes(t.config.ScanSizeLimit))

	// Check scan size limit
	if totalBytes > 0 && uint64(totalBytes) > t.config.ScanSizeLimit {
		msg.Trace(ctx, "❌ Query scan size exceeds limit")

		return nil, goerr.New("query scan size exceeds limit",
			goerr.V("scan_size", totalBytes),
			goerr.V("scan_limit", t.config.ScanSizeLimit))
	}

	// Execute the actual query
	q.DryRun = false
	job, err = q.Run(ctx)
	if err != nil {
		msg.Trace(ctx, "❌ Query execution failed: %v", err)
		return nil, goerr.Wrap(err, "failed to run query")
	}

	// Trace: Job started
	msg.Trace(ctx, "⏳ Waiting for query job to complete...")

	// Wait for job to complete (with timeout)
	waitCtx, cancel := context.WithTimeout(ctx, t.config.GetQueryTimeout())
	defer cancel()

	status, err := job.Wait(waitCtx)
	if err != nil {
		msg.Trace(ctx, "❌ Job wait failed: %v", err)
		return nil, goerr.Wrap(err, "failed to wait for job completion")
	}

	if err := status.Err(); err != nil {
		msg.Trace(ctx, "❌ Job execution failed: %v", err)
		return nil, goerr.Wrap(err, "job failed")
	}

	// Read results
	it, err := job.Read(ctx)
	if err != nil {
		msg.Trace(ctx, "❌ Failed to read query results: %v", err)
		return nil, goerr.Wrap(err, "failed to read query results")
	}

	// Collect rows (limit to 100 rows)
	var rows []map[string]any
	limit := 100
	for len(rows) < limit {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			msg.Trace(ctx, "❌ Failed to iterate results: %v", err)
			return nil, goerr.Wrap(err, "failed to iterate results")
		}

		// Convert row to map
		rowMap := make(map[string]any)
		for i, field := range it.Schema {
			rowMap[field.Name] = convertBigQueryValue(row[i])
		}
		rows = append(rows, rowMap)
	}

	queryID := uuid.New().String()

	// Trace: Query completed successfully
	msg.Trace(ctx, "✅ Query completed: %d rows retrieved (query_id: %s)", len(rows), queryID)

	return map[string]any{
		"query_id":         queryID,
		"rows":             rows,
		"total_rows":       len(rows),
		"bytes_processed":  totalBytes,
		"execution_status": "completed",
		"has_more":         len(rows) >= limit,
		"note":             fmt.Sprintf("Limited to %d rows. Query completed successfully.", limit),
	}, nil
}

// getTableSchema retrieves detailed schema information for a BigQuery table
func (t *internalTool) getTableSchema(ctx context.Context, args map[string]any) (map[string]any, error) {
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

	// Create BigQuery client
	client, err := t.newBigQueryClient(ctx, projectID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create BigQuery client")
	}
	defer func() {
		if err := client.Close(); err != nil {
			msg.Trace(ctx, "⚠️ Failed to close BigQuery client: %v", err)
		}
	}()

	// Get table metadata
	table := client.Dataset(datasetID).Table(tableID)
	metadata, err := table.Metadata(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get table metadata",
			goerr.V("project_id", projectID),
			goerr.V("dataset_id", datasetID),
			goerr.V("table_id", tableID))
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
		"schema":     result,
		"project_id": projectID,
		"dataset_id": datasetID,
		"table_id":   tableID,
	}, nil
}

// newBigQueryClient creates a new BigQuery client
func (t *internalTool) newBigQueryClient(ctx context.Context, projectID string) (*bigquery.Client, error) {
	var opts []option.ClientOption
	// Note: Relies on Application Default Credentials (ADC)
	return bigquery.NewClient(ctx, projectID, opts...)
}

// convertBigQueryValue converts BigQuery values to JSON-safe types
func convertBigQueryValue(val bigquery.Value) any {
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case []bigquery.Value:
		// Handle repeated fields (arrays)
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = convertBigQueryValue(item)
		}
		return result
	case map[string]bigquery.Value:
		// Handle nested/struct fields
		result := make(map[string]any)
		for key, item := range v {
			result[key] = convertBigQueryValue(item)
		}
		return result
	default:
		// For primitive types, return as-is
		return val
	}
}
