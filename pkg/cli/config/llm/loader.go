package llm

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/llm/claude"
	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

// Load reads, renders, parses and validates the TOML config at path, then
// instantiates all referenced LLM clients and returns a Registry.
func Load(ctx context.Context, path string) (*Registry, error) {
	if path == "" {
		return nil, goerr.New("LLM config path is empty (set --llm-config or WARREN_LLM_CONFIG)")
	}

	raw, err := os.ReadFile(path) // #nosec G304 -- path comes from operator-supplied CLI flag
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read LLM config file", goerr.V("path", path))
	}

	rendered, err := renderTemplate(raw)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to render LLM config template", goerr.V("path", path))
	}

	var file File
	if _, err := toml.Decode(string(rendered), &file); err != nil {
		return nil, goerr.Wrap(err, "failed to parse LLM config TOML", goerr.V("path", path))
	}

	if err := validate(&file); err != nil {
		return nil, goerr.Wrap(err, "invalid LLM config", goerr.V("path", path))
	}

	reg, err := build(ctx, &file)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build LLM registry", goerr.V("path", path))
	}

	logUnreferenced(ctx, &file)
	logging.From(ctx).Info("LLM registry loaded", "registry", reg)

	return reg, nil
}

// renderTemplate runs text/template with .Env exposing os environment.
// Missing keys produce errors via missingkey=error. Empty env values pass
// through and are caught downstream by section-level validation (e.g. an
// empty api_key is rejected by validateClaude / validateGemini).
func renderTemplate(raw []byte) ([]byte, error) {
	tmpl, err := template.New("llm-config").Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return nil, goerr.Wrap(err, "template parse error")
	}

	data := map[string]any{"Env": envToMap(os.Environ())}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, goerr.Wrap(err, "template execute error")
	}
	return buf.Bytes(), nil
}

func envToMap(env []string) map[string]string {
	out := make(map[string]string, len(env))
	for _, kv := range env {
		i := strings.IndexByte(kv, '=')
		if i < 0 {
			continue
		}
		out[kv[:i]] = kv[i+1:]
	}
	return out
}

// validate aggregates all validation errors into a single goerr.Join.
func validate(f *File) error {
	var errs []error

	if f.Agent == nil {
		errs = append(errs, goerr.New("[agent] section is required"))
	}
	if len(f.LLMs) == 0 {
		errs = append(errs, goerr.New("at least one [[llm]] entry is required"))
	}
	if f.Embedding == nil {
		errs = append(errs, goerr.New("[embedding] section is required"))
	}

	idSeen := make(map[string]int)
	for i := range f.LLMs {
		entry := &f.LLMs[i]
		if entry.ID == "" {
			errs = append(errs, goerr.New("[[llm]] entry has empty id", goerr.V("index", i)))
			continue
		}
		idSeen[entry.ID]++
		if err := validateLLMEntry(entry); err != nil {
			errs = append(errs, err)
		}
	}
	for id, n := range idSeen {
		if n > 1 {
			errs = append(errs, goerr.New("[[llm]] id is duplicated", goerr.V("id", id), goerr.V("count", n)))
		}
	}

	if f.Agent != nil {
		errs = append(errs, validateAgent(f.Agent, idSeen)...)
	}
	if f.Embedding != nil {
		errs = append(errs, validateEmbedding(f.Embedding)...)
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func validateAgent(a *AgentConfig, idSeen map[string]int) []error {
	var errs []error
	if a.Main == "" {
		errs = append(errs, goerr.New("[agent].main is required"))
	} else if idSeen[a.Main] == 0 {
		errs = append(errs, goerr.New("[agent].main does not match any [[llm]] entry", goerr.V("main", a.Main)))
	}

	if len(a.Task) == 0 {
		errs = append(errs, goerr.New("[agent].task must contain at least one entry"))
	}
	taskSeen := map[string]int{}
	for _, id := range a.Task {
		taskSeen[id]++
		if id == "" {
			errs = append(errs, goerr.New("[agent].task contains an empty entry"))
			continue
		}
		if idSeen[id] == 0 {
			errs = append(errs, goerr.New("[agent].task entry does not match any [[llm]]", goerr.V("task_id", id)))
		}
	}
	for id, n := range taskSeen {
		if id != "" && n > 1 {
			errs = append(errs, goerr.New("[agent].task contains duplicate id", goerr.V("task_id", id), goerr.V("count", n)))
		}
	}
	return errs
}

func validateLLMEntry(e *LLMConfig) error {
	var errs []error
	tag := goerr.V("entry_id", e.ID)

	if e.Description == "" {
		errs = append(errs, goerr.New("[[llm]] description is required", tag))
	}
	if e.Model == "" {
		errs = append(errs, goerr.New("[[llm]] model is required", tag))
	}

	switch e.Provider {
	case "":
		errs = append(errs, goerr.New("[[llm]] provider is required", tag))
	case ProviderClaude:
		if e.Gemini != nil {
			errs = append(errs, goerr.New("[[llm]] provider=claude must not include gemini section", tag))
		}
		if e.Claude == nil {
			errs = append(errs, goerr.New("[[llm]] provider=claude requires claude section", tag))
		} else {
			if err := validateClaude(e.Claude); err != nil {
				errs = append(errs, goerr.Wrap(err, "[[llm]] claude section invalid", tag))
			}
		}
	case ProviderGemini:
		if e.Claude != nil {
			errs = append(errs, goerr.New("[[llm]] provider=gemini must not include claude section", tag))
		}
		if e.Gemini == nil {
			errs = append(errs, goerr.New("[[llm]] provider=gemini requires gemini section", tag))
		} else {
			if err := validateGemini(e.Gemini); err != nil {
				errs = append(errs, goerr.Wrap(err, "[[llm]] gemini section invalid", tag))
			}
		}
	default:
		errs = append(errs, goerr.New("[[llm]] provider must be 'claude' or 'gemini'", tag, goerr.V("provider", e.Provider)))
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func validateClaude(c *ClaudeOptions) error {
	hasVertex := c.ProjectID != "" || c.Location != ""
	hasAPIKey := c.APIKey != ""

	switch {
	case hasVertex && hasAPIKey:
		return goerr.New("mode is ambiguous: both vertex (project_id/location) and api_key are set")
	case hasVertex:
		var errs []error
		if c.ProjectID == "" {
			errs = append(errs, goerr.New("project_id is required for vertex mode"))
		}
		if c.Location == "" {
			errs = append(errs, goerr.New("location is required for vertex mode"))
		}
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
		return nil
	case hasAPIKey:
		return nil
	default:
		return goerr.New("either (project_id, location) or api_key must be set")
	}
}

func validateGemini(g *GeminiOptions) error {
	if g.APIKey != "" {
		return goerr.New("gemini api_key mode is not supported (gollem does not expose API-key direct mode); use vertex (project_id, location)")
	}
	hasVertex := g.ProjectID != "" || g.Location != ""
	if !hasVertex {
		return goerr.New("(project_id, location) is required for gemini")
	}
	var errs []error
	if g.ProjectID == "" {
		errs = append(errs, goerr.New("project_id is required for gemini vertex"))
	}
	if g.Location == "" {
		errs = append(errs, goerr.New("location is required for gemini vertex"))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func validateEmbedding(e *EmbeddingConfig) []error {
	var errs []error
	if e.Provider != ProviderGemini {
		errs = append(errs, goerr.New("[embedding] provider must be 'gemini'", goerr.V("provider", e.Provider)))
	}
	if e.Model == "" {
		errs = append(errs, goerr.New("[embedding] model is required"))
	}
	if e.APIKey != "" {
		errs = append(errs, goerr.New("[embedding] api_key mode is not supported; use vertex (project_id, location)"))
	}
	if e.APIKey == "" {
		if e.ProjectID == "" {
			errs = append(errs, goerr.New("[embedding] project_id is required"))
		}
		if e.Location == "" {
			errs = append(errs, goerr.New("[embedding] location is required"))
		}
	}
	return errs
}

// build instantiates all clients referenced by [agent] and constructs the Registry.
// Unreferenced [[llm]] entries are not instantiated to avoid spending on dead config.
func build(ctx context.Context, f *File) (*Registry, error) {
	referenced := map[string]struct{}{f.Agent.Main: {}}
	for _, id := range f.Agent.Task {
		referenced[id] = struct{}{}
	}

	configByID := make(map[string]*LLMConfig, len(f.LLMs))
	for i := range f.LLMs {
		configByID[f.LLMs[i].ID] = &f.LLMs[i]
	}

	entries := make(map[string]*LLMEntry, len(referenced))
	var errs []error
	for id := range referenced {
		cfg, ok := configByID[id]
		if !ok {
			// already caught by validate(); defensive guard
			continue
		}
		client, err := newClient(ctx, cfg)
		if err != nil {
			errs = append(errs, goerr.Wrap(err, "failed to construct LLM client", goerr.V("entry_id", id)))
			continue
		}
		entries[id] = &LLMEntry{
			ID:          cfg.ID,
			Description: cfg.Description,
			Provider:    cfg.Provider,
			Model:       cfg.Model,
			Client:      client,
		}
	}

	embedding, err := newEmbeddingClient(ctx, f.Embedding)
	if err != nil {
		errs = append(errs, goerr.Wrap(err, "failed to construct embedding client"))
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	taskSet := make(map[string]struct{}, len(f.Agent.Task))
	for _, id := range f.Agent.Task {
		taskSet[id] = struct{}{}
	}

	return &Registry{
		entries:   entries,
		mainID:    f.Agent.Main,
		taskIDs:   append([]string{}, f.Agent.Task...),
		taskSet:   taskSet,
		embedding: embedding,
	}, nil
}

func newClient(ctx context.Context, cfg *LLMConfig) (gollem.LLMClient, error) {
	switch cfg.Provider {
	case ProviderClaude:
		return newClaudeClient(ctx, cfg.Model, cfg.Claude)
	case ProviderGemini:
		return newGeminiClient(ctx, cfg.Model, cfg.Gemini)
	default:
		return nil, goerr.New("unsupported provider", goerr.V("provider", cfg.Provider))
	}
}

func newClaudeClient(ctx context.Context, model string, opts *ClaudeOptions) (gollem.LLMClient, error) {
	if opts.APIKey != "" {
		c, err := claude.New(ctx, opts.APIKey, claude.WithModel(model))
		if err != nil {
			return nil, goerr.Wrap(err, "claude api-key client init")
		}
		return c, nil
	}
	c, err := claude.NewWithVertex(ctx, opts.Location, opts.ProjectID, claude.WithVertexModel(model))
	if err != nil {
		return nil, goerr.Wrap(err, "claude vertex client init",
			goerr.V("project_id", opts.ProjectID), goerr.V("location", opts.Location))
	}
	return c, nil
}

func newGeminiClient(ctx context.Context, model string, opts *GeminiOptions) (gollem.LLMClient, error) {
	geminiOpts := []gemini.Option{gemini.WithModel(model)}
	if opts.ThinkingBudget != nil {
		// #nosec G115 -- thinking_budget is operator-controlled config, fits in int32 in practice
		geminiOpts = append(geminiOpts, gemini.WithThinkingBudget(int32(*opts.ThinkingBudget)))
	} else {
		// Default: disable thinking to match prior warren behavior.
		geminiOpts = append(geminiOpts, gemini.WithThinkingBudget(0))
	}

	c, err := gemini.New(ctx, opts.ProjectID, opts.Location, geminiOpts...)
	if err != nil {
		return nil, goerr.Wrap(err, "gemini vertex client init",
			goerr.V("project_id", opts.ProjectID), goerr.V("location", opts.Location))
	}
	return c, nil
}

func newEmbeddingClient(ctx context.Context, e *EmbeddingConfig) (gollem.LLMClient, error) {
	c, err := gemini.New(ctx, e.ProjectID, e.Location,
		gemini.WithEmbeddingModel(e.Model),
		// content model not used; provide a sane default
		gemini.WithModel(e.Model),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "embedding (gemini vertex) client init",
			goerr.V("project_id", e.ProjectID), goerr.V("location", e.Location))
	}
	return c, nil
}

// logUnreferenced warns about [[llm]] entries that are neither [agent].main
// nor in [agent].task — they are loaded but never used, indicating likely
// stale config left behind during edits.
func logUnreferenced(ctx context.Context, f *File) {
	referenced := map[string]struct{}{f.Agent.Main: {}}
	for _, id := range f.Agent.Task {
		referenced[id] = struct{}{}
	}
	logger := logging.From(ctx)
	for _, e := range f.LLMs {
		if _, ok := referenced[e.ID]; !ok {
			logger.Warn("[[llm]] entry is defined but unreferenced by [agent]; ignoring",
				slog.String("entry_id", e.ID))
		}
	}
}
