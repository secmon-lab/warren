package cli_test

import (
	"testing"

	"github.com/m-mizutani/fireconf"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli"
)

func TestDefineFirestoreIndexes(t *testing.T) {
	config := cli.DefineFirestoreIndexes()

	gt.Value(t, config).NotNil()
	gt.Equal(t, len(config.Collections), 6) // alerts, tickets, lists, execution_memories, ticket_memories, memories

	// Helper function to find collection by name
	findCollection := func(name string) *fireconf.Collection {
		for _, col := range config.Collections {
			if col.Name == name {
				return &col
			}
		}
		return nil
	}

	// Test alerts collection
	t.Run("alerts collection", func(t *testing.T) {
		col := findCollection("alerts")
		gt.Value(t, col).NotNil()
		gt.Equal(t, len(col.Indexes), 2) // Embedding index + CreatedAt+Embedding index

		// Check Embedding vector index
		embeddingIndex := col.Indexes[0]
		gt.Equal(t, len(embeddingIndex.Fields), 1)
		gt.Equal(t, embeddingIndex.Fields[0].Path, "Embedding")
		gt.Value(t, embeddingIndex.Fields[0].Vector).NotNil()
		gt.Equal(t, embeddingIndex.Fields[0].Vector.Dimension, 256)

		// Check CreatedAt + Embedding composite index
		compositeIndex := col.Indexes[1]
		gt.Equal(t, len(compositeIndex.Fields), 2)
		gt.Equal(t, compositeIndex.Fields[0].Path, "CreatedAt")
		gt.Equal(t, compositeIndex.Fields[0].Order, fireconf.OrderDescending)
		gt.Equal(t, compositeIndex.Fields[1].Path, "Embedding")
		gt.Value(t, compositeIndex.Fields[1].Vector).NotNil()
		gt.Equal(t, compositeIndex.Fields[1].Vector.Dimension, 256)
	})

	// Test tickets collection (should have Status+CreatedAt index)
	t.Run("tickets collection", func(t *testing.T) {
		col := findCollection("tickets")
		gt.Value(t, col).NotNil()
		gt.Equal(t, len(col.Indexes), 3) // Embedding + CreatedAt+Embedding + Status+CreatedAt

		// Check Status + CreatedAt index
		statusIndex := col.Indexes[2]
		gt.Equal(t, len(statusIndex.Fields), 2)
		gt.Equal(t, statusIndex.Fields[0].Path, "Status")
		gt.Equal(t, statusIndex.Fields[0].Order, fireconf.OrderAscending)
		gt.Equal(t, statusIndex.Fields[1].Path, "CreatedAt")
		gt.Equal(t, statusIndex.Fields[1].Order, fireconf.OrderDescending)
	})

	// Test lists collection
	t.Run("lists collection", func(t *testing.T) {
		col := findCollection("lists")
		gt.Value(t, col).NotNil()
		gt.Equal(t, len(col.Indexes), 2) // Same as alerts
	})

	// Test execution_memories collection
	t.Run("execution_memories collection", func(t *testing.T) {
		col := findCollection("execution_memories")
		gt.Value(t, col).NotNil()
		gt.Equal(t, len(col.Indexes), 2) // QueryEmbedding + created_at+QueryEmbedding

		// Check QueryEmbedding vector index
		embeddingIndex := col.Indexes[0]
		gt.Equal(t, len(embeddingIndex.Fields), 1)
		gt.Equal(t, embeddingIndex.Fields[0].Path, "QueryEmbedding")
		gt.Value(t, embeddingIndex.Fields[0].Vector).NotNil()
		gt.Equal(t, embeddingIndex.Fields[0].Vector.Dimension, 256)

		// Check created_at + QueryEmbedding composite index
		compositeIndex := col.Indexes[1]
		gt.Equal(t, len(compositeIndex.Fields), 2)
		gt.Equal(t, compositeIndex.Fields[0].Path, "created_at")
		gt.Equal(t, compositeIndex.Fields[0].Order, fireconf.OrderDescending)
		gt.Equal(t, compositeIndex.Fields[1].Path, "QueryEmbedding")
		gt.Value(t, compositeIndex.Fields[1].Vector).NotNil()
		gt.Equal(t, compositeIndex.Fields[1].Vector.Dimension, 256)
	})

	// Test ticket_memories collection
	t.Run("ticket_memories collection", func(t *testing.T) {
		col := findCollection("ticket_memories")
		gt.Value(t, col).NotNil()
		gt.Equal(t, len(col.Indexes), 2) // Same as execution_memories
	})

	// Test memories subcollection (COLLECTION scope)
	t.Run("memories subcollection", func(t *testing.T) {
		col := findCollection("memories")
		gt.Value(t, col).NotNil()
		gt.Equal(t, len(col.Indexes), 1)

		memoryIndex := col.Indexes[0]
		gt.Equal(t, memoryIndex.QueryScope, fireconf.QueryScopeCollection)
		gt.Equal(t, len(memoryIndex.Fields), 1)
		gt.Equal(t, memoryIndex.Fields[0].Path, "QueryEmbedding")
		gt.Value(t, memoryIndex.Fields[0].Vector).NotNil()
		gt.Equal(t, memoryIndex.Fields[0].Vector.Dimension, 256)
	})
}
