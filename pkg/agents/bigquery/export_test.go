package bigquery

import (
	"context"
	"time"

	"github.com/m-mizutani/gollem"
)

// Expose internal types and functions for testing

// GenerateKPTAnalysis is exported for testing
func (a *Agent) GenerateKPTAnalysis(ctx context.Context, query string, resp *gollem.ExecuteResponse, execErr error, duration time.Duration, session gollem.Session) ([]string, []string, []string, error) {
	return a.generateKPTAnalysis(ctx, query, resp, execErr, duration, session)
}
