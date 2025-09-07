package bigquery

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"cloud.google.com/go/bigquery"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
	"google.golang.org/api/iterator"
	"gopkg.in/yaml.v3"
)

func (x *Action) Helper() *cli.Command {
	return &cli.Command{
		Name:    "bigquery",
		Usage:   "BigQuery tool helper",
		Aliases: []string{"bq"},
		Commands: []*cli.Command{
			subCommandGenerateConfig(),
		},
	}
}

type generateConfigConfig struct {
	geminiProjectID   string
	geminiLocation    string
	bigqueryProjectID string // BigQuery client project ID
	tableProjectID    string // Table project ID (parsed from table ID)
	tableDatasetID    string // Table dataset ID (parsed from table ID)
	tableTableID      string // Table table ID (parsed from table ID)
	tableDescription  string
	scanLimit         string
	outputDir         string
	outputFile        string
	configFile        string // Config file for bulk processing
}

// BulkConfigEntry represents a single table configuration for bulk processing
type BulkConfigEntry struct {
	TableID     string `yaml:"table_id" json:"table_id"`
	Description string `yaml:"description" json:"description"`
}

// BulkConfig represents the configuration file format for bulk processing
type BulkConfig struct {
	Tables []BulkConfigEntry `yaml:"tables" json:"tables"`
}

func subCommandGenerateConfig() *cli.Command {
	var (
		cfg     generateConfigConfig
		tableID string // temporary variable for parsing
	)
	return &cli.Command{
		Name:    "generate-config",
		Aliases: []string{"g"},
		Usage:   "Generate BigQuery table config",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "gemini-project-id",
				Usage:       "Gemini project ID",
				Destination: &cfg.geminiProjectID,
				Sources:     cli.EnvVars("WARREN_GEMINI_PROJECT_ID"),
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "gemini-location",
				Usage:       "Gemini location",
				Destination: &cfg.geminiLocation,
				Sources:     cli.EnvVars("WARREN_GEMINI_LOCATION"),
				Value:       "us-central1",
			},
			&cli.StringFlag{
				Name:        "bigquery-project-id",
				Usage:       "BigQuery client project ID",
				Destination: &cfg.bigqueryProjectID,
				Sources:     cli.EnvVars("WARREN_BIGQUERY_PROJECT_ID"),
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "bigquery-table-id",
				Aliases:     []string{"t"},
				Usage:       "BigQuery table ID in format 'project_id.dataset_id.table_id'",
				Destination: &tableID,
				Sources:     cli.EnvVars("WARREN_BIGQUERY_TABLE_ID"),
			},
			&cli.StringFlag{
				Name:        "table-description",
				Aliases:     []string{"d"},
				Usage:       "Description of the table, what type of data is stored in the table",
				Destination: &cfg.tableDescription,
				Sources:     cli.EnvVars("WARREN_BIGQUERY_TABLE_DESCRIPTION"),
			},
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "Config file containing tables and descriptions for bulk processing (YAML format)",
				Destination: &cfg.configFile,
				Sources:     cli.EnvVars("WARREN_BIGQUERY_BULK_CONFIG"),
			},
			&cli.StringFlag{
				Name:        "scan-limit",
				Usage:       "Scan limit",
				Destination: &cfg.scanLimit,
				Sources:     cli.EnvVars("WARREN_BIGQUERY_SCAN_LIMIT"),
				Value:       "1GB",
			},
			&cli.StringFlag{
				Name:        "output-dir",
				Usage:       "Output directory (default: current directory)",
				Destination: &cfg.outputDir,
				Sources:     cli.EnvVars("WARREN_BIGQUERY_OUTPUT_DIR"),
				Value:       ".",
			},
			&cli.StringFlag{
				Name:        "output-file",
				Aliases:     []string{"o"},
				Usage:       "Output filename",
				Destination: &cfg.outputFile,
				Sources:     cli.EnvVars("WARREN_BIGQUERY_OUTPUT_FILE"),
				Value:       "{{ .project_id }}.{{ .dataset_id }}.{{ .table_id }}.yaml",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			// Validate input: either config file or table-id + description required
			if cfg.configFile != "" {
				// Bulk processing mode
				if tableID != "" || cfg.tableDescription != "" {
					return goerr.New("cannot specify both --config and individual table flags (--bigquery-table-id, --table-description)")
				}
				return generateBulkConfigs(ctx, cfg)
			} else {
				// Single table mode
				if tableID == "" {
					return goerr.New("--bigquery-table-id is required when not using --config")
				}
				if cfg.tableDescription == "" {
					return goerr.New("--table-description is required when not using --config")
				}

				// Parse table-id from flag
				if err := parseTableID(tableID, &cfg); err != nil {
					return goerr.Wrap(err, "failed to parse table ID")
				}

				return generateConfigInternal(ctx, cfg)
			}
		},
	}
}

//go:embed prompt/generate_config_query.md
var generateConfigQueryPrompt string

// parseTableID parses table ID in format "project_id.dataset_id.table_id"
func parseTableID(tableID string, cfg *generateConfigConfig) error {
	parts := strings.Split(tableID, ".")
	if len(parts) != 3 {
		return goerr.New("table ID must be in format 'project_id.dataset_id.table_id'")
	}

	// Set table parts from the table ID (completely separate from BigQuery client project)
	cfg.tableProjectID = parts[0]
	cfg.tableDatasetID = parts[1]
	cfg.tableTableID = parts[2]

	return nil
}

// generateOutputPath generates the output file path from config
func generateOutputPath(cfg generateConfigConfig) (string, error) {
	filename := cfg.outputFile
	if filename == "" {
		filename = fmt.Sprintf("%s.%s.%s.yaml", cfg.tableProjectID, cfg.tableDatasetID, cfg.tableTableID)
	}

	// If filename contains template variables, process them
	if strings.Contains(filename, "{") {
		tmpl, err := template.New("filename").Parse(filename)
		if err != nil {
			return "", goerr.Wrap(err, "failed to parse filename template")
		}

		var buf strings.Builder
		err = tmpl.Execute(&buf, map[string]string{
			"project_id": cfg.tableProjectID,
			"dataset_id": cfg.tableDatasetID,
			"table_id":   cfg.tableTableID,
		})
		if err != nil {
			return "", goerr.Wrap(err, "failed to execute filename template")
		}
		filename = buf.String()
	}

	return filepath.Join(cfg.outputDir, filename), nil
}

// generateConfigSchema creates a JSON schema for the Config struct
//
// This function addresses the circular reference issue in ColumnConfig.Fields ([]ColumnConfig)
// which would cause infinite recursion with pure reflection-based schema generation.
//
// While we could use prompt.ToSchema() for non-circular parts, we maintain a static schema here to:
// 1. Avoid runtime errors from circular references
// 2. Ensure schema consistency and reliability
// 3. Maintain backward compatibility with existing LLM prompts
//
// TODO: Consider implementing a more sophisticated schema generator that can handle
// circular references by using JSON Schema's $ref mechanism in the future.
func generateConfigSchema() string {
	// This schema must be kept in sync with the Config, ColumnConfig, and PartitioningConfig structs
	return `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "dataset_id": { "type": "string" },
    "table_id": { "type": "string" },
    "description": { "type": "string" },
    "columns": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": { "type": "string" },
          "description": { "type": "string" },
          "value_example": { "type": "string" },
          "type": { "type": "string" },
          "fields": {
            "type": "array",
            "items": { "type": "object" }
          }
        }
      }
    },
    "partitioning": {
      "type": "object",
      "properties": {
        "field": { "type": "string" },
        "type": { "type": "string" },
        "time_unit": { "type": "string" }
      }
    }
  }
}`
}

// SchemaValidationResult holds the result of schema validation
type SchemaValidationResult struct {
	Valid  bool
	Issues []SchemaValidationIssue
}

type SchemaValidationIssue struct {
	Type         string // "field_not_found", "type_mismatch", "nested_field_not_found"
	FieldPath    string
	ExpectedType string
	ActualType   string
	Message      string
}

// ValidateConfigAgainstSchema validates the generated config against the actual BigQuery table schema
func ValidateConfigAgainstSchema(config *Config, tableMetadata *bigquery.TableMetadata) SchemaValidationResult {
	result := SchemaValidationResult{
		Valid:  true,
		Issues: []SchemaValidationIssue{},
	}

	// Create a map of actual schema fields for quick lookup
	actualFields := BuildSchemaFieldMap(tableMetadata.Schema, "")

	// Validate each column in the config
	for _, column := range config.Columns {
		validateColumnConfig(column, actualFields, &result, "")
	}

	if len(result.Issues) > 0 {
		result.Valid = false
	}

	return result
}

// BuildSchemaFieldMap creates a flat map of all fields (including nested) from BigQuery schema
func BuildSchemaFieldMap(schema bigquery.Schema, prefix string) map[string]*bigquery.FieldSchema {
	fieldMap := make(map[string]*bigquery.FieldSchema)

	for _, field := range schema {
		fieldName := field.Name
		if prefix != "" {
			fieldName = prefix + "." + field.Name
		}

		fieldMap[fieldName] = field

		// Handle nested RECORD fields
		if field.Type == bigquery.RecordFieldType && len(field.Schema) > 0 {
			nestedFields := BuildSchemaFieldMap(field.Schema, fieldName)
			for k, v := range nestedFields {
				fieldMap[k] = v
			}
		}
	}

	return fieldMap
}

// validateColumnConfig validates a single column config against the actual schema
func validateColumnConfig(column ColumnConfig, actualFields map[string]*bigquery.FieldSchema, result *SchemaValidationResult, prefix string) {
	fieldPath := column.Name
	if prefix != "" {
		fieldPath = prefix + "." + column.Name
	}

	actualField, exists := actualFields[fieldPath]
	if !exists {
		result.Issues = append(result.Issues, SchemaValidationIssue{
			Type:      "field_not_found",
			FieldPath: fieldPath,
			Message:   fmt.Sprintf("Field '%s' does not exist in the actual table schema", fieldPath),
		})
		return
	}

	// Validate data type
	expectedType := strings.ToUpper(column.Type)
	actualType := string(actualField.Type)
	if expectedType != actualType {
		result.Issues = append(result.Issues, SchemaValidationIssue{
			Type:         "type_mismatch",
			FieldPath:    fieldPath,
			ExpectedType: expectedType,
			ActualType:   actualType,
			Message:      fmt.Sprintf("Field '%s' type mismatch: config has '%s', actual schema has '%s'", fieldPath, expectedType, actualType),
		})
	}

	// Validate nested fields for RECORD types
	if column.Type == "RECORD" && len(column.Fields) > 0 {
		for _, nestedField := range column.Fields {
			validateColumnConfig(nestedField, actualFields, result, fieldPath)
		}
	}
}

// formatValidationReport creates a formatted report of validation issues
func formatValidationReport(result SchemaValidationResult) string {
	if result.Valid {
		return "✅ Schema validation passed. All fields in the configuration match the actual table schema."
	}

	var report strings.Builder
	report.WriteString("❌ SCHEMA VALIDATION FAILED\n\n")
	report.WriteString(fmt.Sprintf("Found %d issue(s) with the generated configuration:\n\n", len(result.Issues)))

	// Group issues by type
	fieldNotFound := []SchemaValidationIssue{}
	typeMismatch := []SchemaValidationIssue{}
	nestedIssues := []SchemaValidationIssue{}

	for _, issue := range result.Issues {
		switch issue.Type {
		case "field_not_found":
			fieldNotFound = append(fieldNotFound, issue)
		case "type_mismatch":
			typeMismatch = append(typeMismatch, issue)
		case "nested_field_not_found":
			nestedIssues = append(nestedIssues, issue)
		}
	}

	if len(fieldNotFound) > 0 {
		report.WriteString("🔍 FIELDS NOT FOUND IN ACTUAL SCHEMA:\n")
		for i, issue := range fieldNotFound {
			report.WriteString(fmt.Sprintf("  %d. %s\n", i+1, issue.Message))
		}
		report.WriteString("\n")
	}

	if len(typeMismatch) > 0 {
		report.WriteString("🔄 DATA TYPE MISMATCHES:\n")
		for i, issue := range typeMismatch {
			report.WriteString(fmt.Sprintf("  %d. %s\n", i+1, issue.Message))
		}
		report.WriteString("\n")
	}

	if len(nestedIssues) > 0 {
		report.WriteString("🏗️ NESTED FIELD ISSUES:\n")
		for i, issue := range nestedIssues {
			report.WriteString(fmt.Sprintf("  %d. %s\n", i+1, issue.Message))
		}
		report.WriteString("\n")
	}

	report.WriteString("REQUIRED ACTIONS:\n")
	report.WriteString("1. Remove non-existent fields from your configuration\n")
	report.WriteString("2. Correct data type mismatches\n")
	report.WriteString("3. Verify nested field structures against the actual schema\n")
	report.WriteString("4. Re-run field validation queries to confirm field existence\n")
	report.WriteString("5. Generate a new configuration with only validated fields\n\n")
	report.WriteString("Please fix these issues and regenerate the configuration.")

	return report.String()
}

// generateBulkConfigs processes multiple tables from a config file
func generateBulkConfigs(ctx context.Context, cfg generateConfigConfig) error {
	// Load bulk config from file
	bulkConfig, err := loadBulkConfig(cfg.configFile)
	if err != nil {
		return goerr.Wrap(err, "failed to load bulk config file")
	}

	if len(bulkConfig.Tables) == 0 {
		return goerr.New("no tables found in config file")
	}

	logger := logging.From(ctx)
	logger.Info("Processing bulk config", "file", cfg.configFile, "table_count", len(bulkConfig.Tables))

	// Process each table
	for i, entry := range bulkConfig.Tables {
		logger.Info("Processing table", "index", i+1, "total", len(bulkConfig.Tables), "table_id", entry.TableID)

		// Create a copy of cfg for this table
		tableCfg := cfg
		tableCfg.tableDescription = entry.Description

		// Parse table ID
		if err := parseTableID(entry.TableID, &tableCfg); err != nil {
			logger.Error("Failed to parse table ID, skipping", "table_id", entry.TableID, "error", err)
			continue
		}

		// Generate config for this table
		if err := generateConfigInternal(ctx, tableCfg); err != nil {
			logger.Error("Failed to generate config for table, skipping", "table_id", entry.TableID, "error", err)
			continue
		}

		logger.Info("Successfully generated config", "table_id", entry.TableID)
	}

	return nil
}

// loadBulkConfig loads the bulk configuration from a YAML file
func loadBulkConfig(filePath string) (*BulkConfig, error) {
	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read bulk config file", goerr.V("path", filePath))
	}

	var config BulkConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, goerr.Wrap(err, "failed to parse bulk config file", goerr.V("path", filePath))
	}

	return &config, nil
}

func generateConfigInternal(ctx context.Context, cfg generateConfigConfig) error {
	factory := &DefaultBigQueryClientFactory{}
	return generateConfigWithFactoryInternal(ctx, cfg, factory)
}

func generateConfigWithFactoryInternal(ctx context.Context, cfg generateConfigConfig, factory BigQueryClientFactory) error {
	logger := logging.From(ctx)

	// Generate output path
	outputPath, err := generateOutputPath(cfg)
	if err != nil {
		return goerr.Wrap(err, "failed to generate output path")
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return goerr.Wrap(err, "failed to create output directory", goerr.V("dir", outputDir))
	}

	scanLimit, err := humanize.ParseBytes(cfg.scanLimit)
	if err != nil {
		return goerr.Wrap(err, "failed to parse scan limit")
	}

	logger.Info("Generating config", "output", outputPath)

	bqClient, err := factory.NewClient(ctx, cfg.bigqueryProjectID)
	if err != nil {
		return err
	}
	defer func() {
		if err := bqClient.Close(); err != nil {
			logger.Error("failed to close BigQuery client", logging.ErrAttr(err))
		}
	}()

	llmClient, err := gemini.New(ctx, cfg.geminiProjectID, cfg.geminiLocation,
		gemini.WithThinkingBudget(0), // Disable thinking feature
	)
	if err != nil {
		return err
	}

	tableMetadata, err := bqClient.Dataset(cfg.tableDatasetID).Table(cfg.tableTableID).Metadata(ctx)
	if err != nil {
		return err
	}

	// Get flattened schema fields directly
	flattenedFields := flattenSchema(tableMetadata.Schema, []string{})

	logger.Info("Total fields available", "count", len(flattenedFields))

	// Analyze security fields to provide focused guidance
	securityFields := AnalyzeSecurityFields(tableMetadata.Schema)
	securityPrompt := ""
	if len(securityFields) > 0 {
		securityPrompt = generateSecurityPrompt(securityFields)
	}

	// Generate JSON schema from the Config struct
	outputSchema := generateConfigSchema()

	queryPrompt, err := prompt.GenerateWithStruct(ctx, generateConfigQueryPrompt, map[string]any{
		"table_description":     cfg.tableDescription,
		"schema_fields":         flattenedFields, // Use all fields - LLM will process internally
		"total_fields_count":    len(flattenedFields),
		"used_fields_count":     len(flattenedFields),
		"security_analysis":     securityPrompt,
		"security_fields_count": len(securityFields),
		"output_schema":         outputSchema,
		"scan_limit":            cfg.scanLimit,
		"project_id":            cfg.tableProjectID,
		"dataset_id":            cfg.tableDatasetID,
		"table_id":              cfg.tableTableID,
	})
	if err != nil {
		return err
	}

	agent := gollem.New(llmClient,
		gollem.WithSystemPrompt(queryPrompt),
		gollem.WithLoopLimit(20), // Increase to 20 iterations to handle complex schemas and validation retries
		gollem.WithLogger(logging.From(ctx)),
		gollem.WithMessageHook(func(ctx context.Context, msg string) error {
			println("💬", msg)
			return nil
		}),
		gollem.WithToolErrorHook(func(ctx context.Context, err error, tool gollem.FunctionCall) error {
			println("❌", err.Error())
			// For schema validation errors, provide more guidance
			if strings.Contains(err.Error(), "schema validation failed") {
				println("💡 Hint: Only use fields from the provided schema list. Do not add or guess field names.")
			}
			return nil
		}),
		gollem.WithToolRequestHook(func(ctx context.Context, tool gollem.FunctionCall) error {
			println("⚡", tool.Name)
			return nil
		}),
		gollem.WithToolSets(&generateConfigTool{
			bigqueryClient: bqClient,
			scanLimitStr:   cfg.scanLimit,
			scanLimit:      scanLimit,
			outputPath:     outputPath,
			tableDatasetID: cfg.tableDatasetID,
			tableTableID:   cfg.tableTableID,
		}),
	)

	if err := agent.Execute(ctx, "Generate config"); err != nil {
		return err
	}

	return nil
}

type generateConfigTool struct {
	scanLimitStr   string
	scanLimit      uint64
	bigqueryClient BigQueryClient
	outputPath     string
	tableDatasetID string // Added to fetch metadata when needed
	tableTableID   string // Added to fetch metadata when needed
}

func (x *generateConfigTool) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "bigquery_query",
			Description: bigqueryQueryPrompt(x.scanLimitStr),
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
			Description: "Get the results of a previously executed query. Returns rows as JSON string in 'rows_json' field due to Vertex AI type limitations.",
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
			Name:        "generate_config_output",
			Description: "Generate the final YAML configuration file with the analyzed table metadata. MUST include complete nested field structures for all RECORD types - include ALL nested fields that exist in the schema, with proper hierarchy and 3-4 levels of nesting when available. If validation fails, the tool will return specific invalid fields that need to be removed. You should then immediately retry with a corrected configuration that removes only the invalid fields mentioned in the response.",
			Parameters: map[string]*gollem.Parameter{
				"config": {
					Type:        gollem.TypeObject,
					Description: "The complete configuration object following the BigQuery Config schema. Must include complete nested structures for all RECORD fields with ALL their nested children.",
				},
			},
			Required: []string{"config"},
		},
	}, nil
}

func (x *generateConfigTool) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	switch name {
	case "bigquery_query":
		query, ok := args["query"].(string)
		if !ok {
			return nil, goerr.New("query parameter is required")
		}

		println("🔍", query)
		q := x.bigqueryClient.Query(query)

		// Perform dry run to check scan size
		q.SetDryRun(true)
		job, err := q.Run(ctx)
		if err != nil {
			// Provide helpful error message for field not found errors during dry run
			if strings.Contains(err.Error(), "field") || strings.Contains(err.Error(), "column") {
				return nil, goerr.Wrap(err, "SQL query validation failed - likely invalid field name. Only use fields from the provided schema_fields list. Try using SELECT * LIMIT 10 first to see available fields.")
			}
			return nil, goerr.Wrap(err, "failed to dry run query")
		}
		totalBytes := job.LastStatus().Statistics.TotalBytesProcessed
		if totalBytes < 0 {
			return nil, goerr.New("invalid negative bytes processed",
				goerr.V("bytes_processed", totalBytes))
		}
		// Safe conversion after negative check
		if totalBytes > 0 && uint64(totalBytes) > x.scanLimit {
			return nil, goerr.New("query scan size exceeds limit",
				goerr.V("scan_size", job.LastStatus().Statistics.TotalBytesProcessed),
				goerr.V("scan_limit", x.scanLimit))
		}

		// Execute the actual query
		q.SetDryRun(false)
		job, err = q.Run(ctx)
		if err != nil {
			// Provide helpful error message for field not found errors
			if strings.Contains(err.Error(), "field") || strings.Contains(err.Error(), "column") {
				return nil, goerr.Wrap(err, "SQL query failed - likely invalid field name. Only use fields from the provided schema_fields list. Try using SELECT * LIMIT 10 first to see available fields.")
			}
			return nil, goerr.Wrap(err, "failed to run query")
		}

		// Wait for the job to complete
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

		var rows []map[string]any
		for {
			var row []bigquery.Value
			if err := it.Next(&row); err != nil {
				if err == iterator.Done {
					break
				}
				return nil, goerr.Wrap(err, "failed to iterate results")
			}

			rowMap := make(map[string]any)
			schema := it.Schema()
			for i, field := range schema {
				// Convert BigQuery values to JSON-safe types for Vertex AI
				rowMap[field.Name] = convertBigQueryValue(row[i])
			}
			rows = append(rows, rowMap)
		}

		// Cache query results
		queryID := uuid.New().String()
		queryResultsCache[queryID] = rows

		return map[string]any{
			"query_id":   queryID,
			"total_rows": len(rows),
		}, nil

	case "bigquery_result":
		queryID, ok := args["query_id"].(string)
		if !ok {
			return nil, goerr.New("query_id parameter is required")
		}
		limit := 100
		if l, ok := args["limit"].(float64); ok {
			limit = int(l)
		} else if args["limit"] != nil {
			return nil, goerr.New("invalid limit parameter type",
				goerr.V("type", fmt.Sprintf("%T", args["limit"])),
				goerr.V("value", args["limit"]))
		}
		offset := 0
		if o, ok := args["offset"].(float64); ok {
			offset = int(o)
		} else if args["offset"] != nil {
			return nil, goerr.New("invalid offset parameter type",
				goerr.V("type", fmt.Sprintf("%T", args["offset"])),
				goerr.V("value", args["offset"]))
		}

		// Get results from memory cache
		rows := x.getCachedResults(queryID)
		if rows == nil {
			return nil, goerr.New("query results not found", goerr.V("query_id", queryID))
		}

		// Handle pagination
		end := offset + limit
		if end > len(rows) {
			end = len(rows)
		}

		// Convert rows to JSON string to avoid Vertex AI type conversion issues
		paginatedRows := rows[offset:end]
		rowsJSON, err := json.Marshal(paginatedRows)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to marshal rows to JSON")
		}

		return map[string]any{
			"rows_json":  string(rowsJSON),
			"total_rows": len(rows),
			"limit":      limit,
			"offset":     offset,
			"has_more":   end < len(rows),
		}, nil

	case "generate_config_output":
		config, ok := args["config"].(map[string]any)
		if !ok {
			return nil, goerr.New("config parameter is required and must be an object")
		}

		// Convert to BigQuery Config for validation
		configData, err := json.Marshal(config)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to marshal config for validation")
		}

		var bqConfig Config
		if err := json.Unmarshal(configData, &bqConfig); err != nil {
			return nil, goerr.Wrap(err, "failed to unmarshal config to BigQuery Config")
		}

		// Validate configuration against actual schema
		tableMetadata, err := x.getTableMetadata(ctx)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get table metadata for validation")
		}

		validationResult := ValidateConfigAgainstSchema(&bqConfig, tableMetadata)

		if !validationResult.Valid {
			// Print validation report
			report := formatValidationReport(validationResult)
			println("\n" + report + "\n")

			// Provide specific guidance for common errors
			fieldNotFoundCount := 0
			invalidFields := []string{}
			for _, issue := range validationResult.Issues {
				if issue.Type == "field_not_found" {
					fieldNotFoundCount++
					invalidFields = append(invalidFields, issue.FieldPath)
				}
			}

			if fieldNotFoundCount > 0 {
				println("🔧 SOLUTION: Remove the non-existent fields listed above and retry.")
				println("📋 REMINDER: Only use fields from the provided schema_fields list.")
				println("❌ DO NOT add fields that are not explicitly listed in the schema.")
			}

			// Instead of failing, provide a success response with guidance for retry
			return map[string]any{
				"status":             "validation_failed_retry_needed",
				"validation_report":  report,
				"field_errors_count": fieldNotFoundCount,
				"invalid_fields":     invalidFields,
				"message":            "Configuration has validation errors. Please remove the invalid fields and generate a corrected configuration.",
				"retry_instruction":  "Remove these invalid fields and call generate_config_output again with the corrected configuration.",
				"success":            false,
			}, nil // Return nil error to allow LLM to continue and retry

		}

		// If validation passes, save the config
		if err := x.saveConfigAsYAML(config); err != nil {
			return nil, goerr.Wrap(err, "failed to save config as YAML")
		}

		return map[string]any{
			"status":            "success",
			"message":           "✅ Configuration validated and saved successfully",
			"validation_status": "passed",
			"success":           true,
		}, gollem.ErrExitConversation

	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}
}

// Cache to store query results in memory
var queryResultsCache = make(map[string][]map[string]any)

func (x *generateConfigTool) getCachedResults(queryID string) []map[string]any {
	return queryResultsCache[queryID]
}

func (x *generateConfigTool) getTableMetadata(ctx context.Context) (*bigquery.TableMetadata, error) {
	return x.bigqueryClient.Dataset(x.tableDatasetID).Table(x.tableTableID).Metadata(ctx)
}

func (x *generateConfigTool) saveConfigAsYAML(config map[string]any) error {
	// Convert map to BigQuery Config struct
	configData, err := json.Marshal(config)
	if err != nil {
		return goerr.Wrap(err, "failed to marshal config")
	}

	var bqConfig Config
	if err := json.Unmarshal(configData, &bqConfig); err != nil {
		return goerr.Wrap(err, "failed to unmarshal config to BigQuery Config")
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(&bqConfig)
	if err != nil {
		return goerr.Wrap(err, "failed to marshal config to YAML")
	}

	// Write to file
	if err := os.WriteFile(x.outputPath, yamlData, 0600); err != nil {
		return goerr.Wrap(err, "failed to write YAML file", goerr.V("path", x.outputPath))
	}

	return nil
}
