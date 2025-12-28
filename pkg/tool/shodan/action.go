package shodan

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
	return "shodan"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "shodan-api-key",
			Usage:       "Shodan API key",
			Destination: &x.apiKey,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_SHODAN_API_KEY"),
		},
		&cli.StringFlag{
			Name:        "shodan-base-url",
			Usage:       "Shodan API base URL",
			Destination: &x.baseURL,
			Category:    "Tool",
			Value:       "https://api.shodan.io",
			Sources:     cli.EnvVars("WARREN_SHODAN_BASE_URL"),
		},
	}
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "shodan_host",
			Description: "Search the host information from Shodan.",
			Parameters: map[string]*gollem.Parameter{
				"target": {
					Type:        gollem.TypeString,
					Description: "The IP address to search",
				},
			},
		},
		{
			Name:        "shodan_domain",
			Description: "Search the domain information from Shodan.",
			Parameters: map[string]*gollem.Parameter{
				"target": {
					Type:        gollem.TypeString,
					Description: "The domain to search",
				},
			},
		},
		{
			Name:        "shodan_search",
			Description: "Search the internet using Shodan search query.",
			Parameters: map[string]*gollem.Parameter{
				"query": {
					Type:        gollem.TypeString,
					Description: "The search query to use",
				},
				"limit": {
					Type:        gollem.TypeInteger,
					Description: "Maximum number of results to return (default: 100)",
				},
			},
		},
	}, nil
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.apiKey == "" {
		return nil, goerr.New("Shodan API key is required")
	}

	client := &http.Client{}
	var endpoint string
	var queryParams url.Values

	switch name {
	case "shodan_host":
		target, ok := args["target"].(string)
		if !ok {
			return nil, goerr.New("target parameter is required")
		}
		endpoint = fmt.Sprintf("%s/shodan/host/%s", x.baseURL, target)
		queryParams = url.Values{}
		queryParams.Set("key", x.apiKey)

	case "shodan_domain":
		target, ok := args["target"].(string)
		if !ok {
			return nil, goerr.New("target parameter is required")
		}
		endpoint = fmt.Sprintf("%s/dns/domain/%s", x.baseURL, target)
		queryParams = url.Values{}
		queryParams.Set("key", x.apiKey)

	case "shodan_search":
		query, ok := args["query"].(string)
		if !ok {
			return nil, goerr.New("query parameter is required")
		}
		endpoint = fmt.Sprintf("%s/shodan/host/search", x.baseURL)
		queryParams = url.Values{}
		queryParams.Set("key", x.apiKey)
		queryParams.Set("query", query)

		if limit, ok := args["limit"].(float64); ok {
			queryParams.Set("limit", fmt.Sprintf("%d", int(limit)))
		} else if args["limit"] != nil {
			return nil, goerr.New("invalid limit parameter type",
				goerr.V("type", fmt.Sprintf("%T", args["limit"])),
				goerr.V("value", args["limit"]))
		}

	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s?%s", endpoint, queryParams.Encode()), nil)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send request")
	}
	defer safe.Close(ctx, resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, goerr.New("failed to query Shodan",
			goerr.V("status_code", resp.StatusCode),
			goerr.V("body", string(body)),
			goerr.V("endpoint", endpoint))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read response body")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal response body")
	}

	if errMsg, ok := data["error"].(string); ok {
		return nil, goerr.New("shodan api returned error",
			goerr.V("error", errMsg))
	}

	switch name {
	case "shodan_host":
		if _, ok := data["ip"].(string); !ok {
			return nil, goerr.New("invalid response: missing ip")
		}
	case "shodan_domain":
		if _, ok := data["domain"].(string); !ok {
			return nil, goerr.New("invalid response: missing domain")
		}
	case "shodan_search":
		if _, ok := data["matches"].([]interface{}); !ok {
			return nil, goerr.New("invalid response: missing matches")
		}
	}

	return data, nil
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

func New() *Action {
	return &Action{
		baseURL: "https://api.shodan.io",
	}
}
