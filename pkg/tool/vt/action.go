package vt

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
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/urfave/cli/v3"
)

type Action struct {
	apiKey  string
	baseURL string
}

func (x *Action) Name() string {
	return "vt"
}

var _ interfaces.Tool = &Action{}

func (x *Action) Helper() *cli.Command {
	return nil
}
func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "vt-api-key",
			Usage:       "VirusTotal API key",
			Destination: &x.apiKey,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_VT_API_KEY"),
		},
		&cli.StringFlag{
			Name:        "vt-base-url",
			Usage:       "VirusTotal API base URL",
			Destination: &x.baseURL,
			Category:    "Tool",
			Value:       "https://www.virustotal.com/api/v3",
			Sources:     cli.EnvVars("WARREN_VT_BASE_URL"),
		},
	}
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "vt_ip",
			Description: "Search the indicator of IPv4/IPv6 from VirusTotal.",
			Parameters: map[string]*gollem.Parameter{
				"target": {
					Type:        gollem.TypeString,
					Description: "The IP address to search",
				},
			},
		},
		{
			Name:        "vt_domain",
			Description: "Search the indicator of domain from VirusTotal.",
			Parameters: map[string]*gollem.Parameter{
				"target": {
					Type:        gollem.TypeString,
					Description: "The domain to search",
				},
			},
		},
		{
			Name:        "vt_file_hash",
			Description: "Search the indicator of file hash from VirusTotal.",
			Parameters: map[string]*gollem.Parameter{
				"target": {
					Type:        gollem.TypeString,
					Description: "The file hash to search",
				},
			},
		},
		{
			Name:        "vt_url",
			Description: "Search the indicator of URL from VirusTotal.",
			Parameters: map[string]*gollem.Parameter{
				"target": {
					Type:        gollem.TypeString,
					Description: "The URL to search",
				},
			},
		},
	}, nil
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.apiKey == "" {
		return nil, goerr.New("VirusTotal API key is required")
	}

	client := &http.Client{}
	var indicator, indicatorType string

	// Determine which indicator type was provided based on function name
	switch name {
	case "vt_domain":
		indicator = args["target"].(string)
		indicatorType = "domains"
	case "vt_ip":
		indicator = args["target"].(string)
		indicatorType = "ip_addresses"
	case "vt_file_hash":
		indicator = args["target"].(string)
		indicatorType = "files"
	case "vt_url":
		indicator = args["target"].(string)
		indicatorType = "urls"
	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}

	url := fmt.Sprintf("%s/%s/%s", x.baseURL, indicatorType, indicator)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request")
	}

	req.Header.Set("x-apikey", x.apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send request")
	}
	defer safe.Close(ctx, resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, goerr.New("failed to query VirusTotal",
			goerr.V("status_code", resp.StatusCode),
			goerr.V("body", string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read response body")
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal response body")
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

// Prompt returns additional instructions for the system prompt
func (x *Action) Prompt(ctx context.Context) (string, error) {
	return "", nil
}
