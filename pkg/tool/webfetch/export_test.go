package webfetch

import (
	"net/http"

	"github.com/gollem-dev/gollem"
)

// ParseLLMArgs is the test-only export of parseLLMArgs.
var ParseLLMArgs = parseLLMArgs

// BuildLLMClient is the test-only export of buildLLMClient.
var BuildLLMClient = buildLLMClient

// PingLLMClient is the test-only export of pingLLMClient.
var PingLLMClient = pingLLMClient

// SetHTTPClient injects a custom http.Client for tests (e.g. with httptest server).
func (x *Action) SetHTTPClient(c *http.Client) {
	x.client = c
}

// SetLLMClient injects an LLM client for tests, bypassing the flag-driven
// build+ping path used in production. Configure will use this client when
// building the inner ToolSet.
func (x *Action) SetLLMClient(c gollem.LLMClient) {
	x.llmClient = c
}

// LLMClient returns the currently configured LLM client (test-only accessor).
func (x *Action) LLMClient() gollem.LLMClient {
	return x.llmClient
}

// LLMProvider returns the configured provider name (test-only accessor).
func (x *Action) LLMProvider() string {
	return x.llmProvider
}

// SetLLMFlags overrides the flag-bound fields for tests so we don't need a
// full CLI flag-parse round trip.
func (x *Action) SetLLMFlags(provider, model, args, apiKey string) {
	x.llmProvider = provider
	x.llmModel = model
	x.llmArgs = args
	x.llmAPIKey = apiKey
}
