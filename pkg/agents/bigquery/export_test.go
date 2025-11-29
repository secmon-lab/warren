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

// ExportNewInternalTool creates a new internalTool instance for testing
func ExportNewInternalTool(config *Config, projectID string) gollem.ToolSet {
	return &internalTool{
		config:    config,
		projectID: projectID,
	}
}

// ToolSpec is exported for testing
type ToolSpec = gollem.ToolSpec

// ExportedExtractRecords is exported for testing
func (a *Agent) ExportedExtractRecords(ctx context.Context, originalQuery string, session gollem.Session) ([]map[string]any, error) {
	return a.extractRecords(ctx, originalQuery, session)
}
