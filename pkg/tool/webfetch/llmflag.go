package webfetch

import (
	"context"
	"strconv"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/llm/claude"
	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/m-mizutani/gollem/llm/openai"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

const (
	providerGemini = "gemini"
	providerClaude = "claude"
	providerOpenAI = "openai"

	argProjectID   = "project_id"
	argLocation    = "location"
	argTemperature = "temperature"
)

// parseLLMArgs parses a "key=value,key=value" string into a map.
// Empty input yields an empty (non-nil) map.
func parseLLMArgs(raw string) (map[string]string, error) {
	result := map[string]string{}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return result, nil
	}

	for _, token := range strings.Split(trimmed, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			return nil, goerr.New("empty args token",
				goerr.T(errutil.TagValidation),
				goerr.V("raw", raw))
		}
		idx := strings.Index(token, "=")
		if idx < 0 {
			return nil, goerr.New("args token missing '='",
				goerr.T(errutil.TagValidation),
				goerr.V("token", token))
		}
		key := strings.TrimSpace(token[:idx])
		value := strings.TrimSpace(token[idx+1:])
		if key == "" {
			return nil, goerr.New("empty args key",
				goerr.T(errutil.TagValidation),
				goerr.V("token", token))
		}
		if _, exists := result[key]; exists {
			return nil, goerr.New("duplicate args key",
				goerr.T(errutil.TagValidation),
				goerr.V("key", key))
		}
		result[key] = value
	}
	return result, nil
}

// buildLLMClient constructs a gollem.LLMClient from the parsed flags.
//
// Provider dispatch:
//   - "gemini" : Vertex AI Gemini. Requires project_id and location in args.
//   - "claude" : Vertex AI Claude (project_id+location in args) OR direct
//     Anthropic API (apiKey present). The route is auto-detected from the
//     presence of these inputs. Mixing both is an error (ambiguous).
//   - "openai" : OpenAI direct. Requires apiKey.
//
// Recognized args keys are enforced per provider; unknown keys produce an
// errutil.TagValidation error so misconfigurations surface at start-up.
func buildLLMClient(ctx context.Context, provider, model string, args map[string]string, apiKey string) (gollem.LLMClient, error) {
	if model == "" {
		return nil, goerr.New("model is required",
			goerr.T(errutil.TagValidation),
			goerr.V("provider", provider))
	}

	switch provider {
	case providerGemini:
		return buildGeminiClient(ctx, model, args, apiKey)
	case providerClaude:
		return buildClaudeClient(ctx, model, args, apiKey)
	case providerOpenAI:
		return buildOpenAIClient(ctx, model, args, apiKey)
	default:
		return nil, goerr.New("unknown llm provider",
			goerr.T(errutil.TagValidation),
			goerr.V("provider", provider))
	}
}

func buildGeminiClient(ctx context.Context, model string, args map[string]string, apiKey string) (gollem.LLMClient, error) {
	if apiKey != "" {
		return nil, goerr.New("api-key is not used for gemini (Vertex AI)",
			goerr.T(errutil.TagValidation),
			goerr.V("provider", providerGemini))
	}
	if err := checkKnownArgs(providerGemini, args, argProjectID, argLocation, argTemperature); err != nil {
		return nil, err
	}

	projectID := args[argProjectID]
	location := args[argLocation]
	if projectID == "" || location == "" {
		return nil, goerr.New("gemini requires project_id and location args",
			goerr.T(errutil.TagValidation),
			goerr.V("project_id", projectID),
			goerr.V("location", location))
	}

	opts := []gemini.Option{gemini.WithModel(model)}
	if raw, ok := args[argTemperature]; ok {
		temp, err := parseTemperature(raw)
		if err != nil {
			return nil, err
		}
		opts = append(opts, gemini.WithTemperature(float32(temp)))
	}

	client, err := gemini.New(ctx, projectID, location, opts...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to construct gemini client",
			goerr.V("provider", providerGemini),
			goerr.V("model", model))
	}
	return client, nil
}

func buildClaudeClient(ctx context.Context, model string, args map[string]string, apiKey string) (gollem.LLMClient, error) {
	if err := checkKnownArgs(providerClaude, args, argProjectID, argLocation, argTemperature); err != nil {
		return nil, err
	}

	projectID := args[argProjectID]
	location := args[argLocation]
	hasVertexArg := projectID != "" || location != ""
	hasAPIKey := apiKey != ""

	if hasVertexArg && hasAPIKey {
		return nil, goerr.New("claude route is ambiguous: both Vertex args (project_id/location) and api-key are set",
			goerr.T(errutil.TagValidation),
			goerr.V("provider", providerClaude))
	}
	if !hasVertexArg && !hasAPIKey {
		return nil, goerr.New("claude route unspecified: set api-key for Anthropic direct API, or project_id+location for Vertex AI",
			goerr.T(errutil.TagValidation),
			goerr.V("provider", providerClaude))
	}

	if hasVertexArg {
		if projectID == "" || location == "" {
			return nil, goerr.New("claude Vertex route requires both project_id and location",
				goerr.T(errutil.TagValidation),
				goerr.V("project_id", projectID),
				goerr.V("location", location))
		}
		opts := []claude.VertexOption{claude.WithVertexModel(model)}
		if raw, ok := args[argTemperature]; ok {
			temp, err := parseTemperature(raw)
			if err != nil {
				return nil, err
			}
			opts = append(opts, claude.WithVertexTemperature(temp))
		}
		client, err := claude.NewWithVertex(ctx, location, projectID, opts...)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to construct claude Vertex client",
				goerr.V("provider", providerClaude),
				goerr.V("route", "vertex"),
				goerr.V("model", model))
		}
		return client, nil
	}

	// Anthropic direct route.
	opts := []claude.Option{claude.WithModel(model)}
	if raw, ok := args[argTemperature]; ok {
		temp, err := parseTemperature(raw)
		if err != nil {
			return nil, err
		}
		opts = append(opts, claude.WithTemperature(temp))
	}
	client, err := claude.New(ctx, apiKey, opts...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to construct claude direct client",
			goerr.V("provider", providerClaude),
			goerr.V("route", "anthropic"),
			goerr.V("model", model))
	}
	return client, nil
}

func buildOpenAIClient(ctx context.Context, model string, args map[string]string, apiKey string) (gollem.LLMClient, error) {
	if err := checkKnownArgs(providerOpenAI, args, argTemperature); err != nil {
		return nil, err
	}
	if apiKey == "" {
		return nil, goerr.New("openai requires api-key",
			goerr.T(errutil.TagValidation),
			goerr.V("provider", providerOpenAI))
	}

	opts := []openai.Option{openai.WithModel(model)}
	if raw, ok := args[argTemperature]; ok {
		temp, err := parseTemperature(raw)
		if err != nil {
			return nil, err
		}
		opts = append(opts, openai.WithTemperature(float32(temp)))
	}

	client, err := openai.New(ctx, apiKey, opts...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to construct openai client",
			goerr.V("provider", providerOpenAI),
			goerr.V("model", model))
	}
	return client, nil
}

func checkKnownArgs(provider string, args map[string]string, allowed ...string) error {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, k := range allowed {
		allowedSet[k] = struct{}{}
	}
	for k := range args {
		if _, ok := allowedSet[k]; !ok {
			return goerr.New("unknown args key for provider",
				goerr.T(errutil.TagValidation),
				goerr.V("provider", provider),
				goerr.V("key", k))
		}
	}
	return nil
}

// parseTemperature parses the temperature arg and validates the range. The
// accepted range [0.0, 2.0] covers all currently supported providers (OpenAI
// and Gemini allow up to 2.0; Anthropic up to 1.0 — values that satisfy this
// check but exceed a provider's own cap will still fail at the start-up ping,
// just with a less explicit error).
func parseTemperature(raw string) (float64, error) {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to parse temperature as float",
			goerr.T(errutil.TagValidation),
			goerr.V("raw", raw))
	}
	if v < 0 || v > 2 {
		return 0, goerr.New("temperature must be between 0.0 and 2.0",
			goerr.T(errutil.TagValidation),
			goerr.V("value", v))
	}
	return v, nil
}

// pingLLMClient performs a minimal generate request (max_tokens=1) so that
// misconfigurations such as a bad API key, wrong project_id/location, or a
// typo in the model name surface at start-up rather than at first use.
func pingLLMClient(ctx context.Context, client gollem.LLMClient) error {
	if client == nil {
		return goerr.New("llm client is nil", goerr.T(errutil.TagInternal))
	}

	session, err := client.NewSession(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to start llm session for ping",
			goerr.T(errutil.TagLLMError))
	}

	if _, err := session.Generate(ctx,
		[]gollem.Input{gollem.Text("ping")},
		gollem.WithMaxTokens(1),
	); err != nil {
		return goerr.Wrap(err, "llm ping request failed",
			goerr.T(errutil.TagLLMError))
	}
	return nil
}
