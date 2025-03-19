package otx

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model"
	"github.com/urfave/cli/v3"
)

type Action struct {
	apiKey  string
	baseURL string
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "otx-api-key",
			Usage:       "OTX API key",
			Destination: &x.apiKey,
			Category:    "Action",
			Sources:     cli.EnvVars("WARREN_OTX_API_KEY"),
		},
		&cli.StringFlag{
			Name:        "otx-base-url",
			Usage:       "OTX API base URL",
			Destination: &x.baseURL,
			Category:    "Action",
			Value:       "https://otx.alienvault.com/api/v1",
			Sources:     cli.EnvVars("WARREN_OTX_BASE_URL"),
		},
	}
}

func (x *Action) Spec() model.ActionSpec {
	return model.ActionSpec{
		Name:        "otx",
		Description: "You can search the indicator from OTX. Set indicator type and value. You can choose one of the following types: domain, ipv4, ipv6, hostname, file_hash. You need to extract actual value from the alert data. Do not set field name",
		Args: []model.ArgumentSpec{
			{
				Name:        "domain",
				Type:        "string",
				Description: "The domain to search",
			},
			{
				Name:        "ipv4",
				Type:        "string",
				Description: "The IPv4 address to search",
			},
			{
				Name:        "ipv6",
				Type:        "string",
				Description: "The IPv6 address to search",
			},
			{
				Name:        "hostname",
				Type:        "string",
				Description: "The hostname to search",
			},
			{
				Name:        "file_hash",
				Type:        "string",
				Description: "The file hash to search",
			},
		},
	}
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("api_key.len", len(x.apiKey)),
		slog.String("base_url", x.baseURL),
	)
}

func (x *Action) Configure(ctx context.Context) error {
	if x.apiKey == "" {
		return model.ErrActionUnavailable
	}
	if _, err := url.Parse(x.baseURL); err != nil {
		return goerr.Wrap(err, "invalid base URL", goerr.V("base_url", x.baseURL))
	}
	return nil
}

func (x *Action) Execute(ctx context.Context, slack interfaces.SlackThreadService, ssn interfaces.LLMSession, args model.Arguments) (*model.ActionResult, error) {
	if x.apiKey == "" {
		return nil, goerr.New("OTX API key is required")
	}

	client := &http.Client{}
	var indicator, indicatorType string

	// Determine which indicator type was provided
	switch {
	case args["domain"] != nil:
		indicator = args["domain"].(string)
		indicatorType = "domain"
	case args["ipv4"] != nil:
		indicator = args["ipv4"].(string)
		indicatorType = "IPv4"
	case args["ipv6"] != nil:
		indicator = args["ipv6"].(string)
		indicatorType = "IPv6"
	case args["hostname"] != nil:
		indicator = args["hostname"].(string)
		indicatorType = "hostname"
	case args["file_hash"] != nil:
		indicator = args["file_hash"].(string)
		indicatorType = "file"
	default:
		return nil, goerr.New("no valid indicator provided")
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
	defer resp.Body.Close()

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

	return &model.ActionResult{
		Message: fmt.Sprintf("OTX result for %s: %s", indicatorType, indicator),
		Type:    model.ActionResultTypeJSON,
		Data:    string(body),
	}, nil
}
