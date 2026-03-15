package legacy

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

//go:embed prompt/chat_system_prompt.md
var chatSystemPromptTemplate string

//go:embed prompt/ticketless_system_prompt.md
var ticketlessSystemPromptTemplate string

//go:embed prompt/tool_call_to_text.md
var toolCallToTextPromptTemplate string

// GenerateChatSystemPrompt generates the chat system prompt from template and parameters.
func GenerateChatSystemPrompt(ctx context.Context, target *ticket.Ticket, alertCount int, additionalInstructions string, knowledges []*knowledge.Knowledge, requesterID string, threadComments []ticket.Comment, userSystemPrompt string, historyMessages []model.HistoryMessage) (string, error) {
	ticketJSON, err := json.MarshalIndent(target, "", "  ")
	if err != nil {
		return "", goerr.Wrap(err, "failed to marshal ticket to JSON")
	}

	return prompt.GenerateWithStruct(ctx, chatSystemPromptTemplate, map[string]any{
		"ticket_json":             "```json\n" + string(ticketJSON) + "\n```",
		"total":                   alertCount,
		"additional_instructions": additionalInstructions,
		"knowledges":              knowledges,
		"topic":                   target.Topic,
		"lang":                    lang.From(ctx),
		"requester_id":            requesterID,
		"thread_comments":         threadComments,
		"user_system_prompt":      userSystemPrompt,
		"history_messages":        historyMessages,
	})
}

// GenerateTicketlessSystemPrompt generates the system prompt for ticketless chat.
func GenerateTicketlessSystemPrompt(ctx context.Context, historyMessages []model.HistoryMessage, additionalInstructions string, knowledges []*knowledge.Knowledge, requesterID string, userSystemPrompt string) (string, error) {
	return prompt.GenerateWithStruct(ctx, ticketlessSystemPromptTemplate, map[string]any{
		"history_messages":        historyMessages,
		"additional_instructions": additionalInstructions,
		"knowledges":              knowledges,
		"lang":                    lang.From(ctx),
		"requester_id":            requesterID,
		"user_system_prompt":      userSystemPrompt,
	})
}

// ToolCallToText converts a tool call to a human-readable text description using LLM.
func ToolCallToText(ctx context.Context, llmClient gollem.LLMClient, spec *gollem.ToolSpec, call *gollem.FunctionCall) string {
	eb := goerr.NewBuilder(
		goerr.V("tool", call.Name),
		goerr.V("spec", spec),
	)
	defaultMsg := fmt.Sprintf("⚡ Execute Tool: `%s`", call.Name)
	if spec == nil {
		errutil.Handle(ctx, eb.New("tool not found"))
		return defaultMsg
	}

	p, err := prompt.Generate(ctx, toolCallToTextPromptTemplate, map[string]any{
		"spec":      spec,
		"tool_call": call,
		"lang":      lang.From(ctx),
	})
	if err != nil {
		errutil.Handle(ctx, eb.Wrap(err, "failed to generate prompt"))
		return defaultMsg
	}

	session, err := llmClient.NewSession(ctx)
	if err != nil {
		errutil.Handle(ctx, eb.Wrap(err, "failed to create session"))
		return defaultMsg
	}

	response, err := session.GenerateContent(ctx, gollem.Text(p))
	if err != nil {
		errutil.Handle(ctx, eb.Wrap(err, "failed to generate content"))
		return defaultMsg
	}

	if len(response.Texts) == 0 {
		errutil.Handle(ctx, eb.New("no response"))
		return defaultMsg
	}

	return response.Texts[0]
}
