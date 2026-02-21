package memory

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

//go:embed prompt/reflection.md
var reflectionPromptTemplate string

// reflectionResponse defines the structure for reflection LLM response
type reflectionResponse struct {
	NewClaims       []string `json:"new_claims" description:"Array of newly discovered insights (specific, actionable claims)"`
	HelpfulMemories []string `json:"helpful_memories" description:"Array of memory IDs that were helpful during execution"`
	HarmfulMemories []string `json:"harmful_memories" description:"Array of memory IDs that were harmful or misleading during execution"`
}

// generateReflection generates reflection using LLM from execution history
// This is an internal method used by ExtractAndSaveMemories
func (s *Service) generateReflection(
	ctx context.Context,
	query string,
	usedMemories []*memory.AgentMemory,
	history *gollem.History,
) (*memory.Reflection, error) {
	logger := logging.From(ctx)

	// Generate JSON schema using prompt.ToSchema
	schema := prompt.ToSchema(reflectionResponse{})
	jsonSchemaStr, err := schema.Stringify()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to stringify reflection schema")
	}

	// Format used memories for template
	type memoryInfo struct {
		ID    string
		Query string
		Claim string
		Score float64
	}

	memories := make([]memoryInfo, len(usedMemories))
	for i, mem := range usedMemories {
		memories[i] = memoryInfo{
			ID:    string(mem.ID),
			Query: mem.Query,
			Claim: mem.Claim,
			Score: mem.Score,
		}
	}

	// Format execution history for template
	// Convert gollem.History to readable text format
	var historyBuilder strings.Builder
	if history != nil && len(history.Messages) > 0 {
		for i, message := range history.Messages {
			fmt.Fprintf(&historyBuilder, "### Message %d (%s)\n", i+1, message.Role)

			// Format message contents
			for _, content := range message.Contents {
				switch content.Type {
				case gollem.MessageContentTypeText:
					textContent, err := content.GetTextContent()
					if err == nil {
						historyBuilder.WriteString(textContent.Text)
						historyBuilder.WriteString("\n")
					}
				case gollem.MessageContentTypeToolCall:
					toolCall, err := content.GetToolCallContent()
					if err == nil {
						fmt.Fprintf(&historyBuilder, "Tool Call: %s (ID: %s)\n", toolCall.Name, toolCall.ID)
						if toolCall.Arguments != nil {
							if argsJSON, err := json.Marshal(toolCall.Arguments); err == nil {
								fmt.Fprintf(&historyBuilder, "Arguments: %s\n", string(argsJSON))
							}
						}
					}
				case gollem.MessageContentTypeToolResponse:
					toolResp, err := content.GetToolResponseContent()
					if err == nil {
						fmt.Fprintf(&historyBuilder, "Tool Response: %s (Call ID: %s)\n", toolResp.Name, toolResp.ToolCallID)
						if toolResp.Response != nil {
							if respJSON, err := json.Marshal(toolResp.Response); err == nil {
								fmt.Fprintf(&historyBuilder, "Response: %s\n", string(respJSON))
							}
						}
					}
				}
			}
			historyBuilder.WriteString("\n")
		}
	} else {
		historyBuilder.WriteString("No execution history available\n")
	}

	// Build prompt parameters
	promptParams := map[string]any{
		"TaskQuery":        query,
		"UsedMemories":     memories,
		"ExecutionHistory": historyBuilder.String(),
		"JSONSchema":       jsonSchemaStr,
	}

	// Generate prompt text from template
	promptText, err := prompt.GenerateWithStruct(ctx, reflectionPromptTemplate, promptParams)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate reflection prompt")
	}

	historyCount := 0
	if history != nil {
		historyCount = len(history.Messages)
	}
	logger.Debug("generated reflection prompt",
		"query", query,
		"memory_count", len(usedMemories),
		"history_messages", historyCount)

	// Convert to gollem schema for session creation
	gollemSchema, err := gollem.ToSchema(reflectionResponse{})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate gollem schema")
	}

	// Create a new session for reflection generation
	session, err := s.llmClient.NewSession(ctx,
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
		gollem.WithSessionResponseSchema(gollemSchema),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create reflection session")
	}

	// Generate reflection
	response, err := session.GenerateContent(ctx, gollem.Text(promptText))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate reflection content")
	}

	// Parse response
	if len(response.Texts) == 0 {
		return nil, goerr.New("no response text from LLM")
	}

	var resp reflectionResponse
	if err := json.Unmarshal([]byte(response.Texts[0]), &resp); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal reflection response", goerr.V("text", response.Texts[0]))
	}

	// Convert string IDs to types.AgentMemoryID
	helpfulMemories := make([]types.AgentMemoryID, 0, len(resp.HelpfulMemories))
	for _, id := range resp.HelpfulMemories {
		helpfulMemories = append(helpfulMemories, types.AgentMemoryID(id))
	}

	harmfulMemories := make([]types.AgentMemoryID, 0, len(resp.HarmfulMemories))
	for _, id := range resp.HarmfulMemories {
		harmfulMemories = append(harmfulMemories, types.AgentMemoryID(id))
	}

	reflection := &memory.Reflection{
		NewClaims:       resp.NewClaims,
		HelpfulMemories: helpfulMemories,
		HarmfulMemories: harmfulMemories,
	}

	logger.Debug("generated reflection",
		"new_claims_count", len(reflection.NewClaims),
		"helpful_count", len(reflection.HelpfulMemories),
		"harmful_count", len(reflection.HarmfulMemories))

	return reflection, nil
}
