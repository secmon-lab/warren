package prompt

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
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
