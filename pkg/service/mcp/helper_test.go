package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	helpertransport "github.com/secmon-lab/warren/pkg/service/mcp"
)

func writeHelperScript(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	gt.NoError(t, os.WriteFile(path, []byte(content), 0o755))
	return path
}

func TestHelperTransport_BasicHeaderInjection(t *testing.T) {
	script := writeHelperScript(t, "helper.sh", `#!/bin/sh
echo '{"headers":{"Authorization":"Bearer test-token","X-Custom":"custom-value"}}'
`)

	cfg := helpertransport.HelperConfig{
		Command: script,
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gt.Value(t, r.Header.Get("Authorization")).Equal("Bearer test-token")
		gt.Value(t, r.Header.Get("X-Custom")).Equal("custom-value")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	transport := helpertransport.NewHelperTransport(cfg, nil, nil)
	client := &http.Client{Transport: transport}

	resp, err := client.Get(backend.URL)
	gt.NoError(t, err)
	defer func() { gt.NoError(t, resp.Body.Close()) }()
	gt.Value(t, resp.StatusCode).Equal(http.StatusOK)
}

func TestHelperTransport_StaticHeadersMerge(t *testing.T) {
	script := writeHelperScript(t, "helper.sh", `#!/bin/sh
echo '{"headers":{"Authorization":"Bearer dynamic","X-Helper":"from-helper"}}'
`)

	cfg := helpertransport.HelperConfig{
		Command: script,
	}
	staticHeaders := map[string]string{
		"Authorization": "Bearer static-will-be-overridden",
		"X-Static":      "static-value",
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Helper overrides static Authorization
		gt.Value(t, r.Header.Get("Authorization")).Equal("Bearer dynamic")
		// Static header preserved
		gt.Value(t, r.Header.Get("X-Static")).Equal("static-value")
		// Helper-only header added
		gt.Value(t, r.Header.Get("X-Helper")).Equal("from-helper")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	transport := helpertransport.NewHelperTransport(cfg, staticHeaders, nil)
	client := &http.Client{Transport: transport}

	resp, err := client.Get(backend.URL)
	gt.NoError(t, err)
	defer func() { gt.NoError(t, resp.Body.Close()) }()
	gt.Value(t, resp.StatusCode).Equal(http.StatusOK)
}

func TestHelperTransport_ExpiresAtCache(t *testing.T) {
	// Script that writes a counter file to track how many times it's been called
	dir := t.TempDir()
	counterFile := filepath.Join(dir, "counter")
	gt.NoError(t, os.WriteFile(counterFile, []byte("0"), 0o644))

	expiresAt := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	script := writeHelperScript(t, "helper.sh", `#!/bin/sh
COUNTER_FILE="`+counterFile+`"
COUNT=$(cat "$COUNTER_FILE")
COUNT=$((COUNT + 1))
echo "$COUNT" > "$COUNTER_FILE"
echo '{"headers":{"Authorization":"Bearer cached-token"},"expires_at":"`+expiresAt+`"}'
`)

	cfg := helpertransport.HelperConfig{
		Command: script,
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gt.Value(t, r.Header.Get("Authorization")).Equal("Bearer cached-token")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	transport := helpertransport.NewHelperTransport(cfg, nil, nil)
	client := &http.Client{Transport: transport}

	// Make multiple requests
	for range 3 {
		resp, err := client.Get(backend.URL)
		gt.NoError(t, err)
		gt.NoError(t, resp.Body.Close())
	}

	// The helper should have been called only once (cached)
	data, err := os.ReadFile(counterFile)
	gt.NoError(t, err)
	gt.Value(t, string(data)).Equal("1\n")
}

func TestHelperTransport_ExpiredCacheRefreshes(t *testing.T) {
	dir := t.TempDir()
	counterFile := filepath.Join(dir, "counter")
	gt.NoError(t, os.WriteFile(counterFile, []byte("0"), 0o644))

	// Use an already-expired time
	expiresAt := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	script := writeHelperScript(t, "helper.sh", `#!/bin/sh
COUNTER_FILE="`+counterFile+`"
COUNT=$(cat "$COUNTER_FILE")
COUNT=$((COUNT + 1))
echo "$COUNT" > "$COUNTER_FILE"
echo '{"headers":{"Authorization":"Bearer token"},"expires_at":"`+expiresAt+`"}'
`)

	cfg := helpertransport.HelperConfig{
		Command: script,
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	transport := helpertransport.NewHelperTransport(cfg, nil, nil)
	client := &http.Client{Transport: transport}

	// Make 3 requests — each should re-run the helper (expired cache)
	for range 3 {
		resp, err := client.Get(backend.URL)
		gt.NoError(t, err)
		gt.NoError(t, resp.Body.Close())
	}

	data, err := os.ReadFile(counterFile)
	gt.NoError(t, err)
	gt.Value(t, string(data)).Equal("3\n")
}

func TestHelperTransport_CommandFailure(t *testing.T) {
	script := writeHelperScript(t, "helper.sh", `#!/bin/sh
echo "something went wrong" >&2
exit 1
`)

	cfg := helpertransport.HelperConfig{
		Command: script,
	}

	transport := helpertransport.NewHelperTransport(cfg, nil, nil)
	client := &http.Client{Transport: transport}

	_, err := client.Get("http://localhost:1") //nolint:bodyclose
	gt.Error(t, err)
}

func TestHelperTransport_InvalidJSON(t *testing.T) {
	script := writeHelperScript(t, "helper.sh", `#!/bin/sh
echo 'not valid json'
`)

	cfg := helpertransport.HelperConfig{
		Command: script,
	}

	transport := helpertransport.NewHelperTransport(cfg, nil, nil)
	client := &http.Client{Transport: transport}

	_, err := client.Get("http://localhost:1") //nolint:bodyclose
	gt.Error(t, err)
}

func TestHelperTransport_MissingHeadersField(t *testing.T) {
	script := writeHelperScript(t, "helper.sh", `#!/bin/sh
echo '{"expires_at":"2026-12-31T00:00:00Z"}'
`)

	cfg := helpertransport.HelperConfig{
		Command: script,
	}

	transport := helpertransport.NewHelperTransport(cfg, nil, nil)
	client := &http.Client{Transport: transport}

	_, err := client.Get("http://localhost:1") //nolint:bodyclose
	gt.Error(t, err)
}

func TestHelperTransport_EnvVars(t *testing.T) {
	script := writeHelperScript(t, "helper.sh", `#!/bin/sh
echo "{\"headers\":{\"X-Env\":\"$MY_VAR\"}}"
`)

	cfg := helpertransport.HelperConfig{
		Command: script,
		Env: map[string]string{
			"MY_VAR": "env-value",
		},
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gt.Value(t, r.Header.Get("X-Env")).Equal("env-value")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	transport := helpertransport.NewHelperTransport(cfg, nil, nil)
	client := &http.Client{Transport: transport}

	resp, err := client.Get(backend.URL)
	gt.NoError(t, err)
	defer func() { gt.NoError(t, resp.Body.Close()) }()
	gt.Value(t, resp.StatusCode).Equal(http.StatusOK)
}

func TestHelperTransport_ConcurrentAccess(t *testing.T) {
	script := writeHelperScript(t, "helper.sh", `#!/bin/sh
echo '{"headers":{"Authorization":"Bearer concurrent-token"}}'
`)

	cfg := helpertransport.HelperConfig{
		Command: script,
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gt.Value(t, r.Header.Get("Authorization")).Equal("Bearer concurrent-token")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	transport := helpertransport.NewHelperTransport(cfg, nil, nil)
	client := &http.Client{Transport: transport}

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(backend.URL)
			gt.NoError(t, err)
			gt.NoError(t, resp.Body.Close())
		}()
	}
	wg.Wait()
}

func TestHelperTransport_CommandWithArgs(t *testing.T) {
	script := writeHelperScript(t, "helper.sh", `#!/bin/sh
echo "{\"headers\":{\"X-Arg\":\"$1\"}}"
`)

	cfg := helpertransport.HelperConfig{
		Command: script,
		Args:    []string{"my-argument"},
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gt.Value(t, r.Header.Get("X-Arg")).Equal("my-argument")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	transport := helpertransport.NewHelperTransport(cfg, nil, nil)
	client := &http.Client{Transport: transport}

	resp, err := client.Get(backend.URL)
	gt.NoError(t, err)
	defer func() { gt.NoError(t, resp.Body.Close()) }()
	gt.Value(t, resp.StatusCode).Equal(http.StatusOK)
}
