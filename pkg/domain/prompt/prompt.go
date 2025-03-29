package prompt

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"math/rand/v2"
	"text/template"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func stringify(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return "", goerr.Wrap(err, "failed to marshal", goerr.V("data", v))
	}
	return buf.String(), nil
}

//go:embed templates/init.md
var initTemplate string

func BuildInitPrompt(ctx context.Context, alert any, maxRetry int) (string, error) {
	tmpl, err := template.New("init").Parse(initTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	rawAlert, err := stringify(alert)
	if err != nil {
		return "", err
	}

	input := map[string]any{
		"alert":     string(rawAlert),
		"max_retry": maxRetry,
		"lang":      lang.From(ctx).Name(),
	}

	var result bytes.Buffer
	if err := tmpl.ExecuteTemplate(&result, "init", input); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	return result.String(), nil
}

//go:embed templates/aggregate.md
var aggregateTemplate string

type AggregatePromptResult struct {
	AlertID string `json:"alert_id"`
}

func BuildAggregatePrompt(ctx context.Context, newAlert alert.Alert, candidates alert.Alerts) (string, error) {
	tmpl, err := template.New("aggregate").Parse(aggregateTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	rawNewAlert, err := stringify(newAlert)
	if err != nil {
		return "", err
	}

	rawCandidates := []string{}
	for _, candidate := range candidates {
		rawCandidate, err := stringify(candidate)
		if err != nil {
			return "", err
		}
		rawCandidates = append(rawCandidates, rawCandidate)
	}

	schema, err := generateSchema(AggregatePromptResult{}).Stringify()
	if err != nil {
		return "", err
	}

	input := map[string]any{
		"new":        rawNewAlert,
		"candidates": rawCandidates,
		"schema":     schema,
		"lang":       lang.From(ctx).Name(),
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "aggregate", input); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	return buf.String(), nil
}

//go:embed templates/finding.md
var findingTemplate string

func BuildFindingPrompt(ctx context.Context) (string, error) {
	tmpl, err := template.New("finding").Parse(findingTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	schema, err := generateSchema(alert.Finding{}).Stringify()
	if err != nil {
		return "", err
	}

	input := map[string]any{
		"schema": schema,
		"lang":   lang.From(ctx).Name(),
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "finding", input); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	return buf.String(), nil
}

//go:embed templates/meta.md
var metaTemplate string

type MetaPromptResult struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Attrs       []alert.Attribute `json:"attrs"`
}

func BuildMetaPrompt(ctx context.Context, alert alert.Alert) (string, error) {
	tmpl, err := template.New("meta").Parse(metaTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	schema, err := generateSchema(MetaPromptResult{}).Stringify()
	if err != nil {
		return "", err
	}

	rawAlert, err := stringify(alert)
	if err != nil {
		return "", err
	}

	input := map[string]any{
		"alert":  rawAlert,
		"schema": schema,
		"lang":   lang.From(ctx).Name(),
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "meta", input); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	return buf.String(), nil
}

//go:embed templates/ignore_policy.md
var ignorePolicyTemplate string

type IgnorePolicyPromptResult struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Policy      map[string]string `json:"policy"`
}

func BuildIgnorePolicyPrompt(ctx context.Context, contents policy.Contents, alerts alert.Alerts, note string) (string, error) {
	tmpl, err := template.New("ignore_policy").Parse(ignorePolicyTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	outputSchema, err := generateSchema(IgnorePolicyPromptResult{}).Stringify()
	if err != nil {
		return "", err
	}

	rawPolicy, err := stringify(contents)
	if err != nil {
		return "", err
	}

	input := map[string]any{
		"note":   note,
		"policy": rawPolicy,
		"alerts": alerts,
		"output": outputSchema,
		"lang":   lang.From(ctx).Name(),
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "ignore_policy", input); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	return buf.String(), nil
}

//go:embed templates/test_data_readme.md
var testDataReadmeTemplate string

type TestDataReadmePromptResult struct {
	Content string `json:"content"`
}

func BuildTestDataReadmePrompt(ctx context.Context, action string, alerts alert.Alerts) (string, error) {
	tmpl, err := template.New("test_data_readme").Parse(testDataReadmeTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	rawAlerts := []string{}
	for _, alert := range alerts {
		rawAlert, err := stringify(alert)
		if err != nil {
			return "", err
		}
		rawAlerts = append(rawAlerts, rawAlert)
	}

	schema, err := generateSchema(TestDataReadmePromptResult{}).Stringify()
	if err != nil {
		return "", err
	}

	input := map[string]any{
		"alerts": rawAlerts,
		"schema": schema,
		"action": action,
		"lang":   lang.From(ctx).Name(),
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "test_data_readme", input); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	return buf.String(), nil
}

//go:embed templates/filter_query.md
var filterQueryTemplate string

type FilterQueryPromptResult struct {
	AlertIDs []types.AlertID `json:"alert_ids"`
}

func BuildFilterQueryPrompt(ctx context.Context, query string, alerts alert.Alerts) (string, error) {
	tmpl, err := template.New("filter_query").Parse(filterQueryTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	example := FilterQueryPromptResult{
		AlertIDs: []types.AlertID{
			types.NewAlertID(),
			types.NewAlertID(),
			types.NewAlertID(),
		},
	}
	schema, err := generateSchema(example).Stringify()
	if err != nil {
		return "", err
	}

	rawExample, err := stringify(example)
	if err != nil {
		return "", err
	}

	rawAlerts := []string{}
	for _, alert := range alerts {
		rawAlert, err := stringify(alert)
		if err != nil {
			return "", err
		}
		rawAlerts = append(rawAlerts, rawAlert)
	}

	input := map[string]any{
		"query":   query,
		"alerts":  rawAlerts,
		"schema":  schema,
		"example": rawExample,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "filter_query", input); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	return buf.String(), nil
}

//go:embed templates/meta_list.md
var metaListTemplate string

type MetaListPromptResult struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func BuildMetaListPrompt(ctx context.Context, alertList alert.List) (string, error) {
	tmpl, err := template.New("meta_list").Parse(metaListTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}
	rawAlerts := []string{}
	alerts := alertList.Alerts
	if len(alerts) > 10 {
		// Create a random permutation and take first 10
		rand.Shuffle(len(alerts), func(i, j int) {
			alerts[i], alerts[j] = alerts[j], alerts[i]
		})
		alerts = alerts[:10]
	}

	for _, alert := range alerts {
		rawAlert, err := stringify(alert.Data)
		if err != nil {
			return "", err
		}
		rawAlerts = append(rawAlerts, rawAlert)
	}

	schema, err := generateSchema(MetaListPromptResult{}).Stringify()
	if err != nil {
		return "", err
	}

	input := map[string]any{
		"alerts": rawAlerts,
		"schema": schema,
		"lang":   lang.From(ctx).Name(),
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "meta_list", input); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	return buf.String(), nil
}

//go:embed templates/session_start.md
var sessionStartTemplate string

func BuildSessionStartPrompt(ctx context.Context, alerts alert.Alerts) (string, error) {
	tmpl, err := template.New("session_start").Parse(sessionStartTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	input := map[string]any{
		"lang": lang.From(ctx).Name(),
	}

	if len(alerts) < 10 {
		rawAlerts := []string{}
		for _, alert := range alerts {
			rawAlert, err := stringify(alert.Data)
			if err != nil {
				return "", err
			}
			rawAlerts = append(rawAlerts, rawAlert)
		}
		input["alerts"] = rawAlerts
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "session_start", input); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	return buf.String(), nil
}

//go:embed templates/session_next.md
var sessionNextTemplate string

func BuildSessionNextPrompt(ctx context.Context, result *action.Result) (string, error) {
	tmpl, err := template.New("session_next").Parse(sessionNextTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	input := map[string]any{
		"lang": lang.From(ctx).Name(),
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "session_next", input); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	return buf.String(), nil
}
