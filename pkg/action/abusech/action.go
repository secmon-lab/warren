package abusech

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

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
	return "abusech"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "abusech-api-key",
			Usage:       "abuse.ch API key",
			Destination: &x.apiKey,
			Category:    "Action",
			Sources:     cli.EnvVars("WARREN_ABUSECH_API_KEY"),
		},
		&cli.StringFlag{
			Name:        "abusech-base-url",
			Usage:       "abuse.ch API base URL",
			Destination: &x.baseURL,
			Category:    "Action",
			Value:       "https://mb-api.abuse.ch/api/v1",
			Sources:     cli.EnvVars("WARREN_ABUSECH_BASE_URL"),
		},
	}
}

func (x *Action) Specs(ctx context.Context) ([]gollam.ToolSpec, error) {
	return []gollam.ToolSpec{
		{
			Name:        "abusech.bazaar.query",
			Description: "Query malware information from MalwareBazaar by hash value.",
			Parameters: map[string]*gollam.Parameter{
				"hash": {
					Type:        gollam.TypeString,
					Description: "The hash value (MD5, SHA1, or SHA256) to query",
				},
			},
		},
	}, nil
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.apiKey == "" {
		return nil, goerr.New("abuse.ch API key is required")
	}

	client := &http.Client{}
	var hash string

	switch name {
	case "abusech.bazaar.query":
		var ok bool
		hash, ok = args["hash"].(string)
		if !ok {
			return nil, goerr.New("hash parameter is required")
		}
	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}

	// Build form data
	formData := url.Values{}
	formData.Set("query", "get_info")
	formData.Set("hash", hash)

	// Create request with form data
	req, err := http.NewRequestWithContext(ctx, "POST", x.baseURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request")
	}

	req.Header.Set("Auth-Key", x.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, goerr.New("failed to query MalwareBazaar",
			goerr.V("status_code", resp.StatusCode),
			goerr.V("body", string(body)))
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal response")
	}

	// Check for API error response
	if status, ok := result["query_status"].(string); ok && status == "error" {
		errMsg, _ := result["error"].(string)
		return nil, goerr.New("MalwareBazaar API returned error",
			goerr.V("error", errMsg))
	}

	return result, nil
}

func (x *Action) Configure(ctx context.Context) error {
	if x.apiKey == "" {
		return errs.ErrActionUnavailable
	}

	parsedURL, err := url.Parse(x.baseURL)
	if err != nil {
		return goerr.Wrap(err, "invalid base URL", goerr.V("base_url", x.baseURL))
	}

	if !strings.HasPrefix(parsedURL.Scheme, "http") {
		return goerr.New("invalid base URL scheme",
			goerr.V("base_url", x.baseURL),
			goerr.V("scheme", parsedURL.Scheme))
	}

	return nil
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("api_key.len", len(x.apiKey)),
		slog.String("base_url", x.baseURL),
	)
}

func New() *Action {
	return &Action{
		baseURL: "https://mb-api.abuse.ch/api/v1",
	}
}

// SetAPIKey sets the API key for the action
func (x *Action) SetAPIKey(key string) {
	x.apiKey = key
}

// SetBaseURL sets the base URL for the action
func (x *Action) SetBaseURL(url string) {
	x.baseURL = url
}
