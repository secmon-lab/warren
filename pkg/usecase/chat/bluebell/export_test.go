package bluebell

import (
	"context"

	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
)

// ExportResolveIntent exposes resolveIntent for testing.
func ExportResolveIntent(ctx context.Context, c *BluebellChat, message string, chatCtx *chatModel.ChatContext) *ResolvedIntent {
	resolved, _ := c.resolveIntent(ctx, message, chatCtx)
	return resolved
}

// ExportGenerateSystemPrompt exposes generateSystemPrompt for testing via SystemPromptData.
func ExportGenerateSystemPrompt(ctx context.Context, data SystemPromptData) (string, error) {
	return prompt.GenerateWithStruct(ctx, systemPromptTemplate, data)
}
