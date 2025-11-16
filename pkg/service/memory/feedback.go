package memory

import (
	"context"
	_ "embed"
	"encoding/json"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

//go:embed prompt/feedback.md
var feedbackPromptTemplate string

// feedbackResponse defines the structure for feedback LLM response
type feedbackResponse struct {
	Relevance int    `json:"relevance" description:"Relevance score (0-3)"`
	Support   int    `json:"support" description:"Support score (0-4)"`
	Impact    int    `json:"impact" description:"Impact score (0-3)"`
	Reasoning string `json:"reasoning" description:"Explanation for the scores"`
}

// generateMemoryFeedback generates feedback for a memory using LLM
func (s *Service) generateMemoryFeedback(
	ctx context.Context,
	mem *memory.AgentMemory,
	taskQuery string,
	session gollem.Session,
	execResult *gollem.ExecuteResponse,
	execError error,
) (*memory.MemoryFeedback, error) {
	// Generate JSON schema using prompt.ToSchema (like other methods in this service)
	schema := prompt.ToSchema(feedbackResponse{})
	jsonSchemaStr, err := schema.Stringify()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to stringify feedback schema")
	}

	// Build prompt - pass memory fields explicitly to avoid template interface{} issues
	// Ensure slices are non-nil for proper template iteration
	successes := mem.Successes
	if successes == nil {
		successes = []string{}
	}
	problems := mem.Problems
	if problems == nil {
		problems = []string{}
	}
	improvements := mem.Improvements
	if improvements == nil {
		improvements = []string{}
	}

	promptParams := map[string]any{
		"TaskQuery":  taskQuery,
		"ExecResult": execResult,
		"ExecError":  execError,
		"JSONSchema": jsonSchemaStr,
		// Pass memory fields explicitly
		"MemoryTaskQuery":    mem.TaskQuery,
		"MemorySuccesses":    successes,
		"MemoryProblems":     problems,
		"MemoryImprovements": improvements,
	}

	promptText, err := prompt.Generate(ctx, feedbackPromptTemplate, promptParams)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate feedback prompt")
	}

	// Convert to gollem schema for session creation
	gollemSchema, err := gollem.ToSchema(feedbackResponse{})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate gollem schema")
	}

	// Create a new session for feedback generation
	feedbackSession, err := s.llmClient.NewSession(ctx,
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
		gollem.WithSessionResponseSchema(gollemSchema),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create feedback session")
	}

	// Generate feedback
	response, err := feedbackSession.GenerateContent(ctx, gollem.Text(promptText))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate feedback content")
	}

	// Parse response
	if len(response.Texts) == 0 {
		return nil, goerr.New("no response text from LLM")
	}

	text := extractJSONFromMarkdown(response.Texts[0])

	var resp feedbackResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal feedback response", goerr.V("text", text))
	}

	// Validate and clamp scores to expected ranges
	resp.Relevance = clampScore(ctx, resp.Relevance, 0, 3, "relevance")
	resp.Support = clampScore(ctx, resp.Support, 0, 4, "support")
	resp.Impact = clampScore(ctx, resp.Impact, 0, 3, "impact")

	feedback := &memory.MemoryFeedback{
		MemoryID:  mem.ID,
		Relevance: resp.Relevance,
		Support:   resp.Support,
		Impact:    resp.Impact,
		Reasoning: resp.Reasoning,
	}

	return feedback, nil
}

// CollectAndApplyFeedback collects feedback from execution and updates scores
// This is the main entry point for the feedback system
func (s *Service) CollectAndApplyFeedback(
	ctx context.Context,
	agentID string,
	usedMemories []*memory.AgentMemory,
	taskQuery string,
	session gollem.Session,
	execResult *gollem.ExecuteResponse,
	execError error,
) error {
	if len(usedMemories) == 0 {
		return nil
	}

	logger := logging.From(ctx)
	now := time.Now()

	// Collect all score updates
	type scoreUpdate struct {
		Score      float64
		LastUsedAt time.Time
	}
	updates := make(map[types.AgentMemoryID]scoreUpdate)

	// Process each memory to generate feedback and calculate new scores
	for _, mem := range usedMemories {
		// Generate feedback for this memory
		feedback, err := s.generateMemoryFeedback(ctx, mem, taskQuery, session, execResult, execError)
		if err != nil {
			// Non-critical error: log and continue
			logger.Warn("failed to generate feedback for memory",
				"memory_id", mem.ID,
				"error", err)
			continue
		}

		// Calculate normalized score (-10 to +10)
		normalizedScore := feedback.NormalizedScore()

		// Calculate new score using EMA: new = alpha * feedback + (1-alpha) * old
		newScore := s.ScoringConfig.EMAAlpha*normalizedScore + (1-s.ScoringConfig.EMAAlpha)*mem.QualityScore

		// Clip to configured range
		if newScore < s.ScoringConfig.ScoreMin {
			newScore = s.ScoringConfig.ScoreMin
		}
		if newScore > s.ScoringConfig.ScoreMax {
			newScore = s.ScoringConfig.ScoreMax
		}

		// Add to batch update
		updates[mem.ID] = scoreUpdate{
			Score:      newScore,
			LastUsedAt: now,
		}

		logger.Debug("calculated feedback score for memory",
			"memory_id", mem.ID,
			"old_score", mem.QualityScore,
			"new_score", newScore,
			"feedback_score", normalizedScore,
			"relevance", feedback.Relevance,
			"support", feedback.Support,
			"impact", feedback.Impact,
			"reasoning", truncateString(feedback.Reasoning, 100))
	}

	// Batch update all scores
	if len(updates) > 0 {
		// Convert to repository format
		repoUpdates := make(map[types.AgentMemoryID]struct {
			Score      float64
			LastUsedAt time.Time
		})
		for id, update := range updates {
			repoUpdates[id] = struct {
				Score      float64
				LastUsedAt time.Time
			}{
				Score:      update.Score,
				LastUsedAt: update.LastUsedAt,
			}
		}

		if err := s.repository.UpdateMemoryScoreBatch(ctx, agentID, repoUpdates); err != nil {
			return goerr.Wrap(err, "failed to batch update memory scores",
				goerr.V("agent_id", agentID),
				goerr.V("count", len(updates)))
		}

		logger.Debug("batch updated memory scores",
			"agent_id", agentID,
			"count", len(updates))
	}

	return nil
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// clampScore clamps a score to the specified range and logs a warning if out of range
func clampScore(ctx context.Context, score, min, max int, name string) int {
	if score < min || score > max {
		logging.From(ctx).Warn(name+" score out of range, clamping", "score", score)
		if score < min {
			return min
		}
		if score > max {
			return max
		}
	}
	return score
}
