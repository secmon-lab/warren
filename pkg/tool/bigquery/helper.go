package bigquery

import (
	"context"
	_ "embed"
	"fmt"

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
	output            string
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
				Sources:     cli.EnvVars("GEMINI_PROJECT_ID"),
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "gemini-location",
				Usage:       "Gemini location",
				Destination: &cfg.geminiLocation,
				Sources:     cli.EnvVars("GEMINI_LOCATION"),
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "bigquery-project-id",
				Usage:       "BigQuery project ID",
				Destination: &cfg.bigqueryProjectID,
				Sources:     cli.EnvVars("BIGQUERY_PROJECT_ID"),
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "bigquery-dataset-id",
				Usage:       "BigQuery dataset ID",
				Destination: &cfg.bigqueryDatasetID,
				Sources:     cli.EnvVars("BIGQUERY_DATASET_ID"),
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "bigquery-table-id",
				Usage:       "BigQuery table ID",
				Destination: &cfg.bigqueryTableID,
				Sources:     cli.EnvVars("BIGQUERY_TABLE_ID"),
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "table-description",
				Usage:       "Description of the table, what type of data is stored in the table",
				Destination: &cfg.tableDescription,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "scan-limit",
				Usage:       "Scan limit",
				Destination: &cfg.scanLimit,
				DefaultText: "1GB",
			},
			&cli.StringFlag{
				Name:        "output",
				Usage:       "Output file path. Default is {bigquery-project-id}.{bigquery-dataset-id}.{bigquery-table-id}.yaml",
				Destination: &cfg.output,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			return generateConfig(ctx, cfg)
		},
	}
}

//go:embed prompt/generate_config_query.md
var generateConfigQueryPrompt string

//go:embed prompt/generate_config_schema.md
var generateConfigSchemaPrompt string

func generateConfig(ctx context.Context, cfg generateConfigConfig) error {
	logger := logging.From(ctx)

	if cfg.output == "" {
		cfg.output = fmt.Sprintf("%s.%s.%s.yaml", cfg.bigqueryProjectID, cfg.bigqueryDatasetID, cfg.bigqueryTableID)
	}

	scanLimit, err := humanize.ParseBytes(cfg.scanLimit)
	if err != nil {
		return goerr.Wrap(err, "failed to parse scan limit")
	}

	logger.Info("Generating config", "output", cfg.output)

	bqClient, err := bigquery.NewClient(ctx, cfg.bigqueryProjectID)
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

	outputSchema, err := prompt.ToSchema(Config{}).Stringify()
	if err != nil {
		return err
	}

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
		}),
	)

	if _, err := agent.Prompt(ctx, "Generate config"); err != nil {
		return err
	}

	return nil
}

type generateConfigTool struct {
	scanLimitStr   string
	scanLimit      uint64
	bigqueryClient *bigquery.Client
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
		q.DryRun = true
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
		q.DryRun = false
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
			for i, field := range it.Schema {
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
		if l, ok := args["limit"].(int); ok {
			limit = l
		}
		offset := 0
		if o, ok := args["offset"].(int); ok {
			offset = o
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

	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}
}

// Cache to store query results in memory
var queryResultsCache = make(map[string][]map[string]any)

func (x *generateConfigTool) getCachedResults(queryID string) []map[string]any {
	return queryResultsCache[queryID]
}
