package memory

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

// searchAndRerankMemories performs vector search + re-ranking with quality/recency scoring
// This is the core of the memory scoring system
func (s *Service) searchAndRerankMemories(
	ctx context.Context,
	agentID string,
	queryEmbedding []float32,
	limit int,
) ([]*memory.AgentMemory, error) {
	// Step 1: Calculate search limit (limit * multiplier, capped at max)
	searchLimit := limit * s.ScoringConfig.SearchMultiplier
	if searchLimit > s.ScoringConfig.SearchMaxCandidates {
		searchLimit = s.ScoringConfig.SearchMaxCandidates
	}

	// Step 2: Perform vector search to get candidates
	candidates, err := s.repository.SearchMemoriesByEmbedding(ctx, agentID, queryEmbedding, searchLimit)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to search memories by embedding")
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// Step 3: Calculate similarity scores and filter by quality
	type rankedMemory struct {
		memory     *memory.AgentMemory
		similarity float64
		finalScore float64
	}

	now := time.Now()
	var ranked []rankedMemory

	for _, mem := range candidates {
		// Filter out memories with quality score below threshold
		if mem.QualityScore < s.ScoringConfig.FilterMinQuality {
			continue
		}

		// Calculate similarity
		similarity := calculateCosineSimilarity(queryEmbedding, mem.QueryEmbedding)

		// Calculate recency score
		recency := s.calculateRecencyScore(mem.LastUsedAt, now)

		// Normalize quality score from [-10, 10] to [0, 1]
		qualityNormalized := (mem.QualityScore - s.ScoringConfig.ScoreMin) /
			(s.ScoringConfig.ScoreMax - s.ScoringConfig.ScoreMin)

		// Calculate final weighted score
		finalScore := s.ScoringConfig.RankSimilarityWeight*similarity +
			s.ScoringConfig.RankQualityWeight*qualityNormalized +
			s.ScoringConfig.RankRecencyWeight*recency

		ranked = append(ranked, rankedMemory{
			memory:     mem,
			similarity: similarity,
			finalScore: finalScore,
		})
	}

	// Step 4: Sort by final score descending
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].finalScore > ranked[j].finalScore
	})

	// Step 5: Take top N results
	resultSize := limit
	if len(ranked) < resultSize {
		resultSize = len(ranked)
	}

	results := make([]*memory.AgentMemory, resultSize)
	memoryIDs := make([]types.AgentMemoryID, resultSize)
	for i := 0; i < resultSize; i++ {
		results[i] = ranked[i].memory
		memoryIDs[i] = ranked[i].memory.ID
	}

	// Step 6: Update LastUsedAt for returned memories (non-blocking batch update)
	go func() {
		updates := make(map[types.AgentMemoryID]struct {
			Score      float64
			LastUsedAt time.Time
		})
		for i, id := range memoryIDs {
			updates[id] = struct {
				Score      float64
				LastUsedAt time.Time
			}{
				Score:      results[i].QualityScore,
				LastUsedAt: now,
			}
		}

		if err := s.repository.UpdateMemoryScoreBatch(context.Background(), agentID, updates); err != nil {
			logging.Default().Warn("failed to batch update last used at", "agent_id", agentID, "error", err)
		}
	}()

	return results, nil
}

// PruneAgentMemories deletes low-quality memories for an agent based on strict criteria
func (s *Service) PruneAgentMemories(ctx context.Context, agentID string) (int, error) {
	// Get all memories for the agent
	allMemories, err := s.repository.ListAgentMemories(ctx, agentID)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to list agent memories", goerr.V("agent_id", agentID))
	}

	if len(allMemories) == 0 {
		return 0, nil
	}

	// Filter memories that should be deleted based on strict criteria
	var toDelete []types.AgentMemoryID
	now := time.Now()

	for _, mem := range allMemories {
		shouldDelete := false

		// Criterion 1: Critically low score (immediate deletion)
		if mem.QualityScore < s.ScoringConfig.PruneCriticalScore {
			shouldDelete = true
		}

		// Criterion 2: Harmful score AND not used for a long time
		if !shouldDelete && mem.QualityScore < s.ScoringConfig.PruneHarmfulScore {
			daysSinceUsed := 0
			if !mem.LastUsedAt.IsZero() {
				daysSinceUsed = int(now.Sub(mem.LastUsedAt).Hours() / 24)
			} else {
				// If never used, use timestamp
				daysSinceUsed = int(now.Sub(mem.Timestamp).Hours() / 24)
			}
			if daysSinceUsed >= s.ScoringConfig.PruneHarmfulDays {
				shouldDelete = true
			}
		}

		// Criterion 3: Moderately harmful score AND very long unused
		if !shouldDelete && mem.QualityScore < s.ScoringConfig.PruneModerateScore {
			daysSinceUsed := 0
			if !mem.LastUsedAt.IsZero() {
				daysSinceUsed = int(now.Sub(mem.LastUsedAt).Hours() / 24)
			} else {
				daysSinceUsed = int(now.Sub(mem.Timestamp).Hours() / 24)
			}
			if daysSinceUsed >= s.ScoringConfig.PruneModerateDays {
				shouldDelete = true
			}
		}

		if shouldDelete {
			toDelete = append(toDelete, mem.ID)
		}
	}

	if len(toDelete) == 0 {
		return 0, nil
	}

	// Delete in batch
	deleted, err := s.repository.DeleteAgentMemoriesBatch(ctx, agentID, toDelete)
	if err != nil {
		return deleted, goerr.Wrap(err, "failed to delete memories batch",
			goerr.V("agent_id", agentID),
			goerr.V("to_delete_count", len(toDelete)))
	}

	logging.From(ctx).Info("pruned agent memories",
		"agent_id", agentID,
		"deleted_count", deleted,
		"total_memories", len(allMemories))

	return deleted, nil
}

// calculateCosineSimilarity computes cosine similarity between two vectors
// Returns a value between 0 (orthogonal) and 1 (identical)
func calculateCosineSimilarity(v1, v2 []float32) float64 {
	if len(v1) != len(v2) || len(v1) == 0 {
		return 0.0
	}

	var dotProduct, norm1, norm2 float64
	for i := range v1 {
		dotProduct += float64(v1[i]) * float64(v2[i])
		norm1 += float64(v1[i]) * float64(v1[i])
		norm2 += float64(v2[i]) * float64(v2[i])
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// calculateRecencyScore computes recency score using exponential decay
// Uses half-life from ScoringConfig for decay calculation
// Returns a value between 0 (very old) and 1 (very recent)
func (s *Service) calculateRecencyScore(lastUsed time.Time, now time.Time) float64 {
	// If never used, return 0 (oldest possible)
	if lastUsed.IsZero() {
		return 0.0
	}

	// Calculate days since last use
	daysSince := now.Sub(lastUsed).Hours() / 24.0

	// Exponential decay with half-life
	// score = 0.5^(days / half_life)
	decayScore := math.Pow(0.5, daysSince/s.ScoringConfig.RecencyHalfLifeDays)

	return decayScore
}
