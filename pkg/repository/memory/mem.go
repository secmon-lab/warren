package memory

import (
	"context"
	"math"
	"sort"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// GetExecutionMemory retrieves the latest execution memory for the specified schema
func (r *Memory) GetExecutionMemory(ctx context.Context, schemaID types.AlertSchema) (*memory.ExecutionMemory, error) {
	r.memoryMu.RLock()
	defer r.memoryMu.RUnlock()

	mems, exists := r.executionMemories[schemaID]
	if !exists || len(mems) == 0 {
		return nil, nil
	}

	// Return the latest memory (last in slice)
	latest := mems[len(mems)-1]
	memCopy := *latest
	return &memCopy, nil
}

// PutExecutionMemory stores execution memory as history for the specified schema
func (r *Memory) PutExecutionMemory(ctx context.Context, mem *memory.ExecutionMemory) error {
	if err := mem.Validate(); err != nil {
		return r.eb.Wrap(err, "invalid execution memory")
	}

	r.memoryMu.Lock()
	defer r.memoryMu.Unlock()

	// Store a copy to prevent external modification
	memCopy := *mem

	// Append to history
	r.executionMemories[mem.SchemaID] = append(r.executionMemories[mem.SchemaID], &memCopy)

	return nil
}

// SearchExecutionMemoriesByEmbedding searches execution memories by embedding similarity
func (r *Memory) SearchExecutionMemoriesByEmbedding(ctx context.Context, schemaID types.AlertSchema, embedding []float32, limit int) ([]*memory.ExecutionMemory, error) {
	r.memoryMu.RLock()
	defer r.memoryMu.RUnlock()

	// Early return if embedding is empty
	if len(embedding) == 0 {
		return nil, nil
	}

	// Get all memories for the schema
	mems, exists := r.executionMemories[schemaID]
	if !exists || len(mems) == 0 {
		return nil, nil
	}

	// Filter memories that have embeddings
	var memsWithEmbedding []*memory.ExecutionMemory
	for _, m := range mems {
		if len(m.Embedding) > 0 {
			memsWithEmbedding = append(memsWithEmbedding, m)
		}
	}

	if len(memsWithEmbedding) == 0 {
		return nil, nil
	}

	// Sort by similarity (descending)
	sort.Slice(memsWithEmbedding, func(i, j int) bool {
		simI := cosineSimilarity(memsWithEmbedding[i].Embedding, embedding)
		simJ := cosineSimilarity(memsWithEmbedding[j].Embedding, embedding)
		return simI > simJ
	})

	// Return top limit results
	if limit > 0 && limit < len(memsWithEmbedding) {
		memsWithEmbedding = memsWithEmbedding[:limit]
	}

	// Create copies to prevent external modification
	result := make([]*memory.ExecutionMemory, len(memsWithEmbedding))
	for i, m := range memsWithEmbedding {
		memCopy := *m
		result[i] = &memCopy
	}

	return result, nil
}

// GetTicketMemory retrieves the latest ticket memory for the specified schema
func (r *Memory) GetTicketMemory(ctx context.Context, schemaID types.AlertSchema) (*memory.TicketMemory, error) {
	r.memoryMu.RLock()
	defer r.memoryMu.RUnlock()

	mems, exists := r.ticketMemories[schemaID]
	if !exists || len(mems) == 0 {
		return nil, nil
	}

	// Return the latest memory (last in slice)
	latest := mems[len(mems)-1]
	memCopy := *latest
	return &memCopy, nil
}

// PutTicketMemory stores ticket memory as history for the specified schema
func (r *Memory) PutTicketMemory(ctx context.Context, mem *memory.TicketMemory) error {
	if err := mem.Validate(); err != nil {
		return r.eb.Wrap(err, "invalid ticket memory")
	}

	r.memoryMu.Lock()
	defer r.memoryMu.Unlock()

	// Store a copy to prevent external modification
	memCopy := *mem

	// Append to history
	r.ticketMemories[mem.SchemaID] = append(r.ticketMemories[mem.SchemaID], &memCopy)

	return nil
}

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
