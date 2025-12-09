package types_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func TestKnowledgeTopicValidate(t *testing.T) {
	t.Run("valid topic", func(t *testing.T) {
		topic := types.KnowledgeTopic("valid-topic")
		gt.NoError(t, topic.Validate())
	})

	t.Run("empty topic", func(t *testing.T) {
		topic := types.KnowledgeTopic("")
		gt.Error(t, topic.Validate())
	})

	t.Run("topic with forward slash", func(t *testing.T) {
		topic := types.KnowledgeTopic("invalid/topic")
		err := topic.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("forbidden character '/'")
	})

	t.Run("topic with backslash", func(t *testing.T) {
		topic := types.KnowledgeTopic("invalid\\topic")
		err := topic.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("forbidden character '\\'")
	})

	t.Run("topic starting with period", func(t *testing.T) {
		topic := types.KnowledgeTopic(".invalid")
		err := topic.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("cannot start with '.'")
	})

	t.Run("topic ending with period", func(t *testing.T) {
		topic := types.KnowledgeTopic("invalid.")
		err := topic.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("cannot end with '.'")
	})

	t.Run("topic with double underscores", func(t *testing.T) {
		topic := types.KnowledgeTopic("invalid__topic")
		err := topic.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("forbidden sequence '__'")
	})
}

func TestKnowledgeSlugValidate(t *testing.T) {
	t.Run("valid slug", func(t *testing.T) {
		slug := types.KnowledgeSlug("valid-slug")
		gt.NoError(t, slug.Validate())
	})

	t.Run("empty slug", func(t *testing.T) {
		slug := types.KnowledgeSlug("")
		gt.Error(t, slug.Validate())
	})

	t.Run("slug with forward slash", func(t *testing.T) {
		slug := types.KnowledgeSlug("invalid/slug")
		err := slug.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("forbidden character '/'")
	})

	t.Run("slug with backslash", func(t *testing.T) {
		slug := types.KnowledgeSlug("invalid\\slug")
		err := slug.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("forbidden character '\\'")
	})

	t.Run("slug starting with period", func(t *testing.T) {
		slug := types.KnowledgeSlug(".invalid")
		err := slug.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("cannot start with '.'")
	})

	t.Run("slug ending with period", func(t *testing.T) {
		slug := types.KnowledgeSlug("invalid.")
		err := slug.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("cannot end with '.'")
	})

	t.Run("slug with double underscores", func(t *testing.T) {
		slug := types.KnowledgeSlug("invalid__slug")
		err := slug.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("forbidden sequence '__'")
	})
}
