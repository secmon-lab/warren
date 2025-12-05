package memory

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// SaveAgentMemory saves an agent memory record
func (r *Memory) SaveAgentMemory(ctx context.Context, mem *memory.AgentMemory) error {
	if err := mem.Validate(); err != nil {
		return r.eb.Wrap(err, "invalid agent memory")
	}

	r.memoryMu.Lock()
	defer r.memoryMu.Unlock()

	// Store a copy to prevent external modification
	memCopy := *mem
	r.agentMemories[mem.ID] = &memCopy

	return nil
}

// GetAgentMemory retrieves an agent memory record by agentID and memoryID
func (r *Memory) GetAgentMemory(ctx context.Context, agentID string, id types.AgentMemoryID) (*memory.AgentMemory, error) {
	r.memoryMu.RLock()
	defer r.memoryMu.RUnlock()

	mem, ok := r.agentMemories[id]
	if !ok {
		return nil, r.eb.Wrap(goerr.New("agent memory not found"), "not found",
			goerr.T(errs.TagNotFound),
			goerr.V("agent_id", agentID),
			goerr.V("id", id))
	}

	// Verify agent_id matches (for consistency with subcollection behavior)
	if mem.AgentID != agentID {
		return nil, r.eb.Wrap(goerr.New("agent memory not found"), "agent_id mismatch",
			goerr.T(errs.TagNotFound),
			goerr.V("agent_id", agentID),
			goerr.V("id", id))
	}

	// Return a copy to prevent external modification
	memCopy := *mem
	return &memCopy, nil
}

// SearchMemoriesByEmbedding searches for agent memories by embedding similarity
func (r *Memory) SearchMemoriesByEmbedding(ctx context.Context, agentID string, embedding []float32, limit int) ([]*memory.AgentMemory, error) {
	r.memoryMu.RLock()
	defer r.memoryMu.RUnlock()

	// Filter by agent_id and calculate similarity scores
	type scoredMemory struct {
		memory *memory.AgentMemory
		score  float64
	}

	var candidates []scoredMemory
	for _, mem := range r.agentMemories {
		if mem.AgentID != agentID {
			continue
		}

		if len(mem.QueryEmbedding) == 0 {
			continue
		}

		// Calculate cosine distance
		distance := r.cosineDistance(embedding, mem.QueryEmbedding)
		similarity := 1.0 - distance // Convert distance to similarity

		candidates = append(candidates, scoredMemory{
			memory: mem,
			score:  similarity,
		})
	}

	// Sort by similarity score (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Take top K
	resultSize := limit
	if len(candidates) < resultSize {
		resultSize = len(candidates)
	}

	results := make([]*memory.AgentMemory, resultSize)
	for i := 0; i < resultSize; i++ {
		// Return a copy to prevent external modification
		memCopy := *candidates[i].memory
		results[i] = &memCopy
	}

	return results, nil
}

// cosineDistance calculates cosine distance between two vectors
func (r *Memory) cosineDistance(v1, v2 []float32) float64 {
	if len(v1) != len(v2) {
		return 1.0 // Maximum distance for mismatched dimensions
	}

	var dotProduct, norm1, norm2 float64
	for i := range v1 {
		dotProduct += float64(v1[i]) * float64(v2[i])
		norm1 += float64(v1[i]) * float64(v1[i])
		norm2 += float64(v2[i]) * float64(v2[i])
	}

	if norm1 == 0 || norm2 == 0 {
		return 1.0 // Maximum distance if one vector is zero
	}

	cosineSimilarity := dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
	return 1.0 - cosineSimilarity
}

// UpdateMemoryScore updates the quality score and last used timestamp of an agent memory
func (r *Memory) UpdateMemoryScore(ctx context.Context, agentID string, memoryID types.AgentMemoryID, score float64, lastUsedAt time.Time) error {
	r.memoryMu.Lock()
	defer r.memoryMu.Unlock()

	mem, ok := r.agentMemories[memoryID]
	if !ok {
		return r.eb.Wrap(goerr.New("agent memory not found"), "not found",
			goerr.T(errs.TagNotFound),
			goerr.V("agent_id", agentID),
			goerr.V("memory_id", memoryID))
	}

	// Verify agent_id matches
	if mem.AgentID != agentID {
		return r.eb.Wrap(goerr.New("agent memory not found"), "agent_id mismatch",
			goerr.T(errs.TagNotFound),
			goerr.V("agent_id", agentID),
			goerr.V("memory_id", memoryID))
	}

	// Update the score and last used timestamp
	mem.QualityScore = score
	mem.LastUsedAt = lastUsedAt

	return nil
}

// UpdateMemoryScoreBatch updates quality scores and last used timestamps for multiple agent memories
func (r *Memory) UpdateMemoryScoreBatch(ctx context.Context, agentID string, updates map[types.AgentMemoryID]struct {
	Score      float64
	LastUsedAt time.Time
}) error {
	r.memoryMu.Lock()
	defer r.memoryMu.Unlock()

	for memoryID, update := range updates {
		mem, ok := r.agentMemories[memoryID]
		if !ok {
			continue // Skip non-existent memories
		}

		// Verify agent_id matches
		if mem.AgentID != agentID {
			continue // Skip memories that don't belong to this agent
		}

		// Update the score and last used timestamp
		mem.QualityScore = update.Score
		mem.LastUsedAt = update.LastUsedAt
	}

	return nil
}

// DeleteAgentMemoriesBatch deletes multiple agent memories in a batch
// Returns the number of successfully deleted memories
func (r *Memory) DeleteAgentMemoriesBatch(ctx context.Context, agentID string, memoryIDs []types.AgentMemoryID) (int, error) {
	r.memoryMu.Lock()
	defer r.memoryMu.Unlock()

	deletedCount := 0
	for _, id := range memoryIDs {
		mem, ok := r.agentMemories[id]
		if !ok {
			continue // Skip non-existent memories
		}

		// Verify agent_id matches
		if mem.AgentID != agentID {
			continue // Skip memories that don't belong to this agent
		}

		delete(r.agentMemories, id)
		deletedCount++
	}

	return deletedCount, nil
}

// ListAgentMemories lists all memories for an agent
// Results are ordered by Timestamp DESC
func (r *Memory) ListAgentMemories(ctx context.Context, agentID string) ([]*memory.AgentMemory, error) {
	r.memoryMu.RLock()
	defer r.memoryMu.RUnlock()

	var memories []*memory.AgentMemory
	for _, mem := range r.agentMemories {
		if mem.AgentID == agentID {
			// Create a copy to prevent external modification
			memCopy := *mem
			memories = append(memories, &memCopy)
		}
	}

	// Sort by Timestamp DESC
	sort.Slice(memories, func(i, j int) bool {
		return memories[i].Timestamp.After(memories[j].Timestamp)
	})

	return memories, nil
}
