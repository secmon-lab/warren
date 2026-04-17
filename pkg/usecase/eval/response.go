package eval

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/eval"
)

// ResponseStore manages tool response files in the responses/ directory.
// It provides lookup by tool name + args, and saves newly generated responses.
type ResponseStore struct {
	dir   string
	mu    sync.RWMutex
	index map[string]*eval.ResponseFile // key: tool_name + args_hash -> response
}

// NewResponseStore creates a ResponseStore and indexes existing response files.
func NewResponseStore(responsesDir string) (*ResponseStore, error) {
	store := &ResponseStore{
		dir:   responsesDir,
		index: make(map[string]*eval.ResponseFile),
	}

	if err := os.MkdirAll(responsesDir, 0o750); err != nil {
		return nil, goerr.Wrap(err, "failed to create responses directory",
			goerr.V("dir", responsesDir))
	}

	entries, err := os.ReadDir(responsesDir)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read responses directory",
			goerr.V("dir", responsesDir))
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Clean(filepath.Join(responsesDir, entry.Name())))
		if err != nil {
			return nil, goerr.Wrap(err, "failed to read response file",
				goerr.V("file", entry.Name()))
		}

		var rf eval.ResponseFile
		if err := json.Unmarshal(data, &rf); err != nil {
			return nil, goerr.Wrap(err, "failed to parse response file",
				goerr.V("file", entry.Name()))
		}

		key := responseKey(rf.ToolName, rf.Args)
		store.index[key] = &rf
	}

	return store, nil
}

// Lookup finds a cached response for the given tool call.
// Returns nil if no matching response exists.
func (s *ResponseStore) Lookup(toolName string, args map[string]any) *eval.ResponseFile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := responseKey(toolName, args)
	return s.index[key]
}

// Save persists a response to the responses/ directory and adds it to the index.
func (s *ResponseStore) Save(toolName string, args map[string]any, response map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rf := &eval.ResponseFile{
		ToolName: toolName,
		Args:     args,
		Response: response,
	}

	key := responseKey(toolName, args)
	filename := fmt.Sprintf("%s__%s.json", toolName, argsHash(args))
	filePath := filepath.Join(s.dir, filename)

	data, err := json.MarshalIndent(rf, "", "  ")
	if err != nil {
		return goerr.Wrap(err, "failed to marshal response file")
	}

	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return goerr.Wrap(err, "failed to write response file",
			goerr.V("path", filePath))
	}

	s.index[key] = rf
	return nil
}

// responseKey generates a lookup key from tool name and args.
func responseKey(toolName string, args map[string]any) string {
	return toolName + "__" + argsHash(args)
}

// argsHash generates a deterministic hash of tool arguments.
// Rules:
// 1. Map keys sorted recursively
// 2. Numbers kept as-is (float64 from JSON)
// 3. Arrays preserve order
// 4. nil values excluded
// 5. Normalized JSON -> SHA256 first 8 hex chars
func argsHash(args map[string]any) string {
	normalized := normalizeValue(args)
	data, err := json.Marshal(normalized)
	if err != nil {
		// Fallback: hash the fmt representation
		data = fmt.Appendf(nil, "%v", args)
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:4])
}

// normalizeValue recursively normalizes a value for deterministic JSON serialization.
func normalizeValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return normalizeMap(val)
	case []any:
		result := make([]any, 0, len(val))
		for _, item := range val {
			result = append(result, normalizeValue(item))
		}
		return result
	default:
		return v
	}
}

// normalizeMap returns an ordered representation of a map.
// Keys are sorted alphabetically, nil values are excluded.
func normalizeMap(m map[string]any) [][2]any {
	keys := make([]string, 0, len(m))
	for k, v := range m {
		if v == nil {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([][2]any, 0, len(keys))
	for _, k := range keys {
		result = append(result, [2]any{k, normalizeValue(m[k])})
	}
	return result
}
