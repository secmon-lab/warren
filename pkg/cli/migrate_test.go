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
	gt.Equal(t, len(config.Collections), 5) // alerts, tickets, lists, memories, records

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
		gt.Equal(t, len(col.Indexes), 3) // Embedding + __name__+Embedding + CreatedAt+__name__+Embedding

		// Check single-field Embedding vector index
		embeddingIndex := col.Indexes[0]
		gt.Equal(t, len(embeddingIndex.Fields), 1)
		gt.Equal(t, embeddingIndex.Fields[0].Path, "Embedding")
		gt.Value(t, embeddingIndex.Fields[0].Vector).NotNil()
		gt.Equal(t, embeddingIndex.Fields[0].Vector.Dimension, 256)

		// Check __name__ + Embedding composite index
		nameEmbeddingIndex := col.Indexes[1]
		gt.Equal(t, len(nameEmbeddingIndex.Fields), 2)
		gt.Equal(t, nameEmbeddingIndex.Fields[0].Path, "__name__")
		gt.Equal(t, nameEmbeddingIndex.Fields[0].Order, fireconf.OrderAscending)
		gt.Equal(t, nameEmbeddingIndex.Fields[1].Path, "Embedding")
		gt.Value(t, nameEmbeddingIndex.Fields[1].Vector).NotNil()
		gt.Equal(t, nameEmbeddingIndex.Fields[1].Vector.Dimension, 256)

		// Check CreatedAt + __name__ + Embedding composite index
		compositeIndex := col.Indexes[2]
		gt.Equal(t, len(compositeIndex.Fields), 3)
		gt.Equal(t, compositeIndex.Fields[0].Path, "CreatedAt")
		gt.Equal(t, compositeIndex.Fields[0].Order, fireconf.OrderDescending)
		gt.Equal(t, compositeIndex.Fields[1].Path, "__name__")
		gt.Equal(t, compositeIndex.Fields[1].Order, fireconf.OrderDescending)
		gt.Equal(t, compositeIndex.Fields[2].Path, "Embedding")
		gt.Value(t, compositeIndex.Fields[2].Vector).NotNil()
		gt.Equal(t, compositeIndex.Fields[2].Vector.Dimension, 256)
	})

	// Test tickets collection (should have Status+CreatedAt+__name__ index)
	t.Run("tickets collection", func(t *testing.T) {
		col := findCollection("tickets")
		gt.Value(t, col).NotNil()
		gt.Equal(t, len(col.Indexes), 4) // Embedding + __name__+Embedding + CreatedAt+__name__+Embedding + Status+CreatedAt+__name__

		// Check Status + CreatedAt + __name__ index
		statusIndex := col.Indexes[3]
		gt.Equal(t, len(statusIndex.Fields), 3)
		gt.Equal(t, statusIndex.Fields[0].Path, "Status")
		gt.Equal(t, statusIndex.Fields[0].Order, fireconf.OrderAscending)
		gt.Equal(t, statusIndex.Fields[1].Path, "CreatedAt")
		gt.Equal(t, statusIndex.Fields[1].Order, fireconf.OrderDescending)
		gt.Equal(t, statusIndex.Fields[2].Path, "__name__")
		gt.Equal(t, statusIndex.Fields[2].Order, fireconf.OrderDescending)
	})

	// Test lists collection
	t.Run("lists collection", func(t *testing.T) {
		col := findCollection("lists")
		gt.Value(t, col).NotNil()
		gt.Equal(t, len(col.Indexes), 3) // Same as alerts
	})

	// Test memories subcollection (COLLECTION scope)
	t.Run("memories subcollection", func(t *testing.T) {
		col := findCollection("memories")
		gt.Value(t, col).NotNil()
		gt.Equal(t, len(col.Indexes), 2) // QueryEmbedding + __name__+QueryEmbedding

		// Check single-field QueryEmbedding index
		memoryIndex := col.Indexes[0]
		gt.Equal(t, memoryIndex.QueryScope, fireconf.QueryScopeCollection)
		gt.Equal(t, len(memoryIndex.Fields), 1)
		gt.Equal(t, memoryIndex.Fields[0].Path, "QueryEmbedding")
		gt.Value(t, memoryIndex.Fields[0].Vector).NotNil()
		gt.Equal(t, memoryIndex.Fields[0].Vector.Dimension, 256)

		// Check __name__ + QueryEmbedding composite index
		nameQueryIndex := col.Indexes[1]
		gt.Equal(t, nameQueryIndex.QueryScope, fireconf.QueryScopeCollection)
		gt.Equal(t, len(nameQueryIndex.Fields), 2)
		gt.Equal(t, nameQueryIndex.Fields[0].Path, "__name__")
		gt.Equal(t, nameQueryIndex.Fields[0].Order, fireconf.OrderAscending)
		gt.Equal(t, nameQueryIndex.Fields[1].Path, "QueryEmbedding")
		gt.Value(t, nameQueryIndex.Fields[1].Vector).NotNil()
		gt.Equal(t, nameQueryIndex.Fields[1].Vector.Dimension, 256)
	})
}
