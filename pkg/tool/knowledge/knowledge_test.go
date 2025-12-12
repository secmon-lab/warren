package knowledge_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/tool/knowledge"
)

func TestKnowledgeToolListSlugs(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tool := knowledge.New(repo)

	topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", time.Now().UnixNano()))
	tool.SetTopic(topic)

	// Initially empty
	result, err := tool.Run(ctx, "knowledge_list", nil)
	gt.NoError(t, err)
	_, ok := result["slugs"]
	gt.True(t, ok)

	// Save some knowledges
	_, err = tool.Run(ctx, "knowledge_save", map[string]any{
		"slug":    "slug1",
		"name":    "Name 1",
		"content": "Content 1",
	})
	gt.NoError(t, err)

	_, err = tool.Run(ctx, "knowledge_save", map[string]any{
		"slug":    "slug2",
		"name":    "Name 2",
		"content": "Content 2",
	})
	gt.NoError(t, err)

	// List slugs
	result, err = tool.Run(ctx, "knowledge_list", nil)
	gt.NoError(t, err)

	slugsResult := result["slugs"]
	gt.NotEqual(t, slugsResult, nil)
}

func TestKnowledgeToolGetKnowledges(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tool := knowledge.New(repo)

	topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", time.Now().UnixNano()))
	tool.SetTopic(topic)

	// Save knowledge
	_, err := tool.Run(ctx, "knowledge_save", map[string]any{
		"slug":    "test-slug",
		"name":    "Test Name",
		"content": "Test content",
	})
	gt.NoError(t, err)

	// Get specific knowledge
	result, err := tool.Run(ctx, "knowledge_get", map[string]any{
		"slug": "test-slug",
	})
	gt.NoError(t, err)
	gt.Equal(t, result["found"], true)

	// Get all knowledges
	result, err = tool.Run(ctx, "knowledge_get", map[string]any{})
	gt.NoError(t, err)
	knowledges, ok := result["knowledges"]
	gt.True(t, ok)
	gt.NotEqual(t, knowledges, nil)
}

func TestKnowledgeToolSave(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tool := knowledge.New(repo)

	topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", time.Now().UnixNano()))
	tool.SetTopic(topic)

	result, err := tool.Run(ctx, "knowledge_save", map[string]any{
		"slug":    "test-slug",
		"name":    "Test Name",
		"content": "Test content",
	})
	gt.NoError(t, err)
	gt.Equal(t, result["success"], true)
	message, ok := result["message"].(string)
	gt.True(t, ok)
	gt.True(t, strings.Contains(message, "saved successfully"))
}

func TestKnowledgeToolArchive(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tool := knowledge.New(repo)

	topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", time.Now().UnixNano()))
	tool.SetTopic(topic)

	// Save knowledge
	_, err := tool.Run(ctx, "knowledge_save", map[string]any{
		"slug":    "test-slug",
		"name":    "Test Name",
		"content": "Test content",
	})
	gt.NoError(t, err)

	// Archive it
	result, err := tool.Run(ctx, "knowledge_archive", map[string]any{
		"slug": "test-slug",
	})
	gt.NoError(t, err)
	gt.Equal(t, result["success"], true)

	// Should not be retrievable
	result, err = tool.Run(ctx, "knowledge_get", map[string]any{
		"slug": "test-slug",
	})
	gt.NoError(t, err)
	gt.Equal(t, result["found"], false)
}

func TestKnowledgeToolValidation(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tool := knowledge.New(repo)

	topic := types.KnowledgeTopic("test-topic")
	tool.SetTopic(topic)

	t.Run("missing slug", func(t *testing.T) {
		_, err := tool.Run(ctx, "knowledge_save", map[string]any{
			"name":    "Test",
			"content": "Content",
		})
		gt.Error(t, err)
	})

	t.Run("missing name", func(t *testing.T) {
		_, err := tool.Run(ctx, "knowledge_save", map[string]any{
			"slug":    "test",
			"content": "Content",
		})
		gt.Error(t, err)
	})

	t.Run("missing content", func(t *testing.T) {
		_, err := tool.Run(ctx, "knowledge_save", map[string]any{
			"slug": "test",
			"name": "Test",
		})
		gt.Error(t, err)
	})
}

func TestKnowledgeToolPrompt(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tool := knowledge.New(repo)

	topic := types.KnowledgeTopic(fmt.Sprintf("test-topic-%d", time.Now().UnixNano()))
	tool.SetTopic(topic)

	// Prompt method now returns guidance to help LLM use knowledge tools correctly
	prompt, err := tool.Prompt(ctx)
	gt.NoError(t, err)
	gt.V(t, prompt).NotEqual("")
	// Verify key elements are present in the guidance
	if !strings.Contains(prompt, "CRITICAL") {
		t.Error("Prompt should contain CRITICAL instruction")
	}
	if !strings.Contains(prompt, "knowledge_save") {
		t.Error("Prompt should mention knowledge_save")
	}
	if !strings.Contains(prompt, topic.String()) {
		t.Error("Prompt should include current topic")
	}
}
