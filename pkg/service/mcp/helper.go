package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

const defaultHelperTimeout = 30 * time.Second

// HelperOutput represents the JSON output from a credential helper command.
type HelperOutput struct {
	Headers   map[string]string `json:"headers"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
}

// HelperConfig represents the configuration for a credential helper command.
type HelperConfig struct {
	Command string
	Args    []string
	Env     map[string]string
}

// HelperTransport is a custom http.RoundTripper that executes a credential
// helper command to dynamically generate HTTP headers for MCP server requests.
type HelperTransport struct {
	base          http.RoundTripper
	cfg           HelperConfig
	staticHeaders map[string]string
	timeout       time.Duration

	// cache
	mu            sync.Mutex
	cachedHeaders map[string]string
	expiresAt     time.Time
}

// NewHelperTransport creates a new HelperTransport.
// staticHeaders are applied first, then helper-generated headers override them.
// If base is nil, http.DefaultTransport is used.
func NewHelperTransport(cfg HelperConfig, staticHeaders map[string]string, base http.RoundTripper) *HelperTransport {
	if base == nil {
		base = http.DefaultTransport
	}

	return &HelperTransport{
		base:          base,
		cfg:           cfg,
		staticHeaders: staticHeaders,
		timeout:       defaultHelperTimeout,
	}
}

// RoundTrip implements http.RoundTripper. It retrieves headers from the
// credential helper (using cache when valid) and injects them into the request.
func (t *HelperTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	headers, err := t.getHeaders(req.Context())
	if err != nil {
		return nil, err
	}

	// Clone the request to avoid mutating the original
	clone := req.Clone(req.Context())

	// Apply static headers first
	for k, v := range t.staticHeaders {
		clone.Header.Set(k, v)
	}

	// Apply helper headers (overrides static)
	for k, v := range headers {
		clone.Header.Set(k, v)
	}

	return t.base.RoundTrip(clone)
}

// getHeaders returns cached headers if still valid, or runs the helper command.
func (t *HelperTransport) getHeaders(ctx context.Context) (map[string]string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Return cached headers if still valid
	if t.cachedHeaders != nil && !t.expiresAt.IsZero() && time.Now().Before(t.expiresAt) {
		return t.cachedHeaders, nil
	}

	// Run the helper command
	if err := t.runHelper(ctx); err != nil {
		return nil, err
	}

	return t.cachedHeaders, nil
}

// runHelper executes the credential helper command and updates the cache.
func (t *HelperTransport) runHelper(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, t.cfg.Command, t.cfg.Args...) // #nosec G204

	// Set environment variables
	if len(t.cfg.Env) > 0 {
		env := cmd.Environ()
		for k, v := range t.cfg.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logger := logging.From(ctx)

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			logger.Warn("credential helper stderr", "stderr", stderrStr)
		}
		return goerr.Wrap(err, "credential helper command failed",
			goerr.V("command", t.cfg.Command),
			goerr.V("args", t.cfg.Args),
			goerr.V("stderr", stderrStr),
		)
	}

	// Log stderr for debugging
	if stderrStr := stderr.String(); stderrStr != "" {
		logger.Debug("credential helper stderr", "stderr", stderrStr)
	}

	var output HelperOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return goerr.Wrap(err, "failed to parse credential helper output as JSON",
			goerr.V("command", t.cfg.Command),
		)
	}

	if output.Headers == nil {
		return goerr.New("credential helper output missing required 'headers' field",
			goerr.V("command", t.cfg.Command),
		)
	}

	// Update cache
	t.cachedHeaders = output.Headers
	if output.ExpiresAt != nil {
		t.expiresAt = *output.ExpiresAt
	} else {
		t.expiresAt = time.Time{} // no caching without expires_at
	}

	return nil
}
