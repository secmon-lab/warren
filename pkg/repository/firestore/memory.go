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

// Memory related methods
func (r *Firestore) GetExecutionMemory(ctx context.Context, schemaID types.AlertSchema) (*memory.ExecutionMemory, error) {
	docs, err := r.db.Collection(collectionExecutionMemories).
		Doc(string(schemaID)).
		Collection("records").
		OrderBy("CreatedAt", firestore.Desc).
		Limit(1).
		Documents(ctx).GetAll()

	if err != nil {
		return nil, r.eb.Wrap(err, "failed to get execution memory", goerr.V("schema_id", schemaID))
	}

	if len(docs) == 0 {
		return nil, nil
	}

	var mem memory.ExecutionMemory
	if err := docs[0].DataTo(&mem); err != nil {
		return nil, r.eb.Wrap(err, "failed to convert data to execution memory", goerr.V("schema_id", schemaID))
	}

	mem.SchemaID = schemaID
	return &mem, nil
}

func (r *Firestore) PutExecutionMemory(ctx context.Context, mem *memory.ExecutionMemory) error {
	if err := mem.Validate(); err != nil {
		return r.eb.Wrap(err, "invalid execution memory")
	}

	// Reject execution memories with invalid embeddings (nil, empty, or zero vector)
	if isInvalidEmbedding(mem.Embedding) {
		return r.eb.New("execution memory has invalid embedding (nil, empty, or zero vector)",
			goerr.V("schema_id", mem.SchemaID),
			goerr.V("id", mem.ID),
			goerr.V("embedding_length", len(mem.Embedding)))
	}

	// Save as a new record in sub-collection
	doc := r.db.Collection(collectionExecutionMemories).
		Doc(string(mem.SchemaID)).
		Collection("records").
		Doc(mem.ID.String())

	_, err := doc.Set(ctx, mem)
	if err != nil {
		return r.eb.Wrap(err, "failed to put execution memory", goerr.V("schema_id", mem.SchemaID), goerr.V("id", mem.ID))
	}
	return nil
}

// SearchExecutionMemoriesByEmbedding searches execution memories by embedding similarity
func (r *Firestore) SearchExecutionMemoriesByEmbedding(ctx context.Context, schemaID types.AlertSchema, embedding []float32, limit int) ([]*memory.ExecutionMemory, error) {
	if len(embedding) == 0 {
		return nil, nil
	}

	// Convert []float32 to firestore.Vector32
	vector32 := firestore.Vector32(embedding[:])

	// Build vector search query on the sub-collection
	iter := r.db.Collection(collectionExecutionMemories).
		Doc(string(schemaID)).
		Collection("records").
		FindNearest("Embedding",
			vector32,
			limit,
			firestore.DistanceMeasureCosine,
			&firestore.FindNearestOptions{
				DistanceResultField: "vector_distance",
			}).Documents(ctx)

	result := make([]*memory.ExecutionMemory, 0, limit)
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, r.eb.Wrap(err, "failed to get next execution memory")
		}

		var mem memory.ExecutionMemory
		if err := doc.DataTo(&mem); err != nil {
			return nil, r.eb.Wrap(err, "failed to convert data to execution memory")
		}
		mem.SchemaID = schemaID
		result = append(result, &mem)
	}

	return result, nil
}

func (r *Firestore) GetTicketMemory(ctx context.Context, schemaID types.AlertSchema) (*memory.TicketMemory, error) {
	docs, err := r.db.Collection(collectionTicketMemories).
		Doc(string(schemaID)).
		Collection("records").
		OrderBy("CreatedAt", firestore.Desc).
		Limit(1).
		Documents(ctx).GetAll()

	if err != nil {
		return nil, r.eb.Wrap(err, "failed to get ticket memory", goerr.V("schema_id", schemaID))
	}

	if len(docs) == 0 {
		return nil, nil
	}

	var mem memory.TicketMemory
	if err := docs[0].DataTo(&mem); err != nil {
		return nil, r.eb.Wrap(err, "failed to convert data to ticket memory", goerr.V("schema_id", schemaID))
	}

	mem.SchemaID = schemaID
	return &mem, nil
}

func (r *Firestore) PutTicketMemory(ctx context.Context, mem *memory.TicketMemory) error {
	if err := mem.Validate(); err != nil {
		return r.eb.Wrap(err, "invalid ticket memory")
	}

	// Save as a new record in sub-collection
	doc := r.db.Collection(collectionTicketMemories).
		Doc(string(mem.SchemaID)).
		Collection("records").
		Doc(mem.ID.String())

	_, err := doc.Set(ctx, mem)
	if err != nil {
		return r.eb.Wrap(err, "failed to put ticket memory", goerr.V("schema_id", mem.SchemaID), goerr.V("id", mem.ID))
	}
	return nil
}

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

func (r *Firestore) GetExecutionMemories(ctx context.Context, schemaID types.AlertSchema, limit int) ([]*memory.ExecutionMemory, error) {
	docs, err := r.db.Collection(collectionExecutionMemories).
		Doc(string(schemaID)).
		Collection("records").
		OrderBy("CreatedAt", firestore.Desc).
		Limit(limit).
		Documents(ctx).GetAll()

	if err != nil {
		return nil, r.eb.Wrap(err, "failed to get execution memories", goerr.V("schema_id", schemaID))
	}

	var memories []*memory.ExecutionMemory
	for _, doc := range docs {
		var mem memory.ExecutionMemory
		if err := doc.DataTo(&mem); err != nil {
			return nil, r.eb.Wrap(err, "failed to convert data to execution memory", goerr.V("schema_id", schemaID))
		}
		mem.SchemaID = schemaID
		memories = append(memories, &mem)
	}

	return memories, nil
}

func (r *Firestore) DeleteExecutionMemory(ctx context.Context, schemaID types.AlertSchema, memoryID types.MemoryID) error {
	doc := r.db.Collection(collectionExecutionMemories).
		Doc(string(schemaID)).
		Collection("records").
		Doc(memoryID.String())

	_, err := doc.Delete(ctx)
	if err != nil {
		return r.eb.Wrap(err, "failed to delete execution memory", goerr.V("schema_id", schemaID), goerr.V("memory_id", memoryID))
	}
	return nil
}

func (r *Firestore) GetTicketMemories(ctx context.Context, schemaID types.AlertSchema, limit int) ([]*memory.TicketMemory, error) {
	docs, err := r.db.Collection(collectionTicketMemories).
		Doc(string(schemaID)).
		Collection("records").
		OrderBy("CreatedAt", firestore.Desc).
		Limit(limit).
		Documents(ctx).GetAll()

	if err != nil {
		return nil, r.eb.Wrap(err, "failed to get ticket memories", goerr.V("schema_id", schemaID))
	}

	var memories []*memory.TicketMemory
	for _, doc := range docs {
		var mem memory.TicketMemory
		if err := doc.DataTo(&mem); err != nil {
			return nil, r.eb.Wrap(err, "failed to convert data to ticket memory", goerr.V("schema_id", schemaID))
		}
		mem.SchemaID = schemaID
		memories = append(memories, &mem)
	}

	return memories, nil
}

func (r *Firestore) DeleteTicketMemory(ctx context.Context, schemaID types.AlertSchema, memoryID types.MemoryID) error {
	doc := r.db.Collection(collectionTicketMemories).
		Doc(string(schemaID)).
		Collection("records").
		Doc(memoryID.String())

	_, err := doc.Delete(ctx)
	if err != nil {
		return r.eb.Wrap(err, "failed to delete ticket memory", goerr.V("schema_id", schemaID), goerr.V("memory_id", memoryID))
	}
	return nil
}

func (r *Firestore) PutAgentMemory(ctx context.Context, mem *memory.AgentMemory) error {
	return r.SaveAgentMemory(ctx, mem)
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
			{Path: "QualityScore", Value: update.Score},
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
// Results are ordered by Timestamp DESC
func (r *Firestore) ListAgentMemories(ctx context.Context, agentID string) ([]*memory.AgentMemory, error) {
	docs := r.db.Collection(collectionAgents).Doc(agentID).
		Collection(subcollectionMemories).
		OrderBy("Timestamp", firestore.Desc).
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
