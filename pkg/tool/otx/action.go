package otx

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

var _ interfaces.Tool = &Action{}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) Name() string {
	return "otx"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "otx-api-key",
			Usage:       "OTX API key",
			Destination: &x.apiKey,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_OTX_API_KEY"),
		},
		&cli.StringFlag{
			Name:        "otx-base-url",
			Usage:       "OTX API base URL",
			Destination: &x.baseURL,
			Category:    "Tool",
			Value:       "https://otx.alienvault.com/api/v1",
			Sources:     cli.EnvVars("WARREN_OTX_BASE_URL"),
		},
	}
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "otx.ipv4",
			Description: "Search the indicator of IPv4 from OTX.",
			Parameters: map[string]*gollem.Parameter{
				"target": {
					Type:        gollem.TypeString,
					Description: "The IPv4 address to search",
				},
			},
		},
		{
			Name:        "otx.domain",
			Description: "Search the indicator of domain from OTX.",
			Parameters: map[string]*gollem.Parameter{
				"target": {
					Type:        gollem.TypeString,
					Description: "The domain to search",
				},
			},
		},
		{
			Name:        "otx.ipv6",
			Description: "Search the indicator of IPv6 from OTX.",
			Parameters: map[string]*gollem.Parameter{
				"target": {
					Type:        gollem.TypeString,
					Description: "The IPv6 address to search",
				},
			},
		},
		{
			Name:        "otx.hostname",
			Description: "Search the indicator of hostname from OTX.",
			Parameters: map[string]*gollem.Parameter{
				"target": {
					Type:        gollem.TypeString,
					Description: "The hostname to search",
				},
			},
		},
		{
			Name:        "otx.file_hash",
			Description: "Search the indicator of file hash from OTX.",
			Parameters: map[string]*gollem.Parameter{
				"target": {
					Type:        gollem.TypeString,
					Description: "The file hash to search",
				},
			},
		},
	}, nil
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.apiKey == "" {
		return nil, goerr.New("OTX API key is required")
	}

	client := &http.Client{}
	var indicator, indicatorType string

	// Determine which indicator type was provided based on function name
	switch name {
	case "otx.domain":
		indicator = args["target"].(string)
		indicatorType = "domain"
	case "otx.ipv4":
		indicator = args["target"].(string)
		indicatorType = "IPv4"
	case "otx.ipv6":
		indicator = args["target"].(string)
		indicatorType = "IPv6"
	case "otx.hostname":
		indicator = args["target"].(string)
		indicatorType = "hostname"
	case "otx.file_hash":
		indicator = args["target"].(string)
		indicatorType = "file"
	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}

	url := fmt.Sprintf("%s/indicators/%s/%s/general", x.baseURL, indicatorType, indicator)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request")
	}

	req.Header.Set("X-OTX-API-KEY", x.apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send request")
	}
	defer safe.Close(ctx, resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, goerr.New("failed to query OTX",
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

// Prompt returns additional instructions for the system prompt
func (x *Action) Prompt(ctx context.Context) (string, error) {
	return "", nil
}
