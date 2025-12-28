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
	return "abusech"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "abusech-api-key",
			Usage:       "abuse.ch API key",
			Destination: &x.apiKey,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_ABUSECH_AUTH_KEY"),
		},
		&cli.StringFlag{
			Name:        "abusech-base-url",
			Usage:       "abuse.ch API base URL",
			Destination: &x.baseURL,
			Category:    "Tool",
			Value:       "https://mb-api.abuse.ch/api/v1",
			Sources:     cli.EnvVars("WARREN_ABUSECH_BASE_URL"),
		},
	}
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "abusech.bazaar.query",
			Description: "Query malware information from MalwareBazaar by file hash value.",
			Parameters: map[string]*gollem.Parameter{
				"hash": {
					Type:        gollem.TypeString,
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, x.baseURL+"/", strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request")
	}

	// Set headers
	req.Header.Set("Auth-Key", x.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Connection", "Keep-Alive")

	// Send request
	eb := goerr.NewBuilder(
		goerr.V("request_url", x.baseURL),
		goerr.V("request_method", req.Method),
		goerr.V("request_body", formData.Encode()),
	)

	resp, err := client.Do(req)
	if err != nil {
		return nil, eb.Wrap(err, "failed to send request")
	}
	defer safe.Close(ctx, resp.Body)

	body, err := io.ReadAll(resp.Body)
	eb = eb.With(
		goerr.V("response_status_code", resp.StatusCode),
		goerr.V("response_body", string(body)),
	)
	if err != nil {
		return nil, eb.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, eb.Wrap(err, "unexpected response code from MalwareBazaar")
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, eb.Wrap(err, "failed to unmarshal response")
	}

	// Check for API error response
	if status, ok := result["query_status"].(string); ok && status == "error" {
		errMsg, _ := result["error_message"].(string)
		if errMsg == "" {
			errMsg, _ = result["error"].(string)
		}
		return nil, eb.Wrap(err, "MalwareBazaar API returned error",
			goerr.V("error", errMsg))
	}

	// Validate response format
	if _, ok := result["data"].([]interface{}); !ok {
		return nil, eb.Wrap(err, "invalid response format: missing data array")
	}

	return result, nil
}

func (x *Action) Configure(ctx context.Context) error {
	if x.apiKey == "" {
		return errutil.ErrActionUnavailable
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

// Prompt returns additional instructions for the system prompt
func (x *Action) Prompt(ctx context.Context) (string, error) {
	return "", nil
}

func New() *Action {
	return &Action{
		baseURL: "https://mb-api.abuse.ch/api/v1",
	}
}
