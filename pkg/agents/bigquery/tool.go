package bigquery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/dustin/go-humanize"
	"github.com/gollem-dev/gollem"
	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/toolset"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// internalTool implements the actual BigQuery operations
type internalTool struct {
	config                    *Config
	projectID                 string
	impersonateServiceAccount string

	tools gollem.ToolSet
}

// Static tool descriptions. The bigquery_query description is built per-config
// in newInternalTool because it interpolates the available tables and scan
// limit; the rest are constant.
const (
	descBigQuerySchema = `Get detailed schema information for a specific BigQuery table.
Use this to understand available fields, data types, and nested structures before constructing queries.
Returns complete schema including nested RECORD fields.`
	descGetRunbook = "Get the full SQL content of a runbook by its ID. Use this when you want to see or adapt a pre-written query template."
)

// Typed inputs for each tool; the schema is inferred from these struct tags.
type bigqueryQueryInput struct {
	SQL         string `json:"sql" required:"true" description:"The SQL query to execute. Must be a valid BigQuery SQL query."`
	Description string `json:"description" description:"Brief description of what this query is trying to achieve (optional, for logging/tracking)"`
}

type bigquerySchemaInput struct {
	ProjectID string `json:"project_id" required:"true" description:"The project ID of the table"`
	DatasetID string `json:"dataset_id" required:"true" description:"The dataset ID of the table"`
	TableID   string `json:"table_id" required:"true" description:"The table ID to get schema for"`
}

type getRunbookInput struct {
	RunbookID string `json:"runbook_id" required:"true" description:"The ID of the runbook to retrieve"`
}

// newInternalTool creates an internalTool and builds its type-safe tool set.
// The get_runbook tool is included only when runbooks are configured, preserving
// the previous mode-dependent tool surface.
func newInternalTool(config *Config, projectID, impersonateServiceAccount string) *internalTool {
	t := &internalTool{
		config:                    config,
		projectID:                 projectID,
		impersonateServiceAccount: impersonateServiceAccount,
	}

	tools := []gollem.Tool{
		gollem.MustNewTool("bigquery_query", buildQueryDescription(config), t.executeQuery),
		gollem.MustNewTool("bigquery_schema", descBigQuerySchema, t.getTableSchema),
	}
	if len(config.Runbooks) > 0 {
		tools = append(tools, gollem.MustNewTool("get_runbook", descGetRunbook, t.getRunbook))
	}
	t.tools = toolset.New(tools...)

	return t
}

// buildQueryDescription renders the bigquery_query tool description, embedding
// the available tables and the configured scan-size limit.
func buildQueryDescription(config *Config) string {
	var sb strings.Builder
	for i, table := range config.Tables {
		fmt.Fprintf(&sb, "\n%d. %s.%s.%s: %s",
			i+1, table.ProjectID, table.DatasetID, table.TableID, table.Description)
	}
	tableDescriptions := sb.String()

	return fmt.Sprintf(`Execute a BigQuery SQL query to retrieve data.
Available tables:%s

Important guidelines:
- Always specify the full table name as project.dataset.table
- Results are limited to 1000 rows max. Queries returning more than 1000 rows will be truncated
- Scan size limit: %s (queries exceeding this will fail)
- For time-based queries, use proper date/time functions and partitioning fields when available
- Select only the fields needed for analysis to minimize scan size (DO NOT use SELECT *)
- Use WHERE clauses to filter data efficiently
- DO NOT use OFFSET for pagination. If results are truncated, use COUNT/GROUP BY to aggregate or add stricter WHERE filters

Best practices:
- Start with schema inspection if unfamiliar with table structure
- Apply time range filters to reduce scan size
- Use LIMIT to restrict rows appropriately`,
		tableDescriptions,
		humanize.Bytes(config.ScanSizeLimit),
	)
}

// ID implements gollem.ToolSet
func (t *internalTool) ID() string {
	return "bigquery_agent"
}

// Specs implements gollem.ToolSet
func (t *internalTool) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return t.tools.Specs(ctx)
}

// Run implements gollem.ToolSet
func (t *internalTool) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	return t.tools.Run(ctx, name, args)
}

func (t *internalTool) executeQuery(ctx context.Context, in bigqueryQueryInput) (map[string]any, error) {
	log := logging.From(ctx)
	log.Debug("Executing BigQuery query")

	sql := in.SQL
	if sql == "" {
		log.Debug("SQL parameter is missing or invalid")
		return nil, goerr.New("sql parameter is required")
	}

	log.Debug("Query SQL prepared", "sql_length", len(sql))
	msg.Trace(ctx, "📝 Executing query: ```\n%s\n````", sql)

	// Determine project ID: CLI flag takes precedence over config
	projectID := t.projectID
	if projectID == "" {
		// Fallback to first table's project ID from config
		if len(t.config.Tables) == 0 {
			log.Debug("No project ID specified and no tables configured")
			return nil, goerr.New("no project ID specified and no tables configured")
		}
		projectID = t.config.Tables[0].ProjectID
		log.Debug("Using project ID from first table", "project_id", projectID)
	} else {
		log.Debug("Using project ID from configuration", "project_id", projectID)
	}

	// Create BigQuery client
	log.Debug("Creating BigQuery client", "project_id", projectID)
	client, err := t.newBigQueryClient(ctx, projectID)
	if err != nil {
		log.Debug("Failed to create BigQuery client", "error", err)
		msg.Trace(ctx, "❌ Failed to create BigQuery client: %v", err)
		return nil, goerr.Wrap(err, "failed to create BigQuery client")
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Debug("Failed to close BigQuery client", "error", err)
			msg.Trace(ctx, "⚠️ Failed to close BigQuery client: %v", err)
		}
	}()
	log.Debug("BigQuery client created successfully")

	// Create query
	q := client.Query(sql)

	// Perform dry run to check scan size
	log.Debug("Performing dry run to check scan size")
	q.DryRun = true
	job, err := q.Run(ctx)
	if err != nil {
		log.Debug("Dry run failed", "error", err)
		msg.Trace(ctx, "❌ Dry run failed: %v", err)
		return nil, goerr.Wrap(err, "failed to dry run query")
	}

	totalBytes := job.LastStatus().Statistics.TotalBytesProcessed
	log.Debug("Dry run completed", "total_bytes", totalBytes, "job_id", job.ID())

	if totalBytes < 0 {
		log.Debug("Invalid negative bytes processed", "bytes", totalBytes)
		msg.Trace(ctx, "❌ Invalid negative bytes processed: %d", totalBytes)
		return nil, goerr.New("invalid negative bytes processed",
			goerr.V("bytes_processed", totalBytes))
	}

	// Check scan size
	log.Debug("Checking scan size",
		"total_bytes", totalBytes,
		"scan_limit", t.config.ScanSizeLimit,
		"total_bytes_human", humanize.Bytes(uint64(totalBytes)),
		"scan_limit_human", humanize.Bytes(t.config.ScanSizeLimit))

	// Check scan size limit
	if totalBytes > 0 && uint64(totalBytes) > t.config.ScanSizeLimit {
		log.Debug("Query scan size exceeds limit",
			"scan_size", totalBytes,
			"scan_limit", t.config.ScanSizeLimit)
		msg.Trace(ctx, "❌ Query scan size exceeds limit")

		return nil, goerr.New("query scan size exceeds limit",
			goerr.V("scan_size", totalBytes),
			goerr.V("scan_limit", t.config.ScanSizeLimit))
	}

	// Execute the actual query
	log.Debug("Executing actual query")
	q.DryRun = false
	job, err = q.Run(ctx)
	if err != nil {
		log.Debug("Query execution failed", "error", err)
		msg.Trace(ctx, "❌ Query execution failed: %v", err)
		return nil, goerr.Wrap(err, "failed to run query")
	}
	log.Debug("Query job submitted", "job_id", job.ID())

	// Wait for job to complete (with timeout)
	timeout := t.config.GetQueryTimeout()
	log.Debug("Waiting for job completion", "job_id", job.ID(), "timeout", timeout)
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	status, err := job.Wait(waitCtx)
	if err != nil {
		log.Debug("Job wait failed", "job_id", job.ID(), "error", err)
		msg.Trace(ctx, "❌ Job wait failed: %v", err)
		return nil, goerr.Wrap(err, "failed to wait for job completion")
	}
	log.Debug("Job wait completed", "job_id", job.ID())

	if err := status.Err(); err != nil {
		log.Debug("Job execution failed", "job_id", job.ID(), "error", err)
		msg.Trace(ctx, "❌ Job execution failed: %v", err)
		return nil, goerr.Wrap(err, "job failed")
	}
	log.Debug("Job completed successfully", "job_id", job.ID())

	// Read results
	log.Debug("Reading query results", "job_id", job.ID())
	it, err := job.Read(ctx)
	if err != nil {
		log.Debug("Failed to read query results", "job_id", job.ID(), "error", err)
		msg.Trace(ctx, "❌ Failed to read query results: %v", err)
		return nil, goerr.Wrap(err, "failed to read query results")
	}
	log.Debug("Result iterator created", "job_id", job.ID(), "schema_fields", len(it.Schema))

	// Collect rows (limit to 1000 rows as a safety cap)
	var rows []map[string]any
	limit := 1000
	log.Debug("Collecting rows", "limit", limit)
	for len(rows) < limit {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			log.Debug("Reached end of results", "rows_collected", len(rows))
			break
		}
		if err != nil {
			log.Debug("Failed to iterate results", "error", err, "rows_collected", len(rows))
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
	log.Debug("Rows collected", "count", len(rows), "has_more", len(rows) >= limit)

	queryID := uuid.New().String()
	log.Debug("Query execution complete", "query_id", queryID, "total_rows", len(rows))

	// Trace: Query completed successfully
	msg.Trace(ctx, "✅ Query completed: %d rows retrieved, scan size: %s (query_id: %s)",
		len(rows), humanize.Bytes(uint64(totalBytes)), queryID)

	hasMore := len(rows) >= limit
	note := "Query completed successfully."
	if hasMore {
		note = fmt.Sprintf("Limited to %d rows. There are more results. Do NOT paginate with OFFSET. Instead, use COUNT/GROUP BY to aggregate the data, or add more specific WHERE filters to narrow the results.", limit)
	}

	return map[string]any{
		"query_id":         queryID,
		"rows":             rows,
		"total_rows":       len(rows),
		"bytes_processed":  totalBytes,
		"execution_status": "completed",
		"has_more":         hasMore,
		"note":             note,
	}, nil
}

// getTableSchema retrieves detailed schema information for a BigQuery table
func (t *internalTool) getTableSchema(ctx context.Context, in bigquerySchemaInput) (map[string]any, error) {
	log := logging.From(ctx)
	log.Debug("Getting BigQuery table schema")

	projectID := in.ProjectID
	if projectID == "" {
		log.Debug("project_id parameter is missing or invalid")
		return nil, goerr.New("project_id parameter is required")
	}
	datasetID := in.DatasetID
	if datasetID == "" {
		log.Debug("dataset_id parameter is missing or invalid")
		return nil, goerr.New("dataset_id parameter is required")
	}
	tableID := in.TableID
	if tableID == "" {
		log.Debug("table_id parameter is missing or invalid")
		return nil, goerr.New("table_id parameter is required")
	}

	log.Debug("Schema request parameters",
		"project_id", projectID,
		"dataset_id", datasetID,
		"table_id", tableID)
	msg.Trace(ctx, "🧐 Retrieving schema... `%s.%s.%s`", projectID, datasetID, tableID)

	// Create BigQuery client
	log.Debug("Creating BigQuery client for schema retrieval", "project_id", projectID)
	client, err := t.newBigQueryClient(ctx, projectID)
	if err != nil {
		log.Debug("Failed to create BigQuery client", "error", err)
		return nil, goerr.Wrap(err, "failed to create BigQuery client")
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Debug("Failed to close BigQuery client", "error", err)
			msg.Trace(ctx, "⚠️ Failed to close BigQuery client: %v", err)
		}
	}()
	log.Debug("BigQuery client created for schema retrieval")

	// Get table metadata
	log.Debug("Fetching table metadata", "dataset", datasetID, "table", tableID)
	table := client.Dataset(datasetID).Table(tableID)
	metadata, err := table.Metadata(ctx)
	if err != nil {
		log.Debug("Failed to get table metadata",
			"error", err,
			"project_id", projectID,
			"dataset_id", datasetID,
			"table_id", tableID)
		return nil, goerr.Wrap(err, "failed to get table metadata",
			goerr.V("project_id", projectID),
			goerr.V("dataset_id", datasetID),
			goerr.V("table_id", tableID))
	}
	log.Debug("Table metadata retrieved", "schema_fields", len(metadata.Schema))

	// Convert schema to JSON
	log.Debug("Converting schema to JSON", "field_count", len(metadata.Schema))
	schemaJSON, err := json.Marshal(metadata.Schema)
	if err != nil {
		log.Debug("Failed to marshal schema to JSON", "error", err)
		return nil, goerr.Wrap(err, "failed to marshal schema to JSON")
	}
	log.Debug("Schema marshaled to JSON", "json_length", len(schemaJSON))

	var result []map[string]any
	if err := json.Unmarshal(schemaJSON, &result); err != nil {
		log.Debug("Failed to unmarshal schema from JSON", "error", err)
		return nil, goerr.Wrap(err, "failed to unmarshal schema from JSON")
	}
	log.Debug("Schema successfully retrieved",
		"project_id", projectID,
		"dataset_id", datasetID,
		"table_id", tableID,
		"field_count", len(result))

	return map[string]any{
		"schema":     result,
		"project_id": projectID,
		"dataset_id": datasetID,
		"table_id":   tableID,
	}, nil
}

// newBigQueryClient creates a new BigQuery client
func (t *internalTool) newBigQueryClient(ctx context.Context, projectID string) (*bigquery.Client, error) {
	log := logging.From(ctx)
	var opts []option.ClientOption

	// If impersonation is configured, use impersonated credentials
	if t.impersonateServiceAccount != "" {
		log.Debug("Using service account impersonation for BigQuery",
			"service_account", t.impersonateServiceAccount)

		ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
			TargetPrincipal: t.impersonateServiceAccount,
			Scopes: []string{
				"https://www.googleapis.com/auth/bigquery",
				"https://www.googleapis.com/auth/cloud-platform",
			},
		})
		if err != nil {
			log.Debug("Failed to create impersonated credentials",
				"error", err,
				"service_account", t.impersonateServiceAccount)
			return nil, goerr.Wrap(err, "failed to create impersonated credentials",
				goerr.V("service_account", t.impersonateServiceAccount))
		}
		opts = append(opts, option.WithTokenSource(ts))
	} else {
		log.Debug("Using Application Default Credentials for BigQuery")
	}

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

// getRunbook retrieves a runbook entry by ID
func (t *internalTool) getRunbook(ctx context.Context, in getRunbookInput) (map[string]any, error) {
	log := logging.From(ctx)
	log.Debug("Getting runbook")

	runbookIDStr := in.RunbookID
	if runbookIDStr == "" {
		log.Debug("runbook_id parameter is missing or invalid")
		return nil, goerr.New("runbook_id parameter is required")
	}

	// Parse runbook ID
	runbookID := types.RunbookID(runbookIDStr)
	log.Debug("Looking up runbook", "runbook_id", runbookID)

	// Find runbook in config
	entry, exists := t.config.Runbooks[runbookID]
	if !exists {
		log.Debug("Runbook not found", "runbook_id", runbookID)
		return nil, goerr.New("runbook not found", goerr.V("runbook_id", runbookID))
	}

	log.Debug("Runbook found", "runbook_id", runbookID, "title", entry.Title)
	msg.Trace(ctx, "📖 Retrieved runbook: %s", entry.Title)

	return map[string]any{
		"runbook_id":  runbookID.String(),
		"title":       entry.Title,
		"description": entry.Description,
		"sql":         entry.SQLContent,
	}, nil
}
