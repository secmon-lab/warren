package prompt

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"math/rand/v2"
	"text/template"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
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

//go:embed templates/session_init.md
var sessionInitTemplate string

func BuildSessionInitPrompt(ctx context.Context, alerts alert.Alerts) (string, error) {
	tmpl, err := template.New("session_init").Parse(sessionInitTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	const maxAlertSize = 200 * 1000

	rawAlerts := []string{}
	rawAlertsSize := 0
	for _, alert := range alerts {
		rawAlert, err := stringify(alert.Data)
		if err != nil {
			return "", err
		}
		if rawAlertsSize+len(rawAlert) > maxAlertSize {
			break
		}
		rawAlerts = append(rawAlerts, rawAlert)
		rawAlertsSize += len(rawAlert)
	}

	input := map[string]any{
		"alerts": rawAlerts,
		"total":  len(alerts),
		"lang":   lang.From(ctx).Name(),
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
	if err := tmpl.ExecuteTemplate(&buf, "session_init", input); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	return buf.String(), nil
}
