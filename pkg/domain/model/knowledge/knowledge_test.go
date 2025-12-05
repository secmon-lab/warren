package knowledge_test

import (
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func TestKnowledgeValidate(t *testing.T) {
	now := time.Now()
	validKnowledge := &knowledge.Knowledge{
		Slug:      "test-slug",
		Name:      "Test Knowledge",
		Topic:     "test-topic",
		Content:   "Test content",
		CommitID:  "abc123",
		Author:    types.SystemUserID,
		CreatedAt: now,
		UpdatedAt: now,
		State:     types.KnowledgeStateActive,
	}

	t.Run("valid knowledge", func(t *testing.T) {
		err := validKnowledge.Validate()
		gt.NoError(t, err)
	})

	t.Run("empty slug", func(t *testing.T) {
		k := *validKnowledge
		k.Slug = ""
		err := k.Validate()
		gt.Error(t, err)
	})

	t.Run("empty name", func(t *testing.T) {
		k := *validKnowledge
		k.Name = ""
		err := k.Validate()
		gt.Error(t, err)
	})

	t.Run("name too long", func(t *testing.T) {
		k := *validKnowledge
		k.Name = string(make([]byte, 101))
		err := k.Validate()
		gt.Error(t, err)
	})

	t.Run("empty topic", func(t *testing.T) {
		k := *validKnowledge
		k.Topic = ""
		err := k.Validate()
		gt.Error(t, err)
	})

	t.Run("empty content", func(t *testing.T) {
		k := *validKnowledge
		k.Content = ""
		err := k.Validate()
		gt.Error(t, err)
	})

	t.Run("empty commit_id", func(t *testing.T) {
		k := *validKnowledge
		k.CommitID = ""
		err := k.Validate()
		gt.Error(t, err)
	})

	t.Run("empty author", func(t *testing.T) {
		k := *validKnowledge
		k.Author = ""
		err := k.Validate()
		gt.Error(t, err)
	})

	t.Run("invalid state", func(t *testing.T) {
		k := *validKnowledge
		k.State = "invalid"
		err := k.Validate()
		gt.Error(t, err)
	})
}

func TestGenerateCommitID(t *testing.T) {
	now := time.Now()
	author := types.SystemUserID
	content := "Test content"

	t.Run("deterministic", func(t *testing.T) {
		id1 := knowledge.GenerateCommitID(now, author, content)
		id2 := knowledge.GenerateCommitID(now, author, content)
		gt.Equal(t, id1, id2)
	})

	t.Run("different time produces different ID", func(t *testing.T) {
		id1 := knowledge.GenerateCommitID(now, author, content)
		id2 := knowledge.GenerateCommitID(now.Add(time.Second), author, content)
		gt.NotEqual(t, id1, id2)
	})

	t.Run("different author produces different ID", func(t *testing.T) {
		id1 := knowledge.GenerateCommitID(now, author, content)
		id2 := knowledge.GenerateCommitID(now, "user123", content)
		gt.NotEqual(t, id1, id2)
	})

	t.Run("different content produces different ID", func(t *testing.T) {
		id1 := knowledge.GenerateCommitID(now, author, content)
		id2 := knowledge.GenerateCommitID(now, author, "Different content")
		gt.NotEqual(t, id1, id2)
	})

	t.Run("ID is hex string", func(t *testing.T) {
		id := knowledge.GenerateCommitID(now, author, content)
		gt.Equal(t, len(id), 64) // SHA256 produces 64 hex characters
	})
}

func TestKnowledgeSize(t *testing.T) {
	k := &knowledge.Knowledge{
		Content: "Test content",
	}

	size := k.Size()
	gt.Equal(t, size, len("Test content"))
}

func TestUserIDHelpers(t *testing.T) {
	t.Run("IsSystem returns true for system user", func(t *testing.T) {
		gt.True(t, types.SystemUserID.IsSystem())
	})

	t.Run("IsSystem returns false for other users", func(t *testing.T) {
		user := types.UserID("user123")
		gt.False(t, user.IsSystem())
	})
}

func TestKnowledgeStateHelpers(t *testing.T) {
	t.Run("IsActive returns true for active state", func(t *testing.T) {
		gt.True(t, types.KnowledgeStateActive.IsActive())
		gt.False(t, types.KnowledgeStateArchived.IsActive())
	})

	t.Run("IsArchived returns true for archived state", func(t *testing.T) {
		gt.True(t, types.KnowledgeStateArchived.IsArchived())
		gt.False(t, types.KnowledgeStateActive.IsArchived())
	})
}
