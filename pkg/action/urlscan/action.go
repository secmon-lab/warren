package urlscan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/urfave/cli/v3"
)

type Action struct {
	apiKey  string
	baseURL string
	backoff time.Duration
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "urlscan-api-key",
			Usage:       "URLScan API key",
			Destination: &x.apiKey,
			Category:    "Action",
			Sources:     cli.EnvVars("WARREN_URLSCAN_API_KEY"),
		},
		&cli.StringFlag{
			Name:        "urlscan-base-url",
			Usage:       "URLScan API base URL",
			Destination: &x.baseURL,
			Category:    "Action",
			Value:       "https://urlscan.io/api/v1",
			Sources:     cli.EnvVars("WARREN_URLSCAN_BASE_URL"),
		},
		&cli.DurationFlag{
			Name:        "urlscan-backoff",
			Usage:       "URLScan API backoff duration",
			Destination: &x.backoff,
			Category:    "Action",
			Value:       time.Duration(3) * time.Second,
			Sources:     cli.EnvVars("WARREN_URLSCAN_BACKOFF"),
		},
	}
}

func (x *Action) Enabled() bool {
	return x.apiKey != ""
}

func (x Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("api_key.len", len(x.apiKey)),
		slog.String("base_url", x.baseURL),
		slog.Duration("backoff", x.backoff),
	)
}

func (x *Action) Spec() model.ActionSpec {
	return model.ActionSpec{
		Name:        "urlscan",
		Description: "Scan a URL with URLScan",
		Args: []model.ArgumentSpec{
			{
				Name:        "url",
				Type:        "string",
				Description: "The URL to scan",
				Required:    true,
			},
		},
	}
}

func (x *Action) Execute(ctx context.Context, slack interfaces.SlackService, ssn interfaces.GenAIChatSession, args model.Arguments) (string, error) {
	url, ok := args["url"]
	if !ok {
		return "", goerr.New("url is required")
	}

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "POST", x.baseURL+"/scan", strings.NewReader(`{"url":"`+url+`"}`))
	if err != nil {
		return "", goerr.Wrap(err, "failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("API-Key", x.apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", goerr.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", goerr.New("failed to scan URL", goerr.V("status_code", resp.StatusCode), goerr.V("body", string(body)))
	}

	var result struct {
		UUID    string `json:"uuid"`
		Message string `json:"message"`
		Result  string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", goerr.Wrap(err, "failed to decode response")
	}

	// Poll the result API until scan is complete
	resultURL := fmt.Sprintf("%s/result/%s/", x.baseURL, result.UUID)
	for i := 0; i < 5; i++ { // Try up to 5 times with increasing delay
		time.Sleep(time.Duration(1<<i) * x.backoff)

		req, err := http.NewRequestWithContext(ctx, "GET", resultURL, nil)
		if err != nil {
			return "", goerr.Wrap(err, "failed to create result request")
		}

		resp, err := client.Do(req)
		if err != nil {
			return "", goerr.Wrap(err, "failed to get result")
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", goerr.Wrap(err, "failed to read response body")
			}
			return string(body), nil
		case http.StatusNotFound:
			continue
		default:
			body, _ := io.ReadAll(resp.Body)
			return "", goerr.New("failed to get scan result", goerr.V("status_code", resp.StatusCode), goerr.V("body", string(body)))
		}
	}

	return "", goerr.New("failed to get scan result")
}
