package prompt

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"math/rand/v2"
	"text/template"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/utils/lang"
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

//go:embed templates/action.md
var actionTemplate string

type ActionPromptResult struct {
	Action string          `json:"action"`
	Args   model.Arguments `json:"args" schema:"optional"`
}

func BuildActionPrompt(ctx context.Context, actions []model.ActionSpec) (string, error) {
	tmpl, err := template.New("action").Parse(actionTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	schema, err := generateSchema(ActionPromptResult{}).Stringify()
	if err != nil {
		return "", err
	}

	input := map[string]any{
		"actions": actions,
		"schema":  schema,
		"lang":    lang.From(ctx).Name(),
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "action", input); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	return buf.String(), nil
}

//go:embed templates/aggregate.md
var aggregateTemplate string

type AggregatePromptResult struct {
	AlertID string `json:"alert_id"`
}

func BuildAggregatePrompt(ctx context.Context, newAlert model.Alert, candidates []model.Alert) (string, error) {
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

	schema, err := generateSchema(model.AlertFinding{}).Stringify()
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
	Attrs       []model.Attribute `json:"attrs"`
}

func BuildMetaPrompt(ctx context.Context, alert model.Alert) (string, error) {
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

func BuildIgnorePolicyPrompt(ctx context.Context, policy model.PolicyData, alerts []model.Alert, note string) (string, error) {
	tmpl, err := template.New("ignore_policy").Parse(ignorePolicyTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	outputSchema, err := generateSchema(IgnorePolicyPromptResult{}).Stringify()
	if err != nil {
		return "", err
	}

	rawPolicy, err := stringify(policy.Data)
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

func BuildTestDataReadmePrompt(ctx context.Context, action string, alerts []model.Alert) (string, error) {
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
	AlertIDs []model.AlertID `json:"alert_ids"`
}

func BuildFilterQueryPrompt(ctx context.Context, query string, alerts []model.Alert) (string, error) {
	tmpl, err := template.New("filter_query").Parse(filterQueryTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	example := FilterQueryPromptResult{
		AlertIDs: []model.AlertID{
			model.NewAlertID(),
			model.NewAlertID(),
			model.NewAlertID(),
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

func BuildMetaListPrompt(ctx context.Context, alertList model.AlertList) (string, error) {
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
