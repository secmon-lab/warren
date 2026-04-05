package bluebell

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/cli/config"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/utils/logging"

	_ "embed"
)

//go:embed prompt/selector.md
var selectorPromptTemplate string

// ResolvedIntent is the output of the intent resolver.
type ResolvedIntent struct {
	PromptID   string `json:"prompt_id"` // selected prompt id (or "default")
	PromptName string `json:"-"`         // human-readable name (set from PromptEntry, not from LLM)
	Intent     string `json:"intent"`    // situation-specific investigation directive
}

// selectorSchema defines the JSON response schema for the selector LLM call.
var selectorSchema = &gollem.Parameter{
	Type: gollem.TypeObject,
	Properties: map[string]*gollem.Parameter{
		"prompt_id": {
			Type:        gollem.TypeString,
			Description: "The selected prompt's id, or 'default' if none fit",
			Required:    true,
		},
		"intent": {
			Type:        gollem.TypeString,
			Description: "Situation-specific investigation directive (2-5 sentences)",
			Required:    true,
		},
	},
}

// selectorTemplateData holds data passed to the selector.md template.
type selectorTemplateData struct {
	Message string
	Context ContextData
	Prompts []promptSummary
}

// promptSummary holds only id + description for the selector (Content is NOT passed).
type promptSummary struct {
	ID          string
	Description string
}

// resolveIntent performs prompt selection and intent resolution.
// Returns nil if no prompt entries are configured (default behavior).
func (c *BluebellChat) resolveIntent(ctx context.Context, message string, chatCtx *chatModel.ChatContext) (*ResolvedIntent, error) {
	if len(c.promptEntries) == 0 {
		return nil, nil
	}

	// Build selector template data — only id + description, NOT Content
	summaries := make([]promptSummary, len(c.promptEntries))
	for i, p := range c.promptEntries {
		summaries[i] = promptSummary{
			ID:          p.ID,
			Description: p.Description,
		}
	}

	// Build context data for the selector template
	ctxData := buildSelectorContext(chatCtx)

	tmplData := selectorTemplateData{
		Message: message,
		Context: ctxData,
		Prompts: summaries,
	}

	selectorPrompt, err := prompt.GenerateWithStruct(ctx, selectorPromptTemplate, tmplData)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate selector prompt")
	}

	// For single candidate, skip selection but still resolve intent
	if len(c.promptEntries) == 1 {
		return c.resolveIntentForSinglePrompt(ctx, selectorPrompt, c.promptEntries[0])
	}

	// Multiple candidates: selection + intent resolution in one LLM call
	return c.resolveIntentWithSelection(ctx, selectorPrompt, c.promptEntries)
}

// resolveIntentForSinglePrompt resolves intent for a single prompt candidate (no selection needed).
func (c *BluebellChat) resolveIntentForSinglePrompt(ctx context.Context, selectorPrompt string, entry config.PromptEntry) (*ResolvedIntent, error) {
	logger := logging.From(ctx)

	session, err := c.llmClient.NewSession(ctx,
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
		gollem.WithSessionResponseSchema(selectorSchema),
		gollem.WithSessionSystemPrompt(selectorPrompt),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create selector session")
	}

	resp, err := session.Generate(ctx, []gollem.Input{
		gollem.Text("Select the best prompt and resolve the investigation intent."),
	})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate intent resolution")
	}

	resolved, err := parseSelectorResult(resp.Texts)
	if err != nil {
		logger.Warn("failed to parse selector result for single prompt, using fallback",
			"error", err, "prompt_id", entry.ID)
		return &ResolvedIntent{PromptID: entry.ID, PromptName: entry.Name, Intent: entry.Description}, nil
	}

	// Ensure the prompt_id and name match the single candidate
	resolved.PromptID = entry.ID
	resolved.PromptName = entry.Name
	return resolved, nil
}

// resolveIntentWithSelection performs selection + intent resolution for multiple candidates.
func (c *BluebellChat) resolveIntentWithSelection(ctx context.Context, selectorPrompt string, entries []config.PromptEntry) (*ResolvedIntent, error) {
	logger := logging.From(ctx)

	session, err := c.llmClient.NewSession(ctx,
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
		gollem.WithSessionResponseSchema(selectorSchema),
		gollem.WithSessionSystemPrompt(selectorPrompt),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create selector session")
	}

	resp, err := session.Generate(ctx, []gollem.Input{
		gollem.Text("Select the best prompt and resolve the investigation intent."),
	})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate intent resolution")
	}

	resolved, err := parseSelectorResult(resp.Texts)
	if err != nil {
		logger.Warn("failed to parse selector result, falling back to default", "error", err)
		return nil, nil
	}

	// Validate the selected prompt_id exists and set name
	if resolved.PromptID != "default" {
		found := false
		for _, e := range entries {
			if e.ID == resolved.PromptID {
				resolved.PromptName = e.Name
				found = true
				break
			}
		}
		if !found {
			logger.Warn("selector returned unknown prompt_id, falling back to default",
				"prompt_id", resolved.PromptID)
			return nil, nil
		}
	}

	return resolved, nil
}

// buildSelectorContext creates context data for the selector template.
// Uses ThreadComments and SlackHistory but NOT chatCtx.History (LLM session history).
func buildSelectorContext(chatCtx *chatModel.ChatContext) ContextData {
	var ctxData ContextData

	if chatCtx.Ticket != nil && chatCtx.Ticket.ID != "" {
		ticketJSON, _ := json.MarshalIndent(chatCtx.Ticket, "", "  ")
		ctxData.Ticket = string(ticketJSON)

		if len(chatCtx.Alerts) > 0 {
			alertJSON, _ := json.MarshalIndent(chatCtx.Alerts[0], "", "  ")
			ctxData.Alert = AlertData{
				Data:  string(alertJSON),
				Count: len(chatCtx.Alerts),
			}
		}

		ctxData.Thread = ThreadData{
			Comments: chatCtx.ThreadComments,
		}
	}

	if len(chatCtx.SlackHistory) > 0 {
		ctxData.Channel = ChannelData{
			History: chatCtx.SlackHistory,
		}
	}

	return ctxData
}

// parseSelectorResult parses the LLM response into a ResolvedIntent.
func parseSelectorResult(texts []string) (*ResolvedIntent, error) {
	if len(texts) == 0 {
		return nil, goerr.New("no response from selector LLM")
	}
	var result ResolvedIntent
	if err := json.Unmarshal([]byte(texts[0]), &result); err != nil {
		return nil, goerr.Wrap(err, "failed to parse selector result", goerr.V("raw", texts[0]))
	}
	return &result, nil
}
