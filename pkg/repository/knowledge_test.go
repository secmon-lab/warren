package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestKnowledgeRepository(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()

		t.Run("PutAndGet", func(t *testing.T) {
			now := time.Now()

			// Use random topic and slug to avoid conflicts
			topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", now.UnixNano()))
			slug := types.KnowledgeSlug(fmt.Sprintf("test-slug-%d", now.UnixNano()))

			k := &knowledge.Knowledge{
				Slug:      slug,
				Name:      "Test Knowledge",
				Topic:     topic,
				Content:   "This is test content",
				CommitID:  knowledge.GenerateCommitID(now, types.SystemUserID, "This is test content"),
				Author:    types.SystemUserID,
				CreatedAt: now,
				UpdatedAt: now,
				State:     types.KnowledgeStateActive,
			}

			// Put knowledge
			err := repo.PutKnowledge(ctx, k)
			gt.NoError(t, err)

			// Get knowledge
			retrieved, err := repo.GetKnowledge(ctx, topic, slug)
			gt.NoError(t, err)
			gt.NotEqual(t, retrieved, nil)
			gt.Equal(t, retrieved.Slug, k.Slug)
			gt.Equal(t, retrieved.Name, k.Name)
			gt.Equal(t, retrieved.Topic, k.Topic)
			gt.Equal(t, retrieved.Content, k.Content)
			gt.Equal(t, retrieved.CommitID, k.CommitID)
			gt.Equal(t, retrieved.Author, k.Author)
			gt.True(t, retrieved.CreatedAt.Sub(k.CreatedAt) < time.Second)
			gt.True(t, retrieved.UpdatedAt.Sub(k.UpdatedAt) < time.Second)
			gt.Equal(t, retrieved.State, k.State)
		})

		t.Run("GetKnowledges", func(t *testing.T) {
			now := time.Now()

			// Use random topic to avoid conflicts
			topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", now.UnixNano()))

			// Create multiple knowledges
			k1 := &knowledge.Knowledge{
				Slug:      "slug1",
				Name:      "Knowledge 1",
				Topic:     topic,
				Content:   "Content 1",
				CommitID:  knowledge.GenerateCommitID(now, types.SystemUserID, "Content 1"),
				Author:    types.SystemUserID,
				CreatedAt: now,
				UpdatedAt: now,
				State:     types.KnowledgeStateActive,
			}

			k2 := &knowledge.Knowledge{
				Slug:      "slug2",
				Name:      "Knowledge 2",
				Topic:     topic,
				Content:   "Content 2",
				CommitID:  knowledge.GenerateCommitID(now.Add(time.Second), types.SystemUserID, "Content 2"),
				Author:    types.SystemUserID,
				CreatedAt: now.Add(time.Second),
				UpdatedAt: now.Add(time.Second),
				State:     types.KnowledgeStateActive,
			}

			gt.NoError(t, repo.PutKnowledge(ctx, k1))
			gt.NoError(t, repo.PutKnowledge(ctx, k2))

			// Get all knowledges
			knowledges, err := repo.GetKnowledges(ctx, topic)
			gt.NoError(t, err)
			gt.Equal(t, len(knowledges), 2)

			// Verify knowledges (order may vary)
			slugs := map[types.KnowledgeSlug]bool{
				k1.Slug: false,
				k2.Slug: false,
			}
			for _, k := range knowledges {
				if _, ok := slugs[k.Slug]; ok {
					slugs[k.Slug] = true
				}
			}
			gt.True(t, slugs[k1.Slug])
			gt.True(t, slugs[k2.Slug])
		})

		t.Run("Versioning", func(t *testing.T) {
			now := time.Now()

			topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", now.UnixNano()))
			slug := types.KnowledgeSlug(fmt.Sprintf("test-slug-%d", now.UnixNano()))

			// Version 1
			v1 := &knowledge.Knowledge{
				Slug:      slug,
				Name:      "V1",
				Topic:     topic,
				Content:   "Version 1 content",
				CommitID:  knowledge.GenerateCommitID(now, types.SystemUserID, "Version 1 content"),
				Author:    types.SystemUserID,
				CreatedAt: now,
				UpdatedAt: now,
				State:     types.KnowledgeStateActive,
			}
			gt.NoError(t, repo.PutKnowledge(ctx, v1))

			// Version 2
			v2Time := now.Add(time.Second)
			v2 := &knowledge.Knowledge{
				Slug:      slug,
				Name:      "V2",
				Topic:     topic,
				Content:   "Version 2 content",
				CommitID:  knowledge.GenerateCommitID(v2Time, types.SystemUserID, "Version 2 content"),
				Author:    types.SystemUserID,
				CreatedAt: now, // Same created time
				UpdatedAt: v2Time,
				State:     types.KnowledgeStateActive,
			}
			gt.NoError(t, repo.PutKnowledge(ctx, v2))

			// Get latest should return v2
			latest, err := repo.GetKnowledge(ctx, topic, slug)
			gt.NoError(t, err)
			gt.NotEqual(t, latest, nil)
			gt.Equal(t, latest.Name, "V2")
			gt.Equal(t, latest.Content, "Version 2 content")
			gt.Equal(t, latest.CommitID, v2.CommitID)

			// Get by commit ID should return v1
			historical, err := repo.GetKnowledgeByCommit(ctx, topic, slug, v1.CommitID)
			gt.NoError(t, err)
			gt.NotEqual(t, historical, nil)
			gt.Equal(t, historical.Name, "V1")
			gt.Equal(t, historical.Content, "Version 1 content")
			gt.Equal(t, historical.CommitID, v1.CommitID)
		})

		t.Run("ListSlugs", func(t *testing.T) {
			now := time.Now()

			topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", now.UnixNano()))

			k1 := &knowledge.Knowledge{
				Slug:      "slug1",
				Name:      "Name 1",
				Topic:     topic,
				Content:   "Content 1",
				CommitID:  knowledge.GenerateCommitID(now, types.SystemUserID, "Content 1"),
				Author:    types.SystemUserID,
				CreatedAt: now,
				UpdatedAt: now,
				State:     types.KnowledgeStateActive,
			}

			k2 := &knowledge.Knowledge{
				Slug:      "slug2",
				Name:      "Name 2",
				Topic:     topic,
				Content:   "Content 2",
				CommitID:  knowledge.GenerateCommitID(now.Add(time.Second), types.SystemUserID, "Content 2"),
				Author:    types.SystemUserID,
				CreatedAt: now.Add(time.Second),
				UpdatedAt: now.Add(time.Second),
				State:     types.KnowledgeStateActive,
			}

			gt.NoError(t, repo.PutKnowledge(ctx, k1))
			gt.NoError(t, repo.PutKnowledge(ctx, k2))

			slugs, err := repo.ListKnowledgeSlugs(ctx, topic)
			gt.NoError(t, err)
			gt.Equal(t, len(slugs), 2)

			// Verify slugs and names
			slugMap := make(map[types.KnowledgeSlug]string)
			for _, s := range slugs {
				slugMap[s.Slug] = s.Name
			}
			gt.Equal(t, slugMap[k1.Slug], "Name 1")
			gt.Equal(t, slugMap[k2.Slug], "Name 2")
		})

		t.Run("Archive", func(t *testing.T) {
			now := time.Now()

			topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", now.UnixNano()))
			slug := types.KnowledgeSlug(fmt.Sprintf("test-slug-%d", now.UnixNano()))

			k := &knowledge.Knowledge{
				Slug:      slug,
				Name:      "Test",
				Topic:     topic,
				Content:   "Content",
				CommitID:  knowledge.GenerateCommitID(now, types.SystemUserID, "Content"),
				Author:    types.SystemUserID,
				CreatedAt: now,
				UpdatedAt: now,
				State:     types.KnowledgeStateActive,
			}
			gt.NoError(t, repo.PutKnowledge(ctx, k))

			// Verify it exists
			retrieved, err := repo.GetKnowledge(ctx, topic, slug)
			gt.NoError(t, err)
			gt.NotEqual(t, retrieved, nil)

			// Archive it
			err = repo.ArchiveKnowledge(ctx, topic, slug)
			gt.NoError(t, err)

			// Should not be retrieved anymore
			archived, err := repo.GetKnowledge(ctx, topic, slug)
			gt.NoError(t, err)
			gt.Equal(t, archived, nil)

			// Should not appear in list
			knowledges, err := repo.GetKnowledges(ctx, topic)
			gt.NoError(t, err)
			gt.Equal(t, len(knowledges), 0)

			// Should not appear in slugs
			slugs, err := repo.ListKnowledgeSlugs(ctx, topic)
			gt.NoError(t, err)
			gt.Equal(t, len(slugs), 0)

			// But historical access should still work
			historical, err := repo.GetKnowledgeByCommit(ctx, topic, slug, k.CommitID)
			gt.NoError(t, err)
			gt.NotEqual(t, historical, nil)
			gt.Equal(t, historical.CommitID, k.CommitID)
		})

		t.Run("CalculateSize", func(t *testing.T) {
			now := time.Now()

			topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", now.UnixNano()))

			// Empty topic
			size, err := repo.CalculateKnowledgeSize(ctx, topic)
			gt.NoError(t, err)
			gt.Equal(t, size, 0)

			// Add first knowledge
			k1 := &knowledge.Knowledge{
				Slug:      "slug1",
				Name:      "Name 1",
				Topic:     topic,
				Content:   "12345", // 5 bytes
				CommitID:  knowledge.GenerateCommitID(now, types.SystemUserID, "12345"),
				Author:    types.SystemUserID,
				CreatedAt: now,
				UpdatedAt: now,
				State:     types.KnowledgeStateActive,
			}
			gt.NoError(t, repo.PutKnowledge(ctx, k1))

			size, err = repo.CalculateKnowledgeSize(ctx, topic)
			gt.NoError(t, err)
			gt.Equal(t, size, 5)

			// Add second knowledge
			k2 := &knowledge.Knowledge{
				Slug:      "slug2",
				Name:      "Name 2",
				Topic:     topic,
				Content:   "1234567890", // 10 bytes
				CommitID:  knowledge.GenerateCommitID(now.Add(time.Second), types.SystemUserID, "1234567890"),
				Author:    types.SystemUserID,
				CreatedAt: now.Add(time.Second),
				UpdatedAt: now.Add(time.Second),
				State:     types.KnowledgeStateActive,
			}
			gt.NoError(t, repo.PutKnowledge(ctx, k2))

			size, err = repo.CalculateKnowledgeSize(ctx, topic)
			gt.NoError(t, err)
			gt.Equal(t, size, 15) // 5 + 10

			// Archive first knowledge
			err = repo.ArchiveKnowledge(ctx, topic, k1.Slug)
			gt.NoError(t, err)

			size, err = repo.CalculateKnowledgeSize(ctx, topic)
			gt.NoError(t, err)
			gt.Equal(t, size, 10) // Only k2 remains
		})

		t.Run("NonExistent", func(t *testing.T) {
			topic := types.KnowledgeTopic(fmt.Sprintf("nonexistent-%d", time.Now().UnixNano()))
			slug := types.KnowledgeSlug("nonexistent")

			// Get non-existent knowledge
			k, err := repo.GetKnowledge(ctx, topic, slug)
			gt.NoError(t, err)
			gt.Equal(t, k, nil)

			// List slugs in non-existent topic
			slugs, err := repo.ListKnowledgeSlugs(ctx, topic)
			gt.NoError(t, err)
			gt.True(t, len(slugs) == 0)

			// Get knowledges in non-existent topic
			knowledges, err := repo.GetKnowledges(ctx, topic)
			gt.NoError(t, err)
			gt.True(t, len(knowledges) == 0)

			// Calculate size of non-existent topic
			size, err := repo.CalculateKnowledgeSize(ctx, topic)
			gt.NoError(t, err)
			gt.Equal(t, size, 0)
		})
	}

	t.Run("Memory", func(t *testing.T) {
		testFn(t, repository.NewMemory())
	})

	t.Run("Firestore", func(t *testing.T) {
		testFn(t, newFirestoreClient(t))
	})
}
