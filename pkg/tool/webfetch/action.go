package webfetch

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gollem-dev/gollem"
	extwebfetch "github.com/gollem-dev/tools/webfetch"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

const (
	defaultTimeout = 30 * time.Second

	flagCategory = "WebFetch"
)

// Action is the warren-side wrapper around github.com/gollem-dev/tools/webfetch.
// It implements interfaces.Tool, binding CLI flags and warren-specific HITL
// gating onto the external gollem.ToolSet that carries the Specs/Run logic.
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

	// inner is the external ToolSet constructed in Configure().
	inner gollem.ToolSet
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

func (x *Action) Configure(ctx context.Context) error {
	if x.client == nil {
		x.client = &http.Client{Timeout: defaultTimeout}
	}

	var opts []extwebfetch.Option
	if x.client != nil {
		opts = append(opts, extwebfetch.WithHTTPClient(x.client))
	}

	// If llmClient was not pre-injected (e.g. by tests), build one from flags.
	if x.llmClient == nil && x.llmProvider != "" {
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
	}

	if x.llmClient == nil {
		logging.From(ctx).Info("webfetch LLM analysis disabled; HITL approval is required for every web_fetch call")
	} else {
		opts = append(opts, extwebfetch.WithLLMClient(x.llmClient))
		logging.From(ctx).Info("webfetch LLM analysis enabled",
			"provider", x.llmProvider,
			"model", x.llmModel)
	}

	inner, err := extwebfetch.New(opts...)
	if err != nil {
		return goerr.Wrap(err, "failed to construct webfetch tool")
	}
	x.inner = inner
	return nil
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	if x.inner == nil {
		return nil, goerr.New("webfetch tool is not configured")
	}
	return x.inner.Specs(ctx)
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.inner == nil {
		return nil, goerr.New("webfetch tool is not configured")
	}
	return x.inner.Run(ctx, name, args)
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
