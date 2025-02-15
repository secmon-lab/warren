package bigquery

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/template"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/vertexai/genai"
	"github.com/dustin/go-humanize"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// Action represents a BigQuery action.
// Added impersonationSA field to allow specifying an impersonation service account.
type Action struct {
	projectID        string
	impersonationSA  string
	cfgFile          string
	cfg              bqConfig
	byteLimit        uint64
	maxGenQueryRetry int
	bqFactory        BigQueryClientFactory
}

type bqConfig struct {
	Tables []tableConfig `yaml:"tables"`
	Limit  bqLimit       `yaml:"limit"`
	Retry  bqRetry       `yaml:"retry"`
}

type bqLimit struct {
	Bytes string `yaml:"bytes"`
	Rows  int64  `yaml:"rows"`
}

type tableConfig struct {
	TableID     string `yaml:"id"`
	Description string `yaml:"description"`
}

type bqRetry struct {
	MaxGenQuery int `yaml:"max_gen_query"`
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "bigquery-project-id",
			Usage:       "BigQuery project ID",
			Destination: &x.projectID,
			Category:    "Action",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_PROJECT_ID"),
		},
		&cli.StringFlag{
			Name:        "bigquery-config",
			Usage:       "BigQuery config file",
			Destination: &x.cfgFile,
			Category:    "Action",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_CONFIG"),
		},
		&cli.StringFlag{
			Name:        "bigquery-impersonation-account",
			Usage:       "Impersonation service account email for BigQuery queries",
			Destination: &x.impersonationSA,
			Category:    "Action",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_IMPERSONATION_ACCOUNT"),
		},
	}
}

func (x *Action) Spec() model.ActionSpec {

	tableIDSpec := model.ArgumentSpec{
		Name:        "table_id",
		Type:        "string",
		Description: "Table ID to retrieve data from",
	}

	for _, table := range x.cfg.Tables {
		tableIDSpec.Choices = append(tableIDSpec.Choices, model.ChoiceSpec{
			Value:       table.TableID,
			Description: table.Description,
		})
	}

	return model.ActionSpec{
		Name:        "bigquery",
		Description: "Retrieve log data from BigQuery",
		Args: []model.ArgumentSpec{
			tableIDSpec,
		},
	}
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("project_id", x.projectID),
		slog.String("impersonation_sa", x.impersonationSA),
	)
}

func (x *Action) Configure(ctx context.Context) error {
	if x.projectID == "" {
		return model.ErrActionUnavailable
	}

	fd, err := os.Open(x.cfgFile)
	if err != nil {
		return goerr.Wrap(err, "failed to open config file", goerr.V("file", x.cfgFile))
	}
	defer fd.Close()

	dec := yaml.NewDecoder(fd)
	if err := dec.Decode(&x.cfg); err != nil {
		return goerr.Wrap(err, "failed to decode config file", goerr.V("file", x.cfgFile))
	}

	byteLimit, err := humanize.ParseBytes(x.cfg.Limit.Bytes)
	if err != nil {
		return goerr.Wrap(err, "failed to parse byte limit", goerr.V("byte_limit", x.cfg.Limit.Bytes))
	}
	x.byteLimit = byteLimit

	x.maxGenQueryRetry = x.cfg.Retry.MaxGenQuery
	if x.maxGenQueryRetry < 1 {
		x.maxGenQueryRetry = 1
	}

	if x.bqFactory == nil {
		x.bqFactory = newBigQueryClient
	}

	return nil
}

//go:embed prompt/query.md
var queryPrompt string

type queryResult struct {
	Query string `json:"query"`
}

func (x *Action) Execute(ctx context.Context, slack interfaces.SlackThreadService, ssn interfaces.GenAIChatSession, args model.Arguments) (*model.ActionResult, error) {
	if err := x.Spec().Validate(args); err != nil {
		return nil, err
	}

	fullTableID, ok := args["table_id"].(string)
	if !ok {
		return nil, goerr.New("table_id is required")
	}

	parts := strings.Split(fullTableID, ".")
	if len(parts) != 3 {
		return nil, goerr.New("invalid table_id, expected project.dataset.table", goerr.V("table_id", fullTableID))
	}

	projectID := parts[0]
	datasetID := parts[1]
	tableID := parts[2]

	client, err := x.bqFactory(ctx, projectID, x.impersonationSA)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create bigquery client")
	}
	defer client.Close()

	meta, err := client.GetMetadata(ctx, datasetID, tableID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get table metadata")
	}
	eb := goerr.NewBuilder(goerr.V("table_id", fullTableID))

	var finalQuery string

	prompt, err := generateQuery(fullTableID, meta.Schema, x.cfg.Limit.Rows)
	if err != nil {
		return nil, err
	}

	for i := 0; i < x.maxGenQueryRetry && finalQuery == ""; i++ {
		result, err := x.requestNewQuery(ctx, ssn, prompt)
		if err != nil {
			return nil, err
		}

		eb = eb.With(goerr.V("query", result.Query))
		status, err := client.DryRun(ctx, result.Query)
		if err != nil {
			if err := slack.Reply(ctx, fmt.Sprintf("Failed to run query. Retry...\nQuery: %s\nError: %s", result.Query, err.Error())); err != nil {
				return nil, goerr.Wrap(err, "failed to reply to slack")
			}
			prompt = fmt.Sprintf("Failed to run query. Please try again. The query is: %s\nError: %s", result.Query, err.Error())
			continue
		}

		if status.Statistics.TotalBytesProcessed < 0 || uint64(status.Statistics.TotalBytesProcessed) > x.byteLimit {
			msg := fmt.Sprintf("The query result is too large. Retry...\nQuery: %s\nDry run result: %s\nLimit: %s",
				result.Query,
				humanize.Bytes(uint64(status.Statistics.TotalBytesProcessed)),
				humanize.Bytes(uint64(x.byteLimit)),
			)
			if err := slack.Reply(ctx, msg); err != nil {
				return nil, goerr.Wrap(err, "failed to reply to slack")
			}

			prompt = fmt.Sprintf("The query result is too large. Please try again with a smaller query. The your generated query result is %d bytes, but the limit is %d bytes.", status.Statistics.TotalBytesProcessed, x.byteLimit)
			continue
		}

		finalQuery = result.Query
	}
	if finalQuery == "" {
		return nil, eb.New("failed to generate query")
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	writer := func(row map[string]bigquery.Value) error {
		if err := enc.Encode(row); err != nil {
			return goerr.Wrap(err, "failed to encode row")
		}
		return nil
	}

	if err := slack.AttachFile(ctx, "Sending query to BigQuery", "query.sql", []byte(finalQuery)); err != nil {
		return nil, goerr.Wrap(err, "failed to attach query")
	}

	if err = client.Query(ctx, finalQuery, writer); err != nil {
		return nil, eb.Wrap(err, "failed to execute query")
	}

	return &model.ActionResult{
		Type:    model.ActionResultTypeJSON,
		Data:    buf.String(),
		Message: fmt.Sprintf("Retrieved data from %s", fullTableID),
	}, nil
}

func generateQuery(fullTableID string, schema bigquery.Schema, limit int64) (string, error) {
	rawSchema, err := json.Marshal(schema)
	if err != nil {
		return "", goerr.Wrap(err, "failed to marshal schema")
	}

	queryArgs := map[string]any{
		"table_id": fullTableID,
		"schema":   string(rawSchema),
		"limit":    limit,
	}
	tmpl, err := template.New("query").Parse(queryPrompt)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse query prompt")
	}

	var queryRequest bytes.Buffer
	if err := tmpl.Execute(&queryRequest, queryArgs); err != nil {
		return "", goerr.Wrap(err, "failed to execute query template")
	}

	return queryRequest.String(), nil
}

func (x *Action) requestNewQuery(ctx context.Context, ssn interfaces.GenAIChatSession, prompt string) (*queryResult, error) {
	eb := goerr.NewBuilder(goerr.V("prompt", prompt))

	queryResp, err := ssn.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		return nil, eb.Wrap(err, "failed to send message")
	}
	eb = eb.With(goerr.V("candidates", queryResp.Candidates))

	var result queryResult

	if len(queryResp.Candidates) == 0 {
		return nil, eb.New("no query result")
	}
	if queryResp.Candidates[0].Content == nil {
		return nil, eb.New("no query result")
	}
	if len(queryResp.Candidates[0].Content.Parts) == 0 {
		return nil, eb.New("no query result")
	}

	queryData, ok := queryResp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return nil, eb.New("no query result")
	}
	eb = eb.With(goerr.V("query_data", queryData))

	if err := json.Unmarshal([]byte(queryData), &result); err != nil {
		return nil, eb.Wrap(err, "failed to unmarshal query result")
	}

	if result.Query == "" {
		return nil, eb.New("no query result")
	}

	return &result, nil

}
