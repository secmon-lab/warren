package llm

import (
	"maps"

	"github.com/m-mizutani/gollem"
)

// NewRegistryForTest builds a Registry directly from in-memory entries,
// bypassing TOML loading. Used by tests across the codebase to inject
// mock LLMClients. This is exposed (not in _test.go) because cross-package
// test files (e.g. pkg/usecase/...) need to reference it.
func NewRegistryForTest(mainID string, taskIDs []string, entries map[string]*LLMEntry, embedding gollem.LLMClient) *Registry {
	taskSet := make(map[string]struct{}, len(taskIDs))
	for _, id := range taskIDs {
		taskSet[id] = struct{}{}
	}
	dup := make(map[string]*LLMEntry, len(entries))
	maps.Copy(dup, entries)
	return &Registry{
		entries:   dup,
		mainID:    mainID,
		taskIDs:   append([]string{}, taskIDs...),
		taskSet:   taskSet,
		embedding: embedding,
	}
}

// NewLLMEntryForTest constructs an LLMEntry inline.
func NewLLMEntryForTest(id, description, provider, model string, client gollem.LLMClient) *LLMEntry {
	return &LLMEntry{
		ID:          id,
		Description: description,
		Provider:    provider,
		Model:       model,
		Client:      client,
	}
}

// SingleClientRegistryForTest wraps a single LLMClient into a registry where
// it is both the main entry and the only task entry. This is a convenience
// for legacy tests that don't care about per-task LLM routing.
func SingleClientRegistryForTest(client gollem.LLMClient) *Registry {
	entry := NewLLMEntryForTest("test", "default test llm", "claude", "test-model", client)
	return NewRegistryForTest(
		"test",
		[]string{"test"},
		map[string]*LLMEntry{"test": entry},
		client,
	)
}
