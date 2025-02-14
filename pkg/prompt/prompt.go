package prompt

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"text/template"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/model"
)

func stringify(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return "", goerr.Wrap(err, "failed to marshal")
	}
	return buf.String(), nil
}

//go:embed templates/init.md
var initTemplate string

func BuildInitPrompt(alert any, maxRetry int) (string, error) {
	tmpl, err := template.New("init").Parse(initTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	rawAlert, err := stringify(alert)
	if err != nil {
		return "", goerr.Wrap(err, "failed to marshal alert")
	}

	input := map[string]any{
		"alert":     string(rawAlert),
		"max_retry": maxRetry,
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

func BuildActionPrompt(actions []model.ActionSpec) (string, error) {
	tmpl, err := template.New("action").Parse(actionTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	schema, err := generateSchema(ActionPromptResult{}).Stringify()
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate schema")
	}

	input := map[string]any{
		"actions": actions,
		"schema":  schema,
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

func BuildAggregatePrompt(newAlert model.Alert, candidates []model.Alert) (string, error) {
	tmpl, err := template.New("aggregate").Parse(aggregateTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	rawNewAlert, err := stringify(newAlert)
	if err != nil {
		return "", goerr.Wrap(err, "failed to marshal new alert")
	}

	rawCandidates := []string{}
	for _, candidate := range candidates {
		rawCandidate, err := stringify(candidate)
		if err != nil {
			return "", goerr.Wrap(err, "failed to marshal candidate")
		}
		rawCandidates = append(rawCandidates, rawCandidate)
	}

	schema, err := generateSchema(AggregatePromptResult{}).Stringify()
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate schema")
	}

	input := map[string]any{
		"new":        rawNewAlert,
		"candidates": rawCandidates,
		"schema":     schema,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "aggregate", input); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	return buf.String(), nil
}

//go:embed templates/finding.md
var findingTemplate string

func BuildFindingPrompt() (string, error) {
	tmpl, err := template.New("finding").Parse(findingTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	schema, err := generateSchema(model.AlertFinding{}).Stringify()
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate schema")
	}

	input := map[string]any{
		"schema": schema,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "finding", input); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	return buf.String(), nil
}
