package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/m-mizutani/fireconf"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli"
)

func TestDefineFirestoreIndexes(t *testing.T) {
	config := cli.DefineFirestoreIndexes()

	gt.Value(t, config).NotNil()
	// alerts, tickets: vector and composite indexes (unchanged from main).
	// lists, memories, records: declared with empty index sets so that
	//   fireconf's Migrate can clean up residual indexes left over from
	//   previous deployments.
	gt.Equal(t, len(config.Collections), 5)

	findCollection := func(name string) *fireconf.Collection {
		for _, col := range config.Collections {
			if col.Name == name {
				return &col
			}
		}
		return nil
	}

	t.Run("alerts collection", func(t *testing.T) {
		col := findCollection("alerts")
		gt.Value(t, col).NotNil()
		// Embedding + __name__+Embedding + CreatedAt+__name__+Embedding
		gt.Equal(t, len(col.Indexes), 3)

		// Single-field Embedding vector index
		embeddingIndex := col.Indexes[0]
		gt.Equal(t, len(embeddingIndex.Fields), 1)
		gt.Equal(t, embeddingIndex.Fields[0].Path, "Embedding")
		gt.Value(t, embeddingIndex.Fields[0].Vector).NotNil()
		gt.Equal(t, embeddingIndex.Fields[0].Vector.Dimension, 256)

		// __name__ + Embedding composite
		nameEmbeddingIndex := col.Indexes[1]
		gt.Equal(t, len(nameEmbeddingIndex.Fields), 2)
		gt.Equal(t, nameEmbeddingIndex.Fields[0].Path, "__name__")
		gt.Equal(t, nameEmbeddingIndex.Fields[0].Order, fireconf.OrderAscending)
		gt.Equal(t, nameEmbeddingIndex.Fields[1].Path, "Embedding")
		gt.Value(t, nameEmbeddingIndex.Fields[1].Vector).NotNil()
		gt.Equal(t, nameEmbeddingIndex.Fields[1].Vector.Dimension, 256)

		// CreatedAt + __name__ + Embedding composite
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

	t.Run("tickets collection", func(t *testing.T) {
		col := findCollection("tickets")
		gt.Value(t, col).NotNil()
		// Embedding + __name__+Embedding + CreatedAt+__name__+Embedding + Status+CreatedAt+__name__
		gt.Equal(t, len(col.Indexes), 4)

		// Status + CreatedAt + __name__ composite (for GetTicketsByStatusAndSpan)
		statusIndex := col.Indexes[3]
		gt.Equal(t, len(statusIndex.Fields), 3)
		gt.Equal(t, statusIndex.Fields[0].Path, "Status")
		gt.Equal(t, statusIndex.Fields[0].Order, fireconf.OrderAscending)
		gt.Equal(t, statusIndex.Fields[1].Path, "CreatedAt")
		gt.Equal(t, statusIndex.Fields[1].Order, fireconf.OrderDescending)
		gt.Equal(t, statusIndex.Fields[2].Path, "__name__")
		gt.Equal(t, statusIndex.Fields[2].Order, fireconf.OrderDescending)
	})

	t.Run("deprecated collections declared with empty index set", func(t *testing.T) {
		// These collections previously had index declarations that did not
		// correspond to any real query. They remain declared with zero indexes
		// so that fireconf's Migrate can sweep up residual indexes left over
		// in Firestore from previous deployments.
		for _, name := range []string{"lists", "memories", "records"} {
			col := findCollection(name)
			gt.Value(t, col).NotNil()
			gt.Equal(t, len(col.Indexes), 0)
		}
	})
}

func TestFormatIndexFields(t *testing.T) {
	t.Run("single ascending order field", func(t *testing.T) {
		got := cli.FormatIndexFieldsForTest([]fireconf.IndexField{
			{Path: "Status", Order: fireconf.OrderAscending},
		})
		gt.Equal(t, got, "[Status asc]")
	})

	t.Run("single descending order field", func(t *testing.T) {
		got := cli.FormatIndexFieldsForTest([]fireconf.IndexField{
			{Path: "CreatedAt", Order: fireconf.OrderDescending},
		})
		gt.Equal(t, got, "[CreatedAt desc]")
	})

	t.Run("vector field", func(t *testing.T) {
		got := cli.FormatIndexFieldsForTest([]fireconf.IndexField{
			{Path: "Embedding", Vector: &fireconf.VectorConfig{Dimension: 256}},
		})
		gt.Equal(t, got, "[Embedding vector(256)]")
	})

	t.Run("composite with vector last", func(t *testing.T) {
		got := cli.FormatIndexFieldsForTest([]fireconf.IndexField{
			{Path: "CreatedAt", Order: fireconf.OrderDescending},
			{Path: "__name__", Order: fireconf.OrderDescending},
			{Path: "Embedding", Vector: &fireconf.VectorConfig{Dimension: 256}},
		})
		gt.Equal(t, got, "[CreatedAt desc, __name__ desc, Embedding vector(256)]")
	})

	t.Run("array contains", func(t *testing.T) {
		got := cli.FormatIndexFieldsForTest([]fireconf.IndexField{
			{Path: "tags", Array: fireconf.ArrayConfigContains},
		})
		gt.Equal(t, got, "[tags array-contains]")
	})
}

func TestPrintMigrationPlan(t *testing.T) {
	want := &fireconf.Config{
		Collections: []fireconf.Collection{
			{
				Name: "alerts",
				Indexes: []fireconf.Index{
					{
						QueryScope: fireconf.QueryScopeCollection,
						Fields: []fireconf.IndexField{
							{Path: "Embedding", Vector: &fireconf.VectorConfig{Dimension: 256}},
						},
					},
					{
						QueryScope: fireconf.QueryScopeCollection,
						Fields: []fireconf.IndexField{
							{Path: "Status", Order: fireconf.OrderAscending},
							{Path: "CreatedAt", Order: fireconf.OrderDescending},
						},
					},
				},
			},
		},
	}

	t.Run("mixed add delete keep", func(t *testing.T) {
		diff := &fireconf.DiffResult{
			Collections: []fireconf.CollectionDiff{
				{
					Name: "alerts",
					IndexesToAdd: []fireconf.Index{
						{
							QueryScope: fireconf.QueryScopeCollection,
							Fields: []fireconf.IndexField{
								{Path: "Status", Order: fireconf.OrderAscending},
								{Path: "CreatedAt", Order: fireconf.OrderDescending},
							},
						},
					},
				},
				{
					Name: "lists",
					IndexesToDelete: []fireconf.Index{
						{
							QueryScope: fireconf.QueryScopeCollection,
							Fields: []fireconf.IndexField{
								{Path: "Embedding", Vector: &fireconf.VectorConfig{Dimension: 256}},
							},
						},
					},
				},
			},
		}

		var buf bytes.Buffer
		cli.PrintMigrationPlanForTest(&buf, "proj", "(default)", true, want, diff)
		out := buf.String()

		gt.True(t, strings.Contains(out, "Mode:     DRY-RUN"))
		gt.True(t, strings.Contains(out, "Project:  proj"))
		gt.True(t, strings.Contains(out, "Database: (default)"))

		gt.True(t, strings.Contains(out, "Collection: alerts"))
		gt.True(t, strings.Contains(out, "  + ADD    [Status asc, CreatedAt desc]"))
		gt.True(t, strings.Contains(out, "    KEEP   [Embedding vector(256)]"))

		gt.True(t, strings.Contains(out, "Collection: lists (no longer declared)"))
		gt.True(t, strings.Contains(out, "  - DELETE [Embedding vector(256)]"))

		gt.True(t, strings.Contains(out, "Summary: 1 to add, 1 to delete, 1 unchanged (2 total declared)."))
	})

	t.Run("no changes", func(t *testing.T) {
		diff := &fireconf.DiffResult{}

		var buf bytes.Buffer
		cli.PrintMigrationPlanForTest(&buf, "proj", "(default)", false, want, diff)
		out := buf.String()

		gt.True(t, strings.Contains(out, "Mode:     APPLY"))
		gt.True(t, strings.Contains(out, "    KEEP   [Embedding vector(256)]"))
		gt.True(t, strings.Contains(out, "    KEEP   [Status asc, CreatedAt desc]"))
		gt.True(t, strings.Contains(out, "Summary: 0 to add, 0 to delete, 2 unchanged (2 total declared)."))
	})

	t.Run("empty declared collection with pending deletions", func(t *testing.T) {
		// Simulates a deprecated collection that is declared with no indexes
		// but still has residual indexes in Firestore to clean up.
		wantWithEmpty := &fireconf.Config{
			Collections: []fireconf.Collection{
				{Name: "lists", Indexes: nil},
			},
		}
		diff := &fireconf.DiffResult{
			Collections: []fireconf.CollectionDiff{
				{
					Name: "lists",
					IndexesToDelete: []fireconf.Index{
						{
							QueryScope: fireconf.QueryScopeCollection,
							Fields: []fireconf.IndexField{
								{Path: "Embedding", Vector: &fireconf.VectorConfig{Dimension: 256}},
							},
						},
					},
				},
			},
		}

		var buf bytes.Buffer
		cli.PrintMigrationPlanForTest(&buf, "proj", "(default)", true, wantWithEmpty, diff)
		out := buf.String()

		gt.True(t, strings.Contains(out, "Collection: lists"))
		gt.True(t, strings.Contains(out, "  - DELETE [Embedding vector(256)]"))
		// No "(no indexes declared…)" hint because there IS a deletion to show.
		gt.False(t, strings.Contains(out, "(no indexes declared"))
		gt.True(t, strings.Contains(out, "Summary: 0 to add, 1 to delete, 0 unchanged (0 total declared)."))
	})

	t.Run("empty declared collection with no pending work", func(t *testing.T) {
		// Simulates a deprecated collection that has already been fully cleaned up.
		wantWithEmpty := &fireconf.Config{
			Collections: []fireconf.Collection{
				{Name: "memories", Indexes: nil},
			},
		}

		var buf bytes.Buffer
		cli.PrintMigrationPlanForTest(&buf, "proj", "(default)", true, wantWithEmpty, &fireconf.DiffResult{})
		out := buf.String()

		gt.True(t, strings.Contains(out, "Collection: memories"))
		gt.True(t, strings.Contains(out, "  (no indexes declared, no existing indexes to remove)"))
	})
}
