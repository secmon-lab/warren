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
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/urfave/cli/v3"
)

type Action struct {
	apiKey  string
	baseURL string
	backoff time.Duration
	timeout time.Duration
}

var _ interfaces.Tool = &Action{}

func (x *Action) Helper() *cli.Command {
	return nil
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
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_URLSCAN_API_KEY"),
		},
		&cli.StringFlag{
			Name:        "urlscan-base-url",
			Usage:       "URLScan API base URL",
			Destination: &x.baseURL,
			Category:    "Tool",
			Value:       "https://urlscan.io/api/v1",
			Sources:     cli.EnvVars("WARREN_URLSCAN_BASE_URL"),
		},
		&cli.DurationFlag{
			Name:        "urlscan-backoff",
			Usage:       "URLScan API backoff duration",
			Destination: &x.backoff,
			Category:    "Tool",
			Value:       time.Duration(3) * time.Second,
			Sources:     cli.EnvVars("WARREN_URLSCAN_BACKOFF"),
		},
		&cli.DurationFlag{
			Name:        "urlscan-timeout",
			Usage:       "URLScan API timeout duration",
			Destination: &x.timeout,
			Category:    "Tool",
			Value:       time.Duration(30) * time.Second,
		},
	}
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "urlscan_scan",
			Description: "Scan a URL with URLScan",
			Parameters: map[string]*gollem.Parameter{
				"url": {
					Type:        gollem.TypeString,
					Description: "The URL to scan",
				},
			},
			Required: []string{"url"},
		},
	}, nil
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	logger := logging.From(ctx)
	if x.apiKey == "" {
		return nil, goerr.New("URLScan API key is required")
	}

	// Validate function name
	switch name {
	case "urlscan_scan":
		// Valid function name, continue
	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
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

	logger.Debug("sending scan request",
		"url", urlStr,
		"method", req.Method,
		"headers", req.Header,
	)
	resp, err := client.Do(req)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send request")
	}
	defer safe.Close(ctx, resp.Body)

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

	// Get the scan result
	req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/result/%s/", x.baseURL, result.UUID), nil)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request")
	}

	req.Header.Set("API-Key", x.apiKey)

	deadline := clock.Now(ctx).Add(x.timeout)
	for clock.Now(ctx).Before(deadline) {
		time.Sleep(x.backoff)

		resp, err = client.Do(req)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to send request")
		}

		if resp.StatusCode == http.StatusNotFound {
			safe.Close(ctx, resp.Body)
			logger.Debug("scan result not found, retrying",
				"uuid", result.UUID,
				"target", urlStr,
				"backoff", x.backoff,
			)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			safe.Close(ctx, resp.Body)
			return nil, goerr.New("failed to get scan result",
				goerr.V("status_code", resp.StatusCode),
				goerr.V("body", string(body)))
		}

		body, err := io.ReadAll(resp.Body)
		safe.Close(ctx, resp.Body)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to read response body")
		}

		var result map[string]any
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, goerr.Wrap(err, "failed to unmarshal response body")
		}

		return result, nil
	}

	return nil, goerr.New("scan result timeout",
		goerr.V("timeout", x.timeout),
		goerr.V("target", urlStr),
		goerr.V("uuid", result.UUID))
}

func (x *Action) Configure(ctx context.Context) error {
	if x.apiKey == "" {
		return errutil.ErrActionUnavailable
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

// Prompt returns additional instructions for the system prompt
func (x *Action) Prompt(ctx context.Context) (string, error) {
	return "", nil
}
