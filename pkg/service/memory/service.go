package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

const (
	// EmbeddingDimension is the dimension of embedding vectors
	EmbeddingDimension = 256
)

// Service provides agent memory management with scoring, selection, and pruning
// Each service instance is bound to a specific agent
type Service struct {
	agentID    string
	repository interfaces.Repository
	llmClient  gollem.LLMClient

	// Algorithm functions (replaceable)
	scoringAlgo   ScoringAlgorithm
	selectionAlgo SelectionAlgorithm
	pruningAlgo   PruningAlgorithm
}

// New creates a new memory service bound to a specific agent with default algorithms
func New(agentID string, llmClient gollem.LLMClient, repo interfaces.Repository) *Service {
	return &Service{
		agentID:       agentID,
		repository:    repo,
		llmClient:     llmClient,
		scoringAlgo:   DefaultScoringAlgorithm,
		selectionAlgo: DefaultSelectionAlgorithm,
		pruningAlgo:   DefaultPruningAlgorithm,
	}
}

// WithScoringAlgorithm replaces the scoring algorithm
func (s *Service) WithScoringAlgorithm(algo ScoringAlgorithm) *Service {
	s.scoringAlgo = algo
	return s
}

// WithSelectionAlgorithm replaces the selection algorithm
func (s *Service) WithSelectionAlgorithm(algo SelectionAlgorithm) *Service {
	s.selectionAlgo = algo
	return s
}

// WithPruningAlgorithm replaces the pruning algorithm
func (s *Service) WithPruningAlgorithm(algo PruningAlgorithm) *Service {
	s.pruningAlgo = algo
	return s
}

// SearchAndSelectMemories searches for relevant memories and selects top N using injection algorithm
// This is the main method agents should call before execution to get relevant memories
func (s *Service) SearchAndSelectMemories(
	ctx context.Context,
	query string,
	limit int,
) ([]*memory.AgentMemory, error) {
	// Generate embedding for the query
	embeddings, err := s.llmClient.GenerateEmbedding(ctx, EmbeddingDimension, []string{query})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate embedding for search", goerr.V("query", query))
	}

	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return nil, goerr.New("no embedding generated")
	}

	// Convert float64 to float32
	queryEmbedding := make([]float32, len(embeddings[0]))
	for i, v := range embeddings[0] {
		queryEmbedding[i] = float32(v)
	}

	// Calculate search limit using multiplier from injection algorithm
	const searchMultiplier = 5
	const searchMaxCandidates = 100
	searchLimit := limit * searchMultiplier
	if searchLimit > searchMaxCandidates {
		searchLimit = searchMaxCandidates
	}

	// Perform vector search to get candidates
	candidates, err := s.repository.SearchMemoriesByEmbedding(ctx, s.agentID, queryEmbedding, searchLimit)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to search memories by embedding", goerr.V("agent_id", s.agentID))
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// Use selection algorithm to rank and filter
	selected := s.selectionAlgo(candidates, queryEmbedding, limit)

	// Update LastUsedAt for selected memories (non-blocking)
	if len(selected) > 0 {
		go func() {
			now := time.Now()
			updates := make(map[types.AgentMemoryID]struct {
				Score      float64
				LastUsedAt time.Time
			})
			for _, mem := range selected {
				updates[mem.ID] = struct {
					Score      float64
					LastUsedAt time.Time
				}{
					Score:      mem.Score,
					LastUsedAt: now,
				}
			}

			if err := s.repository.UpdateMemoryScoreBatch(context.Background(), s.agentID, updates); err != nil {
				logging.Default().Warn("failed to batch update last used at", "agent_id", s.agentID, "error", err)
			}
		}()
	}

	return selected, nil
}

// ExtractAndSaveMemories extracts new claims from execution history and saves them
// This method uses LLM to generate reflection and extract claims from execution history
// Responsibilities: Extract claims, update scores, generate embeddings, save memories
// Does NOT perform pruning - caller should call PruneMemories separately when needed
func (s *Service) ExtractAndSaveMemories(
	ctx context.Context,
	query string,
	usedMemories []*memory.AgentMemory,
	history *gollem.History,
) error {
	// Step 1: Generate reflection using LLM
	reflection, err := s.generateReflection(ctx, query, usedMemories, history)
	if err != nil {
		return goerr.Wrap(err, "failed to generate reflection",
			goerr.V("agent_id", s.agentID),
			goerr.V("query", query))
	}

	// Step 2: Update scores for helpful/harmful memories and collect score changes
	var scoreChanges []string
	if len(reflection.HelpfulMemories) > 0 || len(reflection.HarmfulMemories) > 0 {
		// Build memory map for scoring algorithm
		memoryMap := make(map[types.AgentMemoryID]*memory.AgentMemory)
		for _, mem := range usedMemories {
			memoryMap[mem.ID] = mem
		}

		// Apply scoring algorithm
		scoreUpdates := s.scoringAlgo(memoryMap, reflection)

		if len(scoreUpdates) > 0 {
			now := time.Now()
			updates := make(map[types.AgentMemoryID]struct {
				Score      float64
				LastUsedAt time.Time
			})
			for memID, newScore := range scoreUpdates {
				oldScore := memoryMap[memID].Score
				updates[memID] = struct {
					Score      float64
					LastUsedAt time.Time
				}{
					Score:      newScore,
					LastUsedAt: now,
				}
				// Record score change for logging
				scoreChanges = append(scoreChanges,
					fmt.Sprintf("%s: %.2f → %.2f (Δ%.2f)",
						memID, oldScore, newScore, newScore-oldScore))
			}

			if err := s.repository.UpdateMemoryScoreBatch(ctx, s.agentID, updates); err != nil {
				return goerr.Wrap(err, "failed to update memory scores batch", goerr.V("agent_id", s.agentID))
			}
		}
	}

	// Log reflection summary with new claims and score changes
	logger := logging.From(ctx)
	logArgs := []any{
		"agent_id", s.agentID,
		"new_claims_count", len(reflection.NewClaims),
		"helpful_memories_count", len(reflection.HelpfulMemories),
		"harmful_memories_count", len(reflection.HarmfulMemories),
	}

	// Add new claims to log
	if len(reflection.NewClaims) > 0 {
		logArgs = append(logArgs, "new_claims", reflection.NewClaims)
	}

	// Add score changes to log
	if len(scoreChanges) > 0 {
		logArgs = append(logArgs, "score_changes", scoreChanges)
	}

	logger.Info("reflection completed", logArgs...)

	// Step 3: Save new claims as memories
	if len(reflection.NewClaims) > 0 {
		// Generate embeddings for all new claims
		embeddings, err := s.llmClient.GenerateEmbedding(ctx, EmbeddingDimension, []string{query})
		if err != nil {
			return goerr.Wrap(err, "failed to generate embeddings for new claims", goerr.V("agent_id", s.agentID))
		}

		if len(embeddings) == 0 || len(embeddings[0]) == 0 {
			return goerr.New("no embedding generated for query")
		}

		// Convert embedding to float32
		queryEmbedding := make([]float32, len(embeddings[0]))
		for i, v := range embeddings[0] {
			queryEmbedding[i] = float32(v)
		}

		// Create new memory records for each claim
		now := time.Now()
		newMemories := make([]*memory.AgentMemory, len(reflection.NewClaims))
		for i, claim := range reflection.NewClaims {
			newMemories[i] = &memory.AgentMemory{
				ID:             types.NewAgentMemoryID(),
				AgentID:        s.agentID,
				Query:          query,
				QueryEmbedding: queryEmbedding,
				Claim:          claim,
				Score:          0.0, // neutral initial score
				CreatedAt:      now,
				LastUsedAt:     time.Time{}, // zero value indicates never used
			}
		}

		// Batch save new memories
		if err := s.repository.BatchSaveAgentMemories(ctx, newMemories); err != nil {
			return goerr.Wrap(err, "failed to batch save new memories",
				goerr.V("agent_id", s.agentID),
				goerr.V("count", len(newMemories)))
		}
	}

	return nil
}

// PruneMemories removes low-quality memories based on score and usage patterns
// This is a separate operation from ExtractAndSaveMemories for clear separation of concerns
// Caller controls when to prune (e.g., periodically, after N executions, manually)
func (s *Service) PruneMemories(ctx context.Context) (int, error) {
	// Get all memories for the agent
	allMemories, err := s.repository.ListAgentMemories(ctx, s.agentID)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to list agent memories", goerr.V("agent_id", s.agentID))
	}

	if len(allMemories) == 0 {
		return 0, nil
	}

	// Apply pruning algorithm
	now := time.Now()
	toDelete := s.pruningAlgo(allMemories, now)

	if len(toDelete) == 0 {
		return 0, nil
	}

	// Delete in batch
	deleted, err := s.repository.DeleteAgentMemoriesBatch(ctx, s.agentID, toDelete)
	if err != nil {
		return deleted, goerr.Wrap(err, "failed to delete memories batch",
			goerr.V("agent_id", s.agentID),
			goerr.V("to_delete_count", len(toDelete)))
	}

	logging.From(ctx).Info("pruned agent memories",
		"agent_id", s.agentID,
		"deleted_count", deleted,
		"total_memories", len(allMemories))

	return deleted, nil
}
