package slack

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

//go:embed prompt/extract.md
var extractPrompt string

// JSON Schema for Slack messages array
var slackMessagesSchema = &gollem.Parameter{
	Type:        gollem.TypeArray,
	Description: "Array of Slack messages that match the user's request",
	Items: &gollem.Parameter{
		Type:        gollem.TypeObject,
		Description: "A single Slack message with its metadata",
		Properties: map[string]*gollem.Parameter{
			"text": {
				Type:        gollem.TypeString,
				Description: "Full message text content",
				Required:    true,
			},
			"user": {
				Type:        gollem.TypeString,
				Description: "User ID or display name who posted the message",
				Required:    true,
			},
			"channel": {
				Type:        gollem.TypeString,
				Description: "Channel ID or channel name where the message was posted",
				Required:    true,
			},
			"timestamp": {
				Type:        gollem.TypeString,
				Description: "Message timestamp (Slack ts format or ISO8601)",
				Required:    true,
			},
		},
	},
}

// extractRecords extracts the raw Slack messages from the session history
func (a *Agent) extractRecords(ctx context.Context, originalRequest string, session gollem.Session) ([]map[string]any, error) {
	log := logging.From(ctx)
	log.Debug("Extracting messages from session history", "original_request", originalRequest)

	// Create new session with JSON schema for messages array
	extractSession, err := a.llmClient.NewSession(ctx,
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
		gollem.WithSessionResponseSchema(slackMessagesSchema),
		gollem.WithSessionSystemPrompt(extractPrompt),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create extraction session")
	}

	// Add original session history
	history, err := session.History()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get session history")
	}
	if err := extractSession.AppendHistory(history); err != nil {
		return nil, goerr.Wrap(err, "failed to append session history")
	}

	// Build extraction request with original user request
	extractionRequest := fmt.Sprintf(
		"Original user request: %s\n\nBased on the conversation history above, "+
			"understand the user's intent and extract the messages that fulfill their actual needs.",
		originalRequest,
	)

	log.Debug("Requesting message extraction", "request", extractionRequest)

	// Request message extraction
	resp, err := extractSession.GenerateContent(ctx, gollem.Text(extractionRequest))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate content for extraction")
	}

	if len(resp.Texts) == 0 {
		log.Warn("No text response from extraction")
		return []map[string]any{}, nil
	}

	// Parse JSON response
	var messages []map[string]any
	if err := json.Unmarshal([]byte(resp.Texts[0]), &messages); err != nil {
		return nil, goerr.Wrap(err, "failed to parse JSON response", goerr.V("response", resp.Texts[0]))
	}

	log.Debug("Successfully extracted messages", "count", len(messages))
	return messages, nil
}
