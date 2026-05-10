package policy

import "context"

// Snapshot is the content of a policy source at a specific point in time.
// Version is an opaque identifier that changes when Files content changes,
// so callers can decide whether downstream artifacts (compiled clients) need
// to be rebuilt.
type Snapshot struct {
	// Files maps a unique key (e.g. file path or "github://...") to the Rego
	// source. Keys MUST be unique across all sources composed in a single
	// Loader.
	Files map[string]string

	// Version changes whenever Files changes.
	Version string
}

// Source represents an origin of Rego policy contents.
// Implementations are expected to be safe for concurrent use.
type Source interface {
	Snapshot(ctx context.Context) (*Snapshot, error)
}
