package falcon

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

// extractRecords extracts the raw Falcon API records from the session history.
//
// This function does NOT use WithSessionResponseSchema because:
// 1. Falcon API responses have dynamic field structures
// 2. Different record types (incidents, alerts, behaviors) have different schemas
// 3. We use JSON mode WITHOUT schema and rely on the prompt
func (a *agent) extractRecords(ctx context.Context, originalRequest string, session gollem.Session) ([]map[string]any, error) {
	log := logging.From(ctx)
	log.Debug("Extracting records from session history", "original_request", originalRequest)

	// Create new session with JSON mode but NO schema
	extractSession, err := a.llmClient.NewSession(ctx,
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
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
		"Original user request: %s\n\n"+
			"Based on the conversation history above, extract the Falcon API records. "+
			"Return a JSON object with a 'records' field containing an array of records. "+
			"Each record MUST be a complete JSON object with ALL field names and values from the API responses. "+
			"Parse nested response structures carefully and convert each resource into a proper JSON object. "+
			"DO NOT return empty objects - each object must contain the actual data fields.",
		originalRequest,
	)

	log.Debug("Requesting record extraction", "request", extractionRequest)

	// Request record extraction
	resp, err := extractSession.GenerateContent(ctx, gollem.Text(extractionRequest))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate content for extraction")
	}

	if len(resp.Texts) == 0 {
		log.Warn("No text response from extraction")
		return []map[string]any{}, nil
	}

	// Parse JSON response - try multiple formats the LLM might return
	rawJSON := []byte(resp.Texts[0])

	// 1. Try wrapper object: {"records": [...]}
	var response struct {
		Records []map[string]any `json:"records"`
	}
	if err := json.Unmarshal(rawJSON, &response); err == nil && response.Records != nil {
		log.Debug("Successfully extracted records from wrapper object", "count", len(response.Records))
		return response.Records, nil
	}

	// 2. Try direct array: [...]
	var records []map[string]any
	if err := json.Unmarshal(rawJSON, &records); err == nil {
		log.Debug("Successfully extracted records from array", "count", len(records))
		return records, nil
	}

	// 3. Try single object: {...} (LLM sometimes returns a single record without wrapping in array)
	var single map[string]any
	if err := json.Unmarshal(rawJSON, &single); err == nil {
		// Check if it's a wrapper with "records" as a single object
		if rec, ok := single["records"]; ok {
			if recMap, ok := rec.(map[string]any); ok {
				log.Debug("Successfully extracted single record from wrapper object")
				return []map[string]any{recMap}, nil
			}
		}
		log.Debug("Successfully extracted single record object")
		return []map[string]any{single}, nil
	}

	return nil, goerr.New("failed to parse JSON response", goerr.V("response", resp.Texts[0]))
}
