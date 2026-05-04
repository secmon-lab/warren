package llm

import (
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
)

// LLMEntry holds the resolved client for a single [[llm]] entry.
type LLMEntry struct {
	ID          string
	Description string
	Provider    string
	Model       string
	Client      gollem.LLMClient
}

// CatalogEntry is the planner-facing view of a task LLM (no client).
type CatalogEntry struct {
	ID          string
	Description string
	Provider    string
	Model       string
}

// Registry holds all LLM clients defined in the TOML config along with the
// role assignments declared in [agent].
type Registry struct {
	entries   map[string]*LLMEntry
	mainID    string
	taskIDs   []string // declared order; deduplicated by validate()
	taskSet   map[string]struct{}
	embedding gollem.LLMClient
}

// Main returns the LLM entry assigned to [agent].main.
func (r *Registry) Main() *LLMEntry {
	return r.entries[r.mainID]
}

// Embedding returns the embedding client.
func (r *Registry) Embedding() gollem.LLMClient {
	return r.embedding
}

// Resolve looks up an LLM by id. Returns sentinel errors if id is empty,
// unknown, or not in the [agent].task allow-list. Wraps each with goerr.V.
func (r *Registry) Resolve(id string) (*LLMEntry, error) {
	if id == "" {
		return nil, goerr.Wrap(ErrEmptyLLMID, "empty llm_id")
	}
	entry, ok := r.entries[id]
	if !ok {
		return nil, goerr.Wrap(ErrUnknownLLMID, "unknown llm_id", goerr.V("llm_id", id))
	}
	if _, allowed := r.taskSet[id]; !allowed {
		return nil, goerr.Wrap(ErrLLMNotInTaskList, "llm_id not in [agent].task allow-list", goerr.V("llm_id", id))
	}
	return entry, nil
}

// Catalog returns the planner-visible catalog in declared order.
func (r *Registry) Catalog() []CatalogEntry {
	out := make([]CatalogEntry, 0, len(r.taskIDs))
	for _, id := range r.taskIDs {
		e, ok := r.entries[id]
		if !ok {
			continue
		}
		out = append(out, CatalogEntry{
			ID:          e.ID,
			Description: e.Description,
			Provider:    e.Provider,
			Model:       e.Model,
		})
	}
	return out
}

// TaskIDs returns the [agent].task ids in declared order.
func (r *Registry) TaskIDs() []string {
	out := make([]string, len(r.taskIDs))
	copy(out, r.taskIDs)
	return out
}

// MainID returns the [agent].main id.
func (r *Registry) MainID() string {
	return r.mainID
}

// LogValue redacts api keys; surfaces only id / provider / model.
func (r *Registry) LogValue() slog.Value {
	mainAttrs := []any{}
	if m := r.Main(); m != nil {
		mainAttrs = []any{slog.String("id", m.ID), slog.String("provider", m.Provider), slog.String("model", m.Model)}
	}
	tasks := make([]any, 0, len(r.taskIDs))
	for _, id := range r.taskIDs {
		if e, ok := r.entries[id]; ok {
			tasks = append(tasks, slog.GroupValue(
				slog.String("id", e.ID),
				slog.String("provider", e.Provider),
				slog.String("model", e.Model),
			))
		}
	}
	return slog.GroupValue(
		slog.Group("main", mainAttrs...),
		slog.Any("tasks", tasks),
	)
}
