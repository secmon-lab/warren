package eval_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/usecase/eval"
)

func TestResponseStore_LookupPreDefined(t *testing.T) {
	dir := t.TempDir()
	responsesDir := filepath.Join(dir, "responses")
	gt.NoError(t, os.MkdirAll(responsesDir, 0o750))

	// Write a pre-defined response file
	content := `{"tool_name":"virustotal","args":{"ip":"1.2.3.4"},"response":{"malicious":true}}`
	gt.NoError(t, os.WriteFile(filepath.Join(responsesDir, "virustotal__test.json"), []byte(content), 0o640))

	store, err := eval.NewResponseStore(responsesDir)
	gt.NoError(t, err)

	// Lookup should find it
	result := store.Lookup("virustotal", map[string]any{"ip": "1.2.3.4"})
	gt.V(t, result).NotNil()
	gt.V(t, result.Response["malicious"]).Equal(true)

	// Lookup with different args should return nil
	result = store.Lookup("virustotal", map[string]any{"ip": "5.6.7.8"})
	gt.V(t, result).Nil()
}

func TestResponseStore_Save(t *testing.T) {
	dir := t.TempDir()
	responsesDir := filepath.Join(dir, "responses")

	store, err := eval.NewResponseStore(responsesDir)
	gt.NoError(t, err)

	args := map[string]any{"query": "SELECT 1"}
	response := map[string]any{"rows": []any{map[string]any{"col": 1.0}}}

	gt.NoError(t, store.Save("bigquery_query", args, response))

	// Should be findable after save
	result := store.Lookup("bigquery_query", args)
	gt.V(t, result).NotNil()
	gt.V(t, result.ToolName).Equal("bigquery_query")

	// Should persist to disk
	entries, err := os.ReadDir(responsesDir)
	gt.NoError(t, err)
	gt.V(t, len(entries)).Equal(1)
}

func TestResponseStore_ArgsHashDeterministic(t *testing.T) {
	dir := t.TempDir()
	responsesDir := filepath.Join(dir, "responses")

	store, err := eval.NewResponseStore(responsesDir)
	gt.NoError(t, err)

	// Save with one key order
	args1 := map[string]any{"b": "2", "a": "1"}
	gt.NoError(t, store.Save("tool", args1, map[string]any{"ok": true}))

	// Lookup with different key order should still match
	args2 := map[string]any{"a": "1", "b": "2"}
	result := store.Lookup("tool", args2)
	gt.V(t, result).NotNil()
	gt.V(t, result.Response["ok"]).Equal(true)
}

func TestResponseStore_NilArgsExcluded(t *testing.T) {
	dir := t.TempDir()
	responsesDir := filepath.Join(dir, "responses")

	store, err := eval.NewResponseStore(responsesDir)
	gt.NoError(t, err)

	// Save without nil
	gt.NoError(t, store.Save("tool", map[string]any{"key": "val"}, map[string]any{"ok": true}))

	// Lookup with extra nil key should match
	result := store.Lookup("tool", map[string]any{"key": "val", "extra": nil})
	gt.V(t, result).NotNil()
}
