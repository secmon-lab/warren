package prompt

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
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

//go:embed templates/make_group.md
var makeGroupTemplate string

type MakeGroupPromptResult struct {
	Groups []model.AlertGroupMetadata `json:"groups"`
}

func BuildMakeGroupPrompt(ctx context.Context, alerts []model.Alert, maxGroups int) (string, error) {
	tmpl, err := template.New("make_group").Parse(makeGroupTemplate)
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

	schema, err := generateSchema(MakeGroupPromptResult{}).Stringify()
	if err != nil {
		return "", err
	}

	example := MakeGroupPromptResult{
		Groups: []model.AlertGroupMetadata{
			{
				Title:       "Group 1",
				Description: "Group 1 description",
				AlertIDs:    []model.AlertID{model.NewAlertID(), model.NewAlertID()},
			},
		},
	}
	rawExample, err := json.MarshalIndent(example, "", "  ")
	if err != nil {
		return "", goerr.Wrap(err, "failed to marshal example", goerr.V("example", example))
	}

	input := map[string]any{
		"alerts":     rawAlerts,
		"schema":     schema,
		"example":    string(rawExample),
		"max_groups": maxGroups,
		"lang":       lang.From(ctx).Name(),
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "make_group", input); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	return buf.String(), nil
}
