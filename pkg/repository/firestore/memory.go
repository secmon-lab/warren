package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SaveAgentMemory saves an agent memory record in subcollection structure
// Path: agents/{agentID}/memories/{memoryID}
func (r *Firestore) SaveAgentMemory(ctx context.Context, mem *memory.AgentMemory) error {
	if err := mem.Validate(); err != nil {
		return r.eb.Wrap(err, "invalid agent memory")
	}

	// Reject agent memories with invalid embeddings (nil, empty, or zero vector)
	if isInvalidEmbedding(mem.QueryEmbedding) {
		return r.eb.New("agent memory has invalid query embedding (nil, empty, or zero vector)",
			goerr.V("agent_id", mem.AgentID),
			goerr.V("id", mem.ID),
			goerr.V("embedding_length", len(mem.QueryEmbedding)))
	}

	doc := r.db.Collection(collectionAgents).Doc(mem.AgentID).
		Collection(subcollectionMemories).Doc(mem.ID.String())
	_, err := doc.Set(ctx, mem)
	if err != nil {
		return r.eb.Wrap(err, "failed to save agent memory",
			goerr.V("agent_id", mem.AgentID),
			goerr.V("id", mem.ID))
	}
	return nil
}

// GetAgentMemory retrieves an agent memory record by agentID and memoryID
// Path: agents/{agentID}/memories/{memoryID}
func (r *Firestore) GetAgentMemory(ctx context.Context, agentID string, id types.AgentMemoryID) (*memory.AgentMemory, error) {
	doc, err := r.db.Collection(collectionAgents).Doc(agentID).
		Collection(subcollectionMemories).Doc(id.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, r.eb.Wrap(goerr.New("agent memory not found"), "not found",
				goerr.T(errs.TagNotFound),
				goerr.V("agent_id", agentID),
				goerr.V("id", id))
		}
		return nil, r.eb.Wrap(err, "failed to get agent memory",
			goerr.V("agent_id", agentID),
			goerr.V("id", id))
	}

	var mem memory.AgentMemory
	if err := doc.DataTo(&mem); err != nil {
		return nil, r.eb.Wrap(err, "failed to unmarshal agent memory",
			goerr.V("agent_id", agentID),
			goerr.V("id", id))
	}

	return &mem, nil
}

// SearchMemoriesByEmbedding searches for agent memories by embedding similarity within an agent's subcollection
// Path: agents/{agentID}/memories/*
func (r *Firestore) SearchMemoriesByEmbedding(ctx context.Context, agentID string, embedding []float32, limit int) ([]*memory.AgentMemory, error) {
	// Query the agent's memories subcollection
	query := r.db.Collection(collectionAgents).Doc(agentID).
		Collection(subcollectionMemories)

	// Execute vector search using FindNearest
	// Results are automatically ordered by similarity (cosine distance)
	docs := query.FindNearest("QueryEmbedding",
		firestore.Vector32(embedding),
		limit,
		firestore.DistanceMeasureCosine,
		nil,
	).Documents(ctx)

	var memories []*memory.AgentMemory
	for {
		doc, err := docs.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, r.eb.Wrap(err, "failed to iterate search results", goerr.V("agent_id", agentID))
		}

		var mem memory.AgentMemory
		if err := doc.DataTo(&mem); err != nil {
			return nil, r.eb.Wrap(err, "failed to unmarshal agent memory", goerr.V("id", doc.Ref.ID))
		}

		memories = append(memories, &mem)
	}

	return memories, nil
}

func (r *Firestore) PutAgentMemory(ctx context.Context, mem *memory.AgentMemory) error {
	return r.SaveAgentMemory(ctx, mem)
}

// BatchSaveAgentMemories saves multiple agent memories in batches
// Firestore supports max 500 operations per batch, so this method splits if needed
func (r *Firestore) BatchSaveAgentMemories(ctx context.Context, memories []*memory.AgentMemory) error {
	if len(memories) == 0 {
		return nil
	}

	// Validate all memories first
	for _, mem := range memories {
		if err := mem.Validate(); err != nil {
			return r.eb.Wrap(err, "invalid agent memory in batch")
		}
		if isInvalidEmbedding(mem.QueryEmbedding) {
			return r.eb.New("agent memory has invalid query embedding",
				goerr.V("agent_id", mem.AgentID),
				goerr.V("id", mem.ID))
		}
	}

	// Use BulkWriter for efficient batch operations
	bulkWriter := r.db.BulkWriter(ctx)

	// Queue all save operations
	for _, mem := range memories {
		doc := r.db.Collection(collectionAgents).Doc(mem.AgentID).
			Collection(subcollectionMemories).Doc(mem.ID.String())
		_, err := bulkWriter.Set(doc, mem)
		if err != nil {
			return r.eb.Wrap(err, "failed to queue save operation",
				goerr.V("agent_id", mem.AgentID),
				goerr.V("memory_id", mem.ID))
		}
	}

	// Flush and wait for all operations to complete
	bulkWriter.End()

	return nil
}

// UpdateMemoryScoreBatch updates quality scores and last used timestamps for multiple agent memories
// Uses BulkWriter for efficient batch updates
func (r *Firestore) UpdateMemoryScoreBatch(ctx context.Context, agentID string, updates map[types.AgentMemoryID]struct {
	Score      float64
	LastUsedAt time.Time
}) error {
	if len(updates) == 0 {
		return nil
	}

	// Create a BulkWriter for efficient bulk operations
	bulkWriter := r.db.BulkWriter(ctx)

	// Queue all update operations
	for memoryID, update := range updates {
		doc := r.db.Collection(collectionAgents).Doc(agentID).
			Collection(subcollectionMemories).Doc(memoryID.String())

		updateFields := []firestore.Update{
			{Path: "Score", Value: update.Score},
			{Path: "LastUsedAt", Value: update.LastUsedAt},
		}

		_, err := bulkWriter.Update(doc, updateFields)
		if err != nil {
			return r.eb.Wrap(err, "failed to queue update operation",
				goerr.V("agent_id", agentID),
				goerr.V("memory_id", memoryID))
		}
	}

	// Flush and wait for all operations to complete
	bulkWriter.End()

	return nil
}

// DeleteAgentMemoriesBatch deletes multiple agent memories in a batch
// Uses BulkWriter for efficient batch deletion
// Returns the number of successfully deleted memories
func (r *Firestore) DeleteAgentMemoriesBatch(ctx context.Context, agentID string, memoryIDs []types.AgentMemoryID) (int, error) {
	if len(memoryIDs) == 0 {
		return 0, nil
	}

	// Create a BulkWriter for efficient bulk operations
	bulkWriter := r.db.BulkWriter(ctx)

	// Queue all delete operations
	for _, id := range memoryIDs {
		doc := r.db.Collection(collectionAgents).Doc(agentID).
			Collection(subcollectionMemories).Doc(id.String())
		_, err := bulkWriter.Delete(doc)
		if err != nil {
			return 0, r.eb.Wrap(err, "failed to queue delete operation",
				goerr.V("agent_id", agentID),
				goerr.V("memory_id", id))
		}
	}

	// Flush all operations
	bulkWriter.End()

	return len(memoryIDs), nil
}

// ListAgentMemories lists all memories for an agent
// Results are ordered by CreatedAt DESC
func (r *Firestore) ListAgentMemories(ctx context.Context, agentID string) ([]*memory.AgentMemory, error) {
	docs := r.db.Collection(collectionAgents).Doc(agentID).
		Collection(subcollectionMemories).
		OrderBy("CreatedAt", firestore.Desc).
		Documents(ctx)

	var memories []*memory.AgentMemory
	for {
		doc, err := docs.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, r.eb.Wrap(err, "failed to iterate agent memories", goerr.V("agent_id", agentID))
		}

		var mem memory.AgentMemory
		if err := doc.DataTo(&mem); err != nil {
			return nil, r.eb.Wrap(err, "failed to unmarshal agent memory", goerr.V("id", doc.Ref.ID))
		}

		memories = append(memories, &mem)
	}

	return memories, nil
}

// ListAllAgentIDs returns all agent IDs that have memories with their counts
// Uses CollectionGroup query to find all memories subcollections regardless of parent document existence
func (r *Firestore) ListAllAgentIDs(ctx context.Context) (map[string]int, error) {
	result := make(map[string]int)

	// Use CollectionGroup to query all memories subcollections
	memoryDocs := r.db.CollectionGroup(subcollectionMemories).Documents(ctx)

	for {
		doc, err := memoryDocs.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, r.eb.Wrap(err, "failed to iterate memory documents")
		}

		// Extract agent ID from document path: agents/{agentID}/memories/{memoryID}
		// doc.Ref.Path = "agents/{agentID}/memories/{memoryID}"
		agentID := doc.Ref.Parent.Parent.ID

		result[agentID]++
	}

	return result, nil
}
