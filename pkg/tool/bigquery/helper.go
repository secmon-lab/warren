package bigquery

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
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
	"github.com/secmon-lab/warren/pkg/service/llm"
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
	bigqueryProjectID string
	bigqueryDatasetID string
	bigqueryTableID   string
	tableDescription  string
	scanLimit         string
	outputDir         string
	outputFile        string
}

func subCommandGenerateConfig() *cli.Command {
	var (
		cfg generateConfigConfig
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
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "bigquery-table-id",
				Aliases:     []string{"t"},
				Usage:       "BigQuery table ID in format 'project_id.dataset_id.table_id'",
				Destination: &cfg.bigqueryTableID,
				Sources:     cli.EnvVars("WARREN_BIGQUERY_TABLE_ID"),
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "table-description",
				Aliases:     []string{"desc"},
				Usage:       "Description of the table, what type of data is stored in the table",
				Destination: &cfg.tableDescription,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "scan-limit",
				Usage:       "Scan limit",
				Destination: &cfg.scanLimit,
				Value:       "1GB",
			},
			&cli.StringFlag{
				Name:        "output-dir",
				Usage:       "Output directory (default: current directory)",
				Destination: &cfg.outputDir,
				Value:       ".",
			},
			&cli.StringFlag{
				Name:        "output-file",
				Aliases:     []string{"o"},
				Usage:       "Output filename (default: {project}.{dataset}.{table}.yaml)",
				Destination: &cfg.outputFile,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			// Parse table-id from flag
			if err := parseTableID(cfg.bigqueryTableID, &cfg); err != nil {
				return goerr.Wrap(err, "failed to parse table ID")
			}

			return generateConfigInternal(ctx, cfg)
		},
	}
}

//go:embed prompt/generate_config_query.md
var generateConfigQueryPrompt string

//go:embed prompt/generate_config_schema.md
var generateConfigSchemaPrompt string

// parseTableID parses table ID in format "project_id.dataset_id.table_id"
func parseTableID(tableID string, cfg *generateConfigConfig) error {
	parts := strings.Split(tableID, ".")
	if len(parts) != 3 {
		return goerr.New("table ID must be in format 'project_id.dataset_id.table_id'")
	}

	// Set all parts from the table ID
	cfg.bigqueryProjectID = parts[0]
	cfg.bigqueryDatasetID = parts[1]
	cfg.bigqueryTableID = parts[2]

	return nil
}

// generateOutputPath generates the output file path from config
func generateOutputPath(cfg generateConfigConfig) (string, error) {
	filename := cfg.outputFile
	if filename == "" {
		filename = fmt.Sprintf("%s.%s.%s.yaml", cfg.bigqueryProjectID, cfg.bigqueryDatasetID, cfg.bigqueryTableID)
	}

	// If filename contains template variables, process them
	if strings.Contains(filename, "{") {
		tmpl, err := template.New("filename").Parse(filename)
		if err != nil {
			return "", goerr.Wrap(err, "failed to parse filename template")
		}

		var buf strings.Builder
		err = tmpl.Execute(&buf, map[string]string{
			"project_id": cfg.bigqueryProjectID,
			"dataset_id": cfg.bigqueryDatasetID,
			"table_id":   cfg.bigqueryTableID,
		})
		if err != nil {
			return "", goerr.Wrap(err, "failed to execute filename template")
		}
		filename = buf.String()
	}

	return filepath.Join(cfg.outputDir, filename), nil
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

	scanLimit, err := humanize.ParseBytes(cfg.scanLimit)
	if err != nil {
		return goerr.Wrap(err, "failed to parse scan limit")
	}

	logger.Info("Generating config", "output", outputPath)

	bqClient, err := factory.NewClient(ctx, cfg.bigqueryProjectID)
	if err != nil {
		return err
	}
	defer bqClient.Close()

	llmClient, err := gemini.New(ctx, cfg.geminiProjectID, cfg.geminiLocation)
	if err != nil {
		return err
	}

	tableMetadata, err := bqClient.Dataset(cfg.bigqueryDatasetID).Table(cfg.bigqueryTableID).Metadata(ctx)
	if err != nil {
		return err
	}

	var tableSchema []any
	for _, field := range flattenSchema(tableMetadata.Schema, []string{}) {
		tableSchema = append(tableSchema, field)
	}
	schemaPrompt, err := prompt.Generate(ctx, generateConfigSchemaPrompt, map[string]any{
		"table_schema": tableSchema,
	})
	if err != nil {
		return err
	}
	schemaSummary, err := llm.Summary(ctx, llmClient, schemaPrompt, tableSchema)
	if err != nil {
		return err
	}

	// Use a simplified schema representation to avoid circular reference issues
	outputSchema := `{
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

	queryPrompt, err := prompt.Generate(ctx, generateConfigQueryPrompt, map[string]any{
		"table_description": cfg.tableDescription,
		"schema_summary":    schemaSummary,
		"output_schema":     outputSchema,
		"scan_limit":        cfg.scanLimit,
		"project_id":        cfg.bigqueryProjectID,
		"dataset_id":        cfg.bigqueryDatasetID,
		"table_id":          cfg.bigqueryTableID,
	})
	if err != nil {
		return err
	}

	agent := gollem.New(llmClient,
		gollem.WithSystemPrompt(queryPrompt),
		gollem.WithMessageHook(func(ctx context.Context, msg string) error {
			println(msg)
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
		}),
		gollem.WithLogger(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
	)

	if _, err := agent.Prompt(ctx, "Generate config"); err != nil {
		return err
	}

	return nil
}

type generateConfigTool struct {
	scanLimitStr   string
	scanLimit      uint64
	bigqueryClient BigQueryClient
	outputPath     string
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
			Name:        "generate_config_output",
			Description: "Generate the final YAML configuration file with the analyzed table metadata",
			Parameters: map[string]*gollem.Parameter{
				"config": {
					Type:        gollem.TypeObject,
					Description: "The complete configuration object following the BigQuery Config schema",
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

		q := x.bigqueryClient.Query(query)

		// Perform dry run to check scan size
		q.SetDryRun(true)
		job, err := q.Run(ctx)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to dry run query")
		}
		status, err := job.Wait(ctx)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to wait for dry run job")
		}
		if status.Statistics.TotalBytesProcessed < 0 {
			return nil, goerr.New("invalid negative bytes processed",
				goerr.V("bytes_processed", status.Statistics.TotalBytesProcessed))
		}
		if uint64(status.Statistics.TotalBytesProcessed) > x.scanLimit {
			return nil, goerr.New("query scan size exceeds limit",
				goerr.V("scan_size", status.Statistics.TotalBytesProcessed),
				goerr.V("scan_limit", x.scanLimit))
		}

		// Execute the actual query
		q.SetDryRun(false)
		job, err = q.Run(ctx)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to run query")
		}

		// Wait for the job to complete
		status, err = job.Wait(ctx)
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
				rowMap[field.Name] = row[i]
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

		return map[string]any{
			"rows":       rows[offset:end],
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

		// Convert to BigQuery Config and save as YAML
		if err := x.saveConfigAsYAML(config); err != nil {
			return nil, goerr.Wrap(err, "failed to save config as YAML")
		}

		return map[string]any{
			"status":  "success",
			"message": "Configuration saved successfully",
		}, nil

	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}
}

// Cache to store query results in memory
var queryResultsCache = make(map[string][]map[string]any)

func (x *generateConfigTool) getCachedResults(queryID string) []map[string]any {
	return queryResultsCache[queryID]
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
	if err := os.WriteFile(x.outputPath, yamlData, 0644); err != nil {
		return goerr.Wrap(err, "failed to write YAML file", goerr.V("path", x.outputPath))
	}

	return nil
}
