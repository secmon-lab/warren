package slack

import (
	"context"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

// Export for testing

type InternalTool = internalTool

// NewInternalToolForTest creates an internalTool for testing
func NewInternalToolForTest(slackClient interfaces.SlackClient, maxLimit int) *internalTool {
	return &internalTool{
		slackClient: slackClient,
		maxLimit:    maxLimit,
	}
}

// ExportedExtractRecords is exported for testing
func (a *Agent) ExportedExtractRecords(ctx context.Context, originalQuery string, session gollem.Session) ([]map[string]any, error) {
	return a.extractRecords(ctx, originalQuery, session)
}
