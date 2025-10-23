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

//go:embed prompt/generate_config.md
var generateConfigPrompt string

// Helper returns the CLI command for BigQuery helper tools
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

type generateConfigInput struct {
	GeminiProjectID  string
	GeminiLocation   string
	BQProjectID      string
	TableProjectID   string
	TableDatasetID   string
	TableTableID     string
	TableDescription string
	ScanLimit        uint64
	ScanLimitStr     string
	OutputDir        string
	OutputFile       string
	ConfigFile       string
}

type bulkConfigEntry struct {
	TableID     string `yaml:"table_id" json:"table_id"`
	Description string `yaml:"description" json:"description"`
}

type bulkConfig struct {
	Tables []bulkConfigEntry `yaml:"tables" json:"tables"`
}

func subCommandGenerateConfig() *cli.Command {
	var (
		cfg     generateConfigInput
		tableID string
	)
	return &cli.Command{
		Name:    "generate-config",
		Aliases: []string{"g"},
		Usage:   "Generate BigQuery table config",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "gemini-project-id",
				Usage:       "Gemini project ID",
				Destination: &cfg.GeminiProjectID,
				Sources:     cli.EnvVars("WARREN_GEMINI_PROJECT_ID"),
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "gemini-location",
				Usage:       "Gemini location",
				Destination: &cfg.GeminiLocation,
				Sources:     cli.EnvVars("WARREN_GEMINI_LOCATION"),
				Value:       "us-central1",
			},
			&cli.StringFlag{
				Name:        "bigquery-project-id",
				Usage:       "BigQuery client project ID",
				Destination: &cfg.BQProjectID,
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
				Usage:       "Description of the table",
				Destination: &cfg.TableDescription,
				Sources:     cli.EnvVars("WARREN_BIGQUERY_TABLE_DESCRIPTION"),
			},
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "Config file for bulk processing (YAML format)",
				Destination: &cfg.ConfigFile,
				Sources:     cli.EnvVars("WARREN_BIGQUERY_BULK_CONFIG"),
			},
			&cli.StringFlag{
				Name:        "scan-limit",
				Usage:       "Scan limit",
				Destination: &cfg.ScanLimitStr,
				Sources:     cli.EnvVars("WARREN_BIGQUERY_SCAN_LIMIT"),
				Value:       "1GB",
			},
			&cli.StringFlag{
				Name:        "output-dir",
				Usage:       "Output directory",
				Destination: &cfg.OutputDir,
				Sources:     cli.EnvVars("WARREN_BIGQUERY_OUTPUT_DIR"),
				Value:       ".",
			},
			&cli.StringFlag{
				Name:        "output-file",
				Aliases:     []string{"o"},
				Usage:       "Output filename",
				Destination: &cfg.OutputFile,
				Sources:     cli.EnvVars("WARREN_BIGQUERY_OUTPUT_FILE"),
				Value:       "{{ .project_id }}.{{ .dataset_id }}.{{ .table_id }}.yaml",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			// Parse scan limit
			scanLimit, err := humanize.ParseBytes(cfg.ScanLimitStr)
			if err != nil {
				return goerr.Wrap(err, "failed to parse scan limit")
			}
			cfg.ScanLimit = scanLimit

			// Validate input
			if cfg.ConfigFile != "" {
				if tableID != "" || cfg.TableDescription != "" {
					return goerr.New("cannot specify both --config and individual table flags")
				}
				return generateBulkConfigs(ctx, cfg)
			}

			if tableID == "" {
				return goerr.New("--bigquery-table-id is required when not using --config")
			}
			if cfg.TableDescription == "" {
				return goerr.New("--table-description is required when not using --config")
			}

			if err := parseTableID(tableID, &cfg); err != nil {
				return goerr.Wrap(err, "failed to parse table ID")
			}

			return generateConfig(ctx, cfg)
		},
	}
}

// parseTableID parses table ID in format "project_id.dataset_id.table_id"
func parseTableID(tableID string, cfg *generateConfigInput) error {
	parts := strings.Split(tableID, ".")
	if len(parts) != 3 {
		return goerr.New("table ID must be in format 'project_id.dataset_id.table_id'")
	}

	cfg.TableProjectID = parts[0]
	cfg.TableDatasetID = parts[1]
	cfg.TableTableID = parts[2]

	return nil
}

// generateOutputPath generates the output file path
func generateOutputPath(cfg generateConfigInput) (string, error) {
	filename := cfg.OutputFile
	if filename == "" {
		filename = fmt.Sprintf("%s.%s.%s.yaml", cfg.TableProjectID, cfg.TableDatasetID, cfg.TableTableID)
	}

	if strings.Contains(filename, "{{") {
		tmpl, err := template.New("filename").Parse(filename)
		if err != nil {
			return "", goerr.Wrap(err, "failed to parse filename template")
		}

		var buf strings.Builder
		err = tmpl.Execute(&buf, map[string]string{
			"project_id": cfg.TableProjectID,
			"dataset_id": cfg.TableDatasetID,
			"table_id":   cfg.TableTableID,
		})
		if err != nil {
			return "", goerr.Wrap(err, "failed to execute filename template")
		}
		filename = buf.String()
	}

	return filepath.Join(cfg.OutputDir, filename), nil
}

// loadBulkConfig loads bulk configuration from YAML file
func loadBulkConfig(filePath string) (*bulkConfig, error) {
	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read bulk config file", goerr.V("path", filePath))
	}

	var config bulkConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, goerr.Wrap(err, "failed to parse bulk config file", goerr.V("path", filePath))
	}

	return &config, nil
}

// generateBulkConfigs processes multiple tables from config file
func generateBulkConfigs(ctx context.Context, cfg generateConfigInput) error {
	bulkCfg, err := loadBulkConfig(cfg.ConfigFile)
	if err != nil {
		return err
	}

	if len(bulkCfg.Tables) == 0 {
		return goerr.New("no tables found in config file")
	}

	logger := logging.From(ctx)
	logger.Info("Processing bulk config", "file", cfg.ConfigFile, "table_count", len(bulkCfg.Tables))

	for i, entry := range bulkCfg.Tables {
		logger.Info("Processing table", "index", i+1, "total", len(bulkCfg.Tables), "table_id", entry.TableID)

		tableCfg := cfg
		tableCfg.TableDescription = entry.Description

		if err := parseTableID(entry.TableID, &tableCfg); err != nil {
			logger.Error("Failed to parse table ID, skipping", "table_id", entry.TableID, "error", err)
			continue
		}

		if err := generateConfig(ctx, tableCfg); err != nil {
			logger.Error("Failed to generate config, skipping", "table_id", entry.TableID, "error", err)
			continue
		}

		logger.Info("Successfully generated config", "table_id", entry.TableID)
	}

	return nil
}

// generateConfig generates configuration for a single table
func generateConfig(ctx context.Context, cfg generateConfigInput) error {
	factory := &DefaultBigQueryClientFactory{}
	return generateConfigWithFactory(ctx, cfg, factory)
}

// generateConfigWithFactory generates configuration with custom factory (for testing)
func generateConfigWithFactory(ctx context.Context, cfg generateConfigInput, factory BigQueryClientFactory) error {
	logger := logging.From(ctx)

	outputPath, err := generateOutputPath(cfg)
	if err != nil {
		return err
	}

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return goerr.Wrap(err, "failed to create output directory", goerr.V("dir", outputDir))
	}

	logger.Info("Generating config", "output", outputPath)

	bqClient, err := factory.NewClient(ctx, cfg.BQProjectID)
	if err != nil {
		return err
	}
	defer func() {
		if err := bqClient.Close(); err != nil {
			logger.Error("failed to close BigQuery client", logging.ErrAttr(err))
		}
	}()

	llmClient, err := gemini.New(ctx, cfg.GeminiProjectID, cfg.GeminiLocation,
		gemini.WithThinkingBudget(0),
	)
	if err != nil {
		return err
	}

	tableMetadata, err := bqClient.Dataset(cfg.TableDatasetID).Table(cfg.TableTableID).Metadata(ctx)
	if err != nil {
		return err
	}

	flattenedFields := flattenSchema(tableMetadata.Schema, []string{})
	logger.Info("Total fields available", "count", len(flattenedFields))

	outputSchema := generateConfigSchema()

	promptText, err := prompt.GenerateWithStruct(ctx, generateConfigPrompt, map[string]any{
		"table_description": cfg.TableDescription,
		"schema_fields":     flattenedFields,
		"total_fields":      len(flattenedFields),
		"output_schema":     outputSchema,
		"scan_limit":        cfg.ScanLimitStr,
		"project_id":        cfg.TableProjectID,
		"dataset_id":        cfg.TableDatasetID,
		"table_id":          cfg.TableTableID,
	})
	if err != nil {
		return err
	}

	agent := gollem.New(llmClient,
		gollem.WithSystemPrompt(promptText),
		gollem.WithLoopLimit(15),
		gollem.WithLogger(logger),
		gollem.WithContentBlockMiddleware(llm.NewCompactionMiddleware(llmClient, logger)),
		gollem.WithContentStreamMiddleware(llm.NewCompactionStreamMiddleware(llmClient)),
		gollem.WithToolSets(&configGeneratorTools{
			bqClient:       bqClient,
			scanLimit:      cfg.ScanLimit,
			tableDatasetID: cfg.TableDatasetID,
			tableTableID:   cfg.TableTableID,
			outputPath:     outputPath,
			metadata:       tableMetadata,
		}),
	)

	if _, err := agent.Execute(ctx, gollem.Text("Generate configuration")); err != nil {
		return err
	}

	return nil
}

// generateConfigSchema creates JSON schema for Config struct
func generateConfigSchema() string {
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

// configGeneratorTools implements gollem.ToolSet
type configGeneratorTools struct {
	bqClient       BigQueryClient
	scanLimit      uint64
	tableDatasetID string
	tableTableID   string
	outputPath     string
	metadata       *bigquery.TableMetadata
}

func (t *configGeneratorTools) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "bigquery_query",
			Description: fmt.Sprintf("Execute SQL query against BigQuery. Performs dry run to check scan size (limit: %s). Only use field names from the provided schema.", humanize.Bytes(t.scanLimit)),
			Parameters: map[string]*gollem.Parameter{
				"query": {
					Type:        gollem.TypeString,
					Description: "The SQL query to execute",
				},
			},
			Required: []string{"query"},
		},
		{
			Name:        "generate_config",
			Description: "Generate the final YAML configuration. This validates the config against the actual schema and saves it to a file.",
			Parameters: map[string]*gollem.Parameter{
				"config": {
					Type:        gollem.TypeObject,
					Description: "The complete configuration object",
				},
			},
			Required: []string{"config"},
		},
	}, nil
}

func (t *configGeneratorTools) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	switch name {
	case "bigquery_query":
		return t.executeBigQueryQuery(ctx, args)
	case "generate_config":
		return t.generateConfigOutput(ctx, args)
	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}
}

func (t *configGeneratorTools) executeBigQueryQuery(ctx context.Context, args map[string]any) (map[string]any, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, goerr.New("query parameter is required")
	}

	q := t.bqClient.Query(query)

	// Dry run to check scan size
	q.SetDryRun(true)
	job, err := q.Run(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to dry run query")
	}

	totalBytes := job.LastStatus().Statistics.TotalBytesProcessed
	if totalBytes < 0 {
		return nil, goerr.New("invalid negative bytes processed", goerr.V("bytes", totalBytes))
	}
	if totalBytes > 0 && uint64(totalBytes) > t.scanLimit {
		return nil, goerr.New("query scan size exceeds limit",
			goerr.V("scan_size", totalBytes),
			goerr.V("scan_limit", t.scanLimit))
	}

	// Execute actual query
	q.SetDryRun(false)
	job, err = q.Run(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to run query")
	}

	status, err := job.Wait(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to wait for job")
	}

	if err := status.Err(); err != nil {
		return nil, goerr.Wrap(err, "job failed")
	}

	iter, err := job.Read(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read results")
	}

	var rows []map[string]any
	for {
		var row []bigquery.Value
		if err := iter.Next(&row); err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to iterate results")
		}

		rowMap := make(map[string]any)
		schema := iter.Schema()
		for i, field := range schema {
			rowMap[field.Name] = convertBigQueryValue(row[i])
		}
		rows = append(rows, rowMap)
	}

	rowsJSON, err := json.Marshal(rows)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal rows")
	}

	return map[string]any{
		"rows_json":  string(rowsJSON),
		"total_rows": len(rows),
	}, nil
}

func (t *configGeneratorTools) generateConfigOutput(ctx context.Context, args map[string]any) (map[string]any, error) {
	config, ok := args["config"].(map[string]any)
	if !ok {
		return nil, goerr.New("config parameter is required")
	}

	configData, err := json.Marshal(config)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal config")
	}

	var bqConfig Config
	if err := json.Unmarshal(configData, &bqConfig); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal config")
	}

	// Validate against schema
	validationResult := ValidateConfigAgainstSchema(&bqConfig, t.metadata)
	if !validationResult.Valid {
		report := formatValidationReport(validationResult)
		return map[string]any{
			"status":  "validation_failed",
			"message": report,
		}, goerr.New("schema validation failed")
	}

	// Save to YAML
	yamlData, err := yaml.Marshal(&bqConfig)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal YAML")
	}

	if err := os.WriteFile(t.outputPath, yamlData, 0600); err != nil {
		return nil, goerr.Wrap(err, "failed to write YAML file", goerr.V("path", t.outputPath))
	}

	return map[string]any{
		"status":  "success",
		"message": "Configuration saved successfully",
		"path":    t.outputPath,
	}, gollem.ErrExitConversation
}
