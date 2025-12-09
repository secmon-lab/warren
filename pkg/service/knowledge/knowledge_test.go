package knowledge_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/knowledge"
)

func TestServiceSaveAndGet(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	svc := knowledge.New(repo)

	topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", time.Now().UnixNano()))
	slug := types.KnowledgeSlug("test-slug")

	// Save knowledge
	commitID, err := svc.SaveKnowledge(ctx, topic, slug, "Test Name", "Test content", types.SystemUserID)
	gt.NoError(t, err)
	gt.NotEqual(t, commitID, "")

	// Get knowledge
	k, err := svc.GetKnowledge(ctx, topic, slug)
	gt.NoError(t, err)
	gt.NotEqual(t, k, nil)
	gt.Equal(t, k.Slug, slug)
	gt.Equal(t, k.Name, "Test Name")
	gt.Equal(t, k.Content, "Test content")
	gt.Equal(t, k.CommitID, commitID)
}

func TestServiceQuotaCheck(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	svc := knowledge.New(repo)

	topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", time.Now().UnixNano()))

	// Fill up to just under the limit (10KB)
	// Use 9KB first
	content9KB := strings.Repeat("a", 9*1024)
	_, err := svc.SaveKnowledge(ctx, topic, "slug1", "Name 1", content9KB, types.SystemUserID)
	gt.NoError(t, err)

	// Add 1KB more (should succeed, total 10KB)
	content1KB := strings.Repeat("b", 1*1024)
	_, err = svc.SaveKnowledge(ctx, topic, "slug2", "Name 2", content1KB, types.SystemUserID)
	gt.NoError(t, err)

	// Try to add 1 more byte (should fail)
	_, err = svc.SaveKnowledge(ctx, topic, "slug3", "Name 3", "x", types.SystemUserID)
	gt.Error(t, err)
	gt.True(t, errors.Is(err, errs.ErrKnowledgeQuotaExceeded))
}

func TestServiceQuotaWithUpdate(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	svc := knowledge.New(repo)

	topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", time.Now().UnixNano()))

	// Save initial knowledge (5KB)
	content5KB := strings.Repeat("a", 5*1024)
	_, err := svc.SaveKnowledge(ctx, topic, "slug1", "Name 1", content5KB, types.SystemUserID)
	gt.NoError(t, err)

	// Update to 6KB (should succeed, only uses 6KB total)
	content6KB := strings.Repeat("b", 6*1024)
	_, err = svc.SaveKnowledge(ctx, topic, "slug1", "Name 1 Updated", content6KB, types.SystemUserID)
	gt.NoError(t, err)

	// Verify the update
	k, err := svc.GetKnowledge(ctx, topic, "slug1")
	gt.NoError(t, err)
	gt.Equal(t, k.Name, "Name 1 Updated")
	gt.Equal(t, len(k.Content), 6*1024)
}

func TestServiceArchive(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	svc := knowledge.New(repo)

	topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", time.Now().UnixNano()))

	// Save knowledge
	_, err := svc.SaveKnowledge(ctx, topic, "slug1", "Name 1", "Content 1", types.SystemUserID)
	gt.NoError(t, err)

	// Archive it
	err = svc.ArchiveKnowledge(ctx, topic, "slug1")
	gt.NoError(t, err)

	// Should not be retrievable anymore
	k, err := svc.GetKnowledge(ctx, topic, "slug1")
	gt.NoError(t, err)
	gt.Equal(t, k, nil)
}

func TestServiceListSlugs(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	svc := knowledge.New(repo)

	topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", time.Now().UnixNano()))

	// Add knowledges
	_, err := svc.SaveKnowledge(ctx, topic, "slug1", "Name 1", "Content 1", types.SystemUserID)
	gt.NoError(t, err)
	_, err = svc.SaveKnowledge(ctx, topic, "slug2", "Name 2", "Content 2", types.SystemUserID)
	gt.NoError(t, err)

	// List slugs
	slugs, err := svc.ListSlugs(ctx, topic)
	gt.NoError(t, err)
	gt.Equal(t, len(slugs), 2)

	// Verify slugs
	slugMap := make(map[types.KnowledgeSlug]string)
	for _, s := range slugs {
		slugMap[s.Slug] = s.Name
	}
	gt.Equal(t, slugMap["slug1"], "Name 1")
	gt.Equal(t, slugMap["slug2"], "Name 2")
}

func TestServiceValidation(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	svc := knowledge.New(repo)

	topic := types.KnowledgeTopic("test-topic")
	slug := types.KnowledgeSlug("test-slug")

	t.Run("empty name", func(t *testing.T) {
		_, err := svc.SaveKnowledge(ctx, topic, slug, "", "Content", types.SystemUserID)
		gt.Error(t, err)
	})

	t.Run("name too long", func(t *testing.T) {
		longName := strings.Repeat("a", 101)
		_, err := svc.SaveKnowledge(ctx, topic, slug, longName, "Content", types.SystemUserID)
		gt.Error(t, err)
	})

	t.Run("empty content", func(t *testing.T) {
		_, err := svc.SaveKnowledge(ctx, topic, slug, "Name", "", types.SystemUserID)
		gt.Error(t, err)
	})

	t.Run("empty topic defaults to 'default'", func(t *testing.T) {
		commitID, err := svc.SaveKnowledge(ctx, "", slug, "Name", "Content", types.SystemUserID)
		gt.NoError(t, err)
		gt.V(t, commitID).NotEqual("")

		// Verify knowledge was saved with default topic
		k, err := svc.GetKnowledge(ctx, types.KnowledgeTopic(knowledge.DefaultKnowledgeTopic), slug)
		gt.NoError(t, err)
		gt.V(t, k).NotNil()
		gt.Equal(t, k.Topic, types.KnowledgeTopic(knowledge.DefaultKnowledgeTopic))
	})

	t.Run("empty slug", func(t *testing.T) {
		_, err := svc.SaveKnowledge(ctx, topic, "", "Name", "Content", types.SystemUserID)
		gt.Error(t, err)
	})

	t.Run("empty author", func(t *testing.T) {
		_, err := svc.SaveKnowledge(ctx, topic, slug, "Name", "Content", "")
		gt.Error(t, err)
	})
}
