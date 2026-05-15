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
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/urfave/cli/v3"
)

const (
	defaultTimeout = 30 * time.Second
	userAgent      = "warren-webfetch/1.0 (+https://github.com/secmon-lab/warren)"

	flagCategory = "WebFetch"
)

// Action implements the interfaces.Tool interface for fetching web content
// and sanitizing it through an LLM-based pipeline.
//
// LLM analysis is configured via the --webfetch-llm-* flags. If
// --webfetch-llm-provider is empty, the analyze step is disabled and the
// extracted text is returned verbatim. In that mode, HITL approval is
// required for every web_fetch invocation (RequiresHITL returns true) so a
// human gates each request.
type Action struct {
	client *http.Client

	// Flag-bound fields populated via cli.StringFlag Destination.
	llmProvider string
	llmModel    string
	llmArgs     string
	llmAPIKey   string

	// Resolved at Configure(). Nil when LLM analysis is disabled.
	llmClient gollem.LLMClient
}

var _ interfaces.Tool = &Action{}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) ID() string {
	return "webfetch"
}

func (x *Action) Description() string {
	return "Web page content fetching with optional LLM-based Markdown extraction and indirect prompt injection detection"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "webfetch-llm-provider",
			Usage:       "LLM provider for webfetch analyze step (gemini, claude, openai). Empty disables LLM analysis and forces HITL approval per call.",
			Sources:     cli.EnvVars("WARREN_WEBFETCH_LLM_PROVIDER"),
			Destination: &x.llmProvider,
			Category:    flagCategory,
		},
		&cli.StringFlag{
			Name:        "webfetch-llm-model",
			Usage:       "LLM model name (e.g. gemini-2.5-flash, claude-sonnet-4@20250514, gpt-4o). Required when --webfetch-llm-provider is set.",
			Sources:     cli.EnvVars("WARREN_WEBFETCH_LLM_MODEL"),
			Destination: &x.llmModel,
			Category:    flagCategory,
		},
		&cli.StringFlag{
			Name:        "webfetch-llm-args",
			Usage:       "Provider-specific options as comma-separated key=value (e.g. 'project_id=my-proj,location=us-central1,temperature=0.2').",
			Sources:     cli.EnvVars("WARREN_WEBFETCH_LLM_ARGS"),
			Destination: &x.llmArgs,
			Category:    flagCategory,
		},
		&cli.StringFlag{
			Name:        "webfetch-llm-api-key",
			Usage:       "API key for the LLM provider. Required for openai and for claude direct (Anthropic) route. Ignored for gemini and for claude Vertex route.",
			Sources:     cli.EnvVars("WARREN_WEBFETCH_LLM_API_KEY"),
			Destination: &x.llmAPIKey,
			Category:    flagCategory,
		},
	}
}

func (x *Action) Specs(_ context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "web_fetch",
			Description: "Fetch a web page and return its body. When LLM analysis is enabled, the body is reformatted as Markdown and screened for indirect prompt injection; otherwise the extracted text is returned verbatim (HITL approval gates each call in that mode).",
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

	status, contentType, body, err := x.fetch(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	text, _, err := extract(contentType, body)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to extract body",
			goerr.V("url", rawURL))
	}

	if x.llmClient == nil {
		// LLM analysis disabled: return the extracted text verbatim. HITL
		// approval (wired in pkg/cli/serve.go) is the safety net in this mode.
		return map[string]any{
			"result":       text,
			"url":          rawURL,
			"status":       status,
			"content_type": contentType,
			"llm_analysis": "disabled",
		}, nil
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

func (x *Action) Configure(ctx context.Context) error {
	if x.client == nil {
		x.client = &http.Client{Timeout: defaultTimeout}
	}

	if x.llmProvider == "" {
		logging.From(ctx).Info("webfetch LLM analysis disabled; HITL approval is required for every web_fetch call")
		return nil
	}

	parsedArgs, err := parseLLMArgs(x.llmArgs)
	if err != nil {
		return goerr.Wrap(err, "failed to parse --webfetch-llm-args",
			goerr.V("provider", x.llmProvider))
	}

	client, err := buildLLMClient(ctx, x.llmProvider, x.llmModel, parsedArgs, x.llmAPIKey)
	if err != nil {
		return goerr.Wrap(err, "failed to build webfetch LLM client",
			goerr.V("provider", x.llmProvider),
			goerr.V("model", x.llmModel))
	}

	if err := pingLLMClient(ctx, client); err != nil {
		return goerr.Wrap(err, "webfetch LLM ping failed",
			goerr.V("provider", x.llmProvider),
			goerr.V("model", x.llmModel))
	}

	x.llmClient = client
	logging.From(ctx).Info("webfetch LLM analysis enabled",
		"provider", x.llmProvider,
		"model", x.llmModel)
	return nil
}

// RequiresHITL reports whether the web_fetch tool should be gated by a HITL
// (Human-in-the-Loop) approval dialog. When LLM analysis is enabled, the LLM
// performs indirect-prompt-injection screening so the dialog is unnecessary
// and is skipped; when LLM is disabled, the dialog is the only remaining
// safety layer and is always required.
func (x *Action) RequiresHITL() bool {
	return x.llmProvider == ""
}

func (x *Action) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.Bool("hitl_required", x.RequiresHITL()),
	}
	if x.llmProvider != "" {
		attrs = append(attrs,
			slog.String("llm_provider", x.llmProvider),
			slog.String("llm_model", x.llmModel),
			slog.String("llm_args", x.llmArgs),
		)
	}
	return slog.GroupValue(attrs...)
}

func (x *Action) Prompt(_ context.Context) (string, error) {
	return "", nil
}
