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
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/urfave/cli/v3"
)

type Action struct {
	apiKey  string
	baseURL string
}

var _ interfaces.Tool = &Action{}

func (x *Action) Helper() *cli.Command {
	return nil
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
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_IPDB_API_KEY"),
		},
		&cli.StringFlag{
			Name:        "ipdb-base-url",
			Usage:       "AbuseIPDB API base URL",
			Destination: &x.baseURL,
			Category:    "Tool",
			Value:       "https://api.abuseipdb.com/api/v2",
			Sources:     cli.EnvVars("WARREN_IPDB_BASE_URL"),
		},
	}
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "ipdb_check",
			Description: "Check IP address information from AbuseIPDB.",
			Parameters: map[string]*gollem.Parameter{
				"target": {
					Type:        gollem.TypeString,
					Description: "The IP address to check",
				},
				"maxAgeInDays": {
					Type:        gollem.TypeInteger,
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
	case "ipdb_check":
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
	if maxAge, ok := args["maxAgeInDays"].(float64); ok {
		q.Add("maxAgeInDays", fmt.Sprintf("%d", int(maxAge)))
	} else if args["maxAgeInDays"] != nil {
		return nil, goerr.New("invalid maxAgeInDays parameter type",
			goerr.V("type", fmt.Sprintf("%T", args["maxAgeInDays"])),
			goerr.V("value", args["maxAgeInDays"]))
	}
	req.URL.RawQuery = q.Encode()

	// Add headers
	req.Header.Set("Key", x.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send request")
	}
	defer safe.Close(ctx, resp.Body)

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
	)
}

// Prompt returns additional instructions for the system prompt
func (x *Action) Prompt(ctx context.Context) (string, error) {
	return "", nil
}
