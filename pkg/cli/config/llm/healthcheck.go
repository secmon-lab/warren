package llm

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

const (
	healthCheckTimeout    = 30 * time.Second
	healthCheckPrompt     = "ping"
	healthCheckEmbeddings = 256
)

// HealthCheck pings every LLM referenced by [agent].main + [agent].task
// (deduplicated) and the embedding client, all in parallel. All goroutines
// run to completion; failures are aggregated and returned via errors.Join.
func (r *Registry) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	logger := logging.From(ctx)

	targets := r.referencedLLMs()

	type result struct {
		id      string
		latency time.Duration
		err     error
	}
	results := make(chan result, len(targets)+1)

	var wg sync.WaitGroup
	for _, e := range targets {
		entry := e
		wg.Go(func() {
			start := time.Now()
			err := pingLLM(ctx, entry)
			results <- result{id: entry.ID, latency: time.Since(start), err: err}
		})
	}
	wg.Go(func() {
		start := time.Now()
		err := pingEmbedding(ctx, r.embedding)
		results <- result{id: "embedding", latency: time.Since(start), err: err}
	})

	wg.Wait()
	close(results)

	var errs []error
	for res := range results {
		if res.err != nil {
			errs = append(errs, res.err)
			logger.Error("health check failed",
				"target", res.id, "latency", res.latency, "error", res.err)
		} else {
			logger.Info("health check ok",
				"target", res.id, "latency", res.latency)
		}
	}
	if len(errs) > 0 {
		return goerr.Wrap(errors.Join(errs...), "llm health check failed")
	}
	logger.Info("health check passed",
		"llm_count", len(targets), "embedding", true)
	return nil
}

// referencedLLMs returns the deduplicated set of LLMEntry objects that are
// reachable from [agent].main + [agent].task.
func (r *Registry) referencedLLMs() []*LLMEntry {
	seen := map[string]struct{}{}
	var out []*LLMEntry

	if e, ok := r.entries[r.mainID]; ok {
		seen[e.ID] = struct{}{}
		out = append(out, e)
	}
	for _, id := range r.taskIDs {
		if _, dup := seen[id]; dup {
			continue
		}
		if e, ok := r.entries[id]; ok {
			seen[id] = struct{}{}
			out = append(out, e)
		}
	}
	return out
}

func pingLLM(ctx context.Context, e *LLMEntry) error {
	session, err := e.Client.NewSession(ctx)
	if err != nil {
		return goerr.Wrap(err, "health check failed",
			goerr.V("entry_id", e.ID),
			goerr.V("provider", e.Provider),
			goerr.V("model", e.Model))
	}
	if _, err := session.Generate(ctx, []gollem.Input{gollem.Text(healthCheckPrompt)}); err != nil {
		return goerr.Wrap(err, "health check failed",
			goerr.V("entry_id", e.ID),
			goerr.V("provider", e.Provider),
			goerr.V("model", e.Model))
	}
	return nil
}

func pingEmbedding(ctx context.Context, c gollem.LLMClient) error {
	if _, err := c.GenerateEmbedding(ctx, healthCheckEmbeddings, []string{healthCheckPrompt}); err != nil {
		return goerr.Wrap(err, "embedding health check failed")
	}
	return nil
}
