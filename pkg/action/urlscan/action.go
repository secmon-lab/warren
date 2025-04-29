package urlscan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollam"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/urfave/cli/v3"
)

type Action struct {
	apiKey  string
	baseURL string
	backoff time.Duration
}

func (x *Action) Name() string {
	return "urlscan"
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

func (x *Action) Specs(ctx context.Context) ([]gollam.ToolSpec, error) {
	return []gollam.ToolSpec{
		{
			Name:        "urlscan.scan",
			Description: "Scan a URL with URLScan",
			Parameters: map[string]*gollam.Parameter{
				"url": {
					Type:        gollam.TypeString,
					Description: "The URL to scan",
				},
			},
		},
	}, nil
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.apiKey == "" {
		return nil, goerr.New("URLScan API key is required")
	}

	urlStr, ok := args["url"].(string)
	if !ok {
		return nil, goerr.New("url parameter is required")
	}

	if _, err := url.Parse(urlStr); err != nil {
		return nil, goerr.Wrap(err, "invalid URL", goerr.V("url", urlStr))
	}

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/scan/", x.baseURL), strings.NewReader(fmt.Sprintf(`{"url": "%s"}`, urlStr)))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request")
	}

	req.Header.Set("API-Key", x.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, goerr.New("failed to query URLScan",
			goerr.V("status_code", resp.StatusCode),
			goerr.V("body", string(body)))
	}

	var result struct {
		UUID string `json:"uuid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, goerr.Wrap(err, "failed to decode response")
	}

	// Wait for the scan to complete
	time.Sleep(x.backoff)

	// Get the scan result
	req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/result/%s/", x.baseURL, result.UUID), nil)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request")
	}

	req.Header.Set("API-Key", x.apiKey)

	resp, err = client.Do(req)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, goerr.New("failed to get scan result",
			goerr.V("status_code", resp.StatusCode),
			goerr.V("body", string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read response body")
	}

	return map[string]any{
		"body": string(body),
	}, nil
}

func (x *Action) Configure(ctx context.Context) error {
	if x.apiKey == "" {
		return errs.ErrActionUnavailable
	}
	if _, err := url.Parse(x.baseURL); err != nil {
		return goerr.Wrap(err, "invalid base URL", goerr.V("base_url", x.baseURL))
	}
	return nil
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("api_key.len", len(x.apiKey)),
		slog.String("base_url", x.baseURL),
		slog.Duration("backoff", x.backoff),
	)
}
