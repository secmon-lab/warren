package webfetch

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/urfave/cli/v3"
)

const (
	defaultTimeout = 30 * time.Second
	userAgent      = "warren-webfetch/1.0 (+https://github.com/secmon-lab/warren)"
)

// Action implements the interfaces.Tool interface for fetching web content
// and sanitizing it through an LLM-based pipeline.
type Action struct {
	client    *http.Client
	llmClient gollem.LLMClient
}

var _ interfaces.Tool = &Action{}

// SetLLMClient injects the LLM client used by the analyze step.
// It is invoked from pkg/cli during start-up so that all entrypoints that
// own an LLM client share the same instance with the webfetch tool.
func (x *Action) SetLLMClient(client gollem.LLMClient) {
	x.llmClient = client
}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) ID() string {
	return "webfetch"
}

func (x *Action) Description() string {
	return "Web page content fetching with LLM-based Markdown extraction and indirect prompt injection detection"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{}
}

func (x *Action) Specs(_ context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "web_fetch",
			Description: "Fetch a web page and return its body as Markdown after extracting the main content and running an indirect-prompt-injection check.",
			Parameters: map[string]*gollem.Parameter{
				"url": {
					Type:        gollem.TypeString,
					Description: "The URL to fetch (http or https only)",
					Required:    true,
				},
			},
		},
	}, nil
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if name != "web_fetch" {
		return nil, goerr.New("invalid function name",
			goerr.T(errutil.TagValidation),
			goerr.V("name", name))
	}

	rawURL, ok := args["url"].(string)
	if !ok || rawURL == "" {
		return nil, goerr.New("url is required",
			goerr.T(errutil.TagValidation))
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to parse url",
			goerr.T(errutil.TagValidation),
			goerr.V("url", rawURL))
	}
	switch parsed.Scheme {
	case "http", "https":
	default:
		return nil, goerr.New("unsupported url scheme (only http/https are allowed)",
			goerr.T(errutil.TagValidation),
			goerr.V("url", rawURL),
			goerr.V("scheme", parsed.Scheme))
	}
	if parsed.Host == "" {
		return nil, goerr.New("url is missing a host",
			goerr.T(errutil.TagValidation),
			goerr.V("url", rawURL))
	}

	if x.llmClient == nil {
		return nil, goerr.New("LLM client is not injected for webfetch",
			goerr.T(errutil.TagInternal))
	}

	status, contentType, body, err := x.fetch(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	text, _, err := extract(contentType, body)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to extract body",
			goerr.V("url", rawURL))
	}

	result, err := analyze(ctx, x.llmClient, text)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to analyze body",
			goerr.V("url", rawURL))
	}

	if result.Malicious {
		return nil, goerr.New("indirect prompt injection detected in fetched body",
			goerr.T(errutil.TagValidation),
			goerr.V("url", rawURL),
			goerr.V("reason", result.Reason))
	}

	return map[string]any{
		"result":       result.Markdown,
		"url":          rawURL,
		"status":       status,
		"content_type": contentType,
	}, nil
}

// fetch performs the HTTP GET. It enforces a request timeout and sets a stable
// User-Agent. No response-size cap is applied; the request timeout and the
// context deadline are the only bound on the operation.
func (x *Action) fetch(ctx context.Context, rawURL string) (int, string, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, "", nil, goerr.Wrap(err, "failed to create http request",
			goerr.T(errutil.TagValidation),
			goerr.V("url", rawURL))
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := x.client.Do(req)
	if err != nil {
		return 0, "", nil, goerr.Wrap(err, "failed to fetch url",
			goerr.T(errutil.TagExternal),
			goerr.V("url", rawURL))
	}
	defer safe.Close(ctx, resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, resp.Header.Get("Content-Type"), nil,
			goerr.Wrap(err, "failed to read response body",
				goerr.T(errutil.TagExternal),
				goerr.V("url", rawURL))
	}

	return resp.StatusCode, resp.Header.Get("Content-Type"), body, nil
}

func (x *Action) Configure(_ context.Context) error {
	if x.client == nil {
		x.client = &http.Client{Timeout: defaultTimeout}
	}
	return nil
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue()
}

func (x *Action) Prompt(_ context.Context) (string, error) {
	return "", nil
}
