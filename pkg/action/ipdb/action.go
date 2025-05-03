package ipdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollam"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/urfave/cli/v3"
)

type Action struct {
	apiKey  string
	baseURL string
}

func (x *Action) Name() string {
	return "ipdb"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "ipdb-api-key",
			Usage:       "AbuseIPDB API key",
			Destination: &x.apiKey,
			Category:    "Action",
			Sources:     cli.EnvVars("WARREN_IPDB_API_KEY"),
		},
		&cli.StringFlag{
			Name:        "ipdb-base-url",
			Usage:       "AbuseIPDB API base URL",
			Destination: &x.baseURL,
			Category:    "Action",
			Value:       "https://api.abuseipdb.com/api/v2",
			Sources:     cli.EnvVars("WARREN_IPDB_BASE_URL"),
		},
	}
}

func (x *Action) Specs(ctx context.Context) ([]gollam.ToolSpec, error) {
	return []gollam.ToolSpec{
		{
			Name:        "ipdb.check",
			Description: "Check IP address information from AbuseIPDB.",
			Parameters: map[string]*gollam.Parameter{
				"target": {
					Type:        gollam.TypeString,
					Description: "The IP address to check",
				},
				"maxAgeInDays": {
					Type:        gollam.TypeInteger,
					Description: "The maximum age of reports in days (1-365)",
				},
			},
		},
	}, nil
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.apiKey == "" {
		return nil, goerr.New("AbuseIPDB API key is required")
	}

	client := &http.Client{}
	var ipAddress string

	// Determine which function was called
	switch name {
	case "ipdb.check":
		ipAddress = args["target"].(string)
	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}

	// Build URL with query parameters
	url := fmt.Sprintf("%s/check", x.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request")
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("ipAddress", ipAddress)
	if maxAge, ok := args["maxAgeInDays"].(int); ok {
		q.Add("maxAgeInDays", fmt.Sprintf("%d", maxAge))
	}
	req.URL.RawQuery = q.Encode()

	// Add headers
	req.Header.Set("Key", x.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, goerr.New("failed to query AbuseIPDB",
			goerr.V("status_code", resp.StatusCode),
			goerr.V("body", string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read response body")
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal response")
	}

	return result, nil
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
	)
}
