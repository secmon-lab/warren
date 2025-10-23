package llm

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/middleware/compacter"
)

// NewCompactionMiddleware creates a content block middleware for automatic conversation history compaction.
// It compresses the oldest 70% of conversation history when the history grows too large,
// reducing token usage and preventing context window overflow.
func NewCompactionMiddleware(llmClient gollem.LLMClient, logger *slog.Logger) gollem.ContentBlockMiddleware {
	return compacter.NewContentBlockMiddleware(
		llmClient,
		compacter.WithCompactRatio(0.7),
		compacter.WithMaxRetries(3),
		compacter.WithLogger(logger),
		compacter.WithCompactionHook(func(ctx context.Context, event *compacter.CompactionEvent) {
			logger.Info("conversation history compacted",
				"original_size", event.OriginalDataSize,
				"compacted_size", event.CompactedDataSize,
				"input_tokens", event.InputTokens,
				"output_tokens", event.OutputTokens,
				"compression_ratio", float64(event.CompactedDataSize)/float64(event.OriginalDataSize))
		}),
	)
}

// NewCompactionStreamMiddleware creates a content stream middleware for automatic conversation history compaction.
func NewCompactionStreamMiddleware(llmClient gollem.LLMClient) gollem.ContentStreamMiddleware {
	return compacter.NewContentStreamMiddleware(llmClient)
}
