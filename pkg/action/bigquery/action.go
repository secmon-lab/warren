package bigquery

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/vertexai/genai"
	"github.com/dustin/go-humanize"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

type Action struct {
	projectID        string
	cfgFile          string
	cfg              bqConfig
	byteLimit        int64
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
	x.byteLimit = int64(byteLimit)

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

	client, err := x.bqFactory(ctx, projectID)
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
	var comment string

	for i := 0; i < x.maxGenQueryRetry; i++ {
		result, err := x.getNewQuery(ctx, ssn, fullTableID, meta, comment)
		if err != nil {
			return nil, err
		}
		eb = eb.With(goerr.V("query", result.Query))

		status, err := client.DryRun(ctx, result.Query)
		if err != nil {
			return nil, eb.Wrap(err, "failed to run query")
		}

		if status.Statistics.TotalBytesProcessed > x.byteLimit {
			comment = fmt.Sprintf("The query result is too large. Please try again with a smaller query. The your generated query result is %d bytes, but the limit is %d bytes.", status.Statistics.TotalBytesProcessed, x.byteLimit)
			continue
		}

		finalQuery = result.Query
		break
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

	if err = client.Query(ctx, finalQuery, writer); err != nil {
		return nil, eb.Wrap(err, "failed to execute query")
	}

	return &model.ActionResult{
		Type:    model.ActionResultTypeJSON,
		Data:    buf.String(),
		Message: fmt.Sprintf("Retrieved data from %s", fullTableID),
	}, nil
}

func (x *Action) getNewQuery(ctx context.Context, ssn interfaces.GenAIChatSession, fullTableID string, meta *bigquery.TableMetadata, comment string) (*queryResult, error) {
	queryArgs := map[string]any{
		"table_id": fullTableID,
		"schema":   meta.Schema,
		"limit":    x.cfg.Limit.Rows,
	}
	tmpl, err := template.New("query").Parse(queryPrompt)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to parse query prompt")
	}

	var queryRequest bytes.Buffer
	if err := tmpl.Execute(&queryRequest, queryArgs); err != nil {
		return nil, goerr.Wrap(err, "failed to execute query template")
	}

	eb := goerr.NewBuilder(goerr.V("query", queryRequest.String()))

	queryResp, err := ssn.SendMessage(ctx, genai.Text(comment), genai.Text(queryRequest.String()))
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
