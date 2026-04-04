package knowledge_test

import (
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func validKnowledge() *knowledge.Knowledge {
	return &knowledge.Knowledge{
		ID:        types.NewKnowledgeID(),
		Category:  types.KnowledgeCategoryFact,
		Title:     "svchost.exe",
		Claim:     "Windows service host process.",
		Tags:      []types.KnowledgeTagID{types.NewKnowledgeTagID()},
		Author:    "user-1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestKnowledgeValidate(t *testing.T) {
	t.Run("valid knowledge", func(t *testing.T) {
		k := validKnowledge()
		gt.NoError(t, k.Validate())
	})

	t.Run("empty ID", func(t *testing.T) {
		k := validKnowledge()
		k.ID = ""
		gt.Error(t, k.Validate())
	})

	t.Run("invalid category", func(t *testing.T) {
		k := validKnowledge()
		k.Category = "invalid"
		gt.Error(t, k.Validate())
	})

	t.Run("empty title", func(t *testing.T) {
		k := validKnowledge()
		k.Title = ""
		gt.Error(t, k.Validate())
	})

	t.Run("empty claim", func(t *testing.T) {
		k := validKnowledge()
		k.Claim = ""
		gt.Error(t, k.Validate())
	})

	t.Run("empty author", func(t *testing.T) {
		k := validKnowledge()
		k.Author = ""
		gt.Error(t, k.Validate())
	})

	t.Run("zero created_at", func(t *testing.T) {
		k := validKnowledge()
		k.CreatedAt = time.Time{}
		gt.Error(t, k.Validate())
	})

	t.Run("zero updated_at", func(t *testing.T) {
		k := validKnowledge()
		k.UpdatedAt = time.Time{}
		gt.Error(t, k.Validate())
	})

	t.Run("invalid tag ID", func(t *testing.T) {
		k := validKnowledge()
		k.Tags = []types.KnowledgeTagID{""}
		gt.Error(t, k.Validate())
	})

	t.Run("fact category", func(t *testing.T) {
		k := validKnowledge()
		k.Category = types.KnowledgeCategoryFact
		gt.NoError(t, k.Validate())
	})

	t.Run("technique category", func(t *testing.T) {
		k := validKnowledge()
		k.Category = types.KnowledgeCategoryTechnique
		gt.NoError(t, k.Validate())
	})
}

func TestKnowledgeLogValidate(t *testing.T) {
	validLog := func() *knowledge.KnowledgeLog {
		return &knowledge.KnowledgeLog{
			ID:          types.NewKnowledgeLogID(),
			KnowledgeID: types.NewKnowledgeID(),
			Title:       "svchost.exe",
			Claim:       "Windows service host process.",
			Author:      "user-1",
			Message:     "Initial creation",
			CreatedAt:   time.Now(),
		}
	}

	t.Run("valid log", func(t *testing.T) {
		l := validLog()
		gt.NoError(t, l.Validate())
	})

	t.Run("empty ID", func(t *testing.T) {
		l := validLog()
		l.ID = ""
		gt.Error(t, l.Validate())
	})

	t.Run("empty knowledge ID", func(t *testing.T) {
		l := validLog()
		l.KnowledgeID = ""
		gt.Error(t, l.Validate())
	})

	t.Run("empty title", func(t *testing.T) {
		l := validLog()
		l.Title = ""
		gt.Error(t, l.Validate())
	})

	t.Run("empty claim", func(t *testing.T) {
		l := validLog()
		l.Claim = ""
		gt.Error(t, l.Validate())
	})

	t.Run("empty message", func(t *testing.T) {
		l := validLog()
		l.Message = ""
		gt.Error(t, l.Validate())
	})

	t.Run("with ticket ID", func(t *testing.T) {
		l := validLog()
		l.TicketID = "ticket-123"
		gt.NoError(t, l.Validate())
	})

	t.Run("without ticket ID", func(t *testing.T) {
		l := validLog()
		l.TicketID = ""
		gt.NoError(t, l.Validate())
	})
}

func TestKnowledgeTagValidate(t *testing.T) {
	validTag := func() *knowledge.KnowledgeTag {
		return &knowledge.KnowledgeTag{
			ID:        types.NewKnowledgeTagID(),
			Name:      "crowdstrike",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	t.Run("valid tag", func(t *testing.T) {
		tag := validTag()
		gt.NoError(t, tag.Validate())
	})

	t.Run("empty ID", func(t *testing.T) {
		tag := validTag()
		tag.ID = ""
		gt.Error(t, tag.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		tag := validTag()
		tag.Name = ""
		gt.Error(t, tag.Validate())
	})

	t.Run("with description", func(t *testing.T) {
		tag := validTag()
		tag.Description = "CrowdStrike Falcon EDR"
		gt.NoError(t, tag.Validate())
	})

	t.Run("zero created_at", func(t *testing.T) {
		tag := validTag()
		tag.CreatedAt = time.Time{}
		gt.Error(t, tag.Validate())
	})

	t.Run("zero updated_at", func(t *testing.T) {
		tag := validTag()
		tag.UpdatedAt = time.Time{}
		gt.Error(t, tag.Validate())
	})
}

func TestKnowledgeCategoryValidate(t *testing.T) {
	t.Run("fact is valid", func(t *testing.T) {
		gt.NoError(t, types.KnowledgeCategoryFact.Validate())
	})

	t.Run("technique is valid", func(t *testing.T) {
		gt.NoError(t, types.KnowledgeCategoryTechnique.Validate())
	})

	t.Run("empty is invalid", func(t *testing.T) {
		gt.Error(t, types.KnowledgeCategory("").Validate())
	})

	t.Run("unknown is invalid", func(t *testing.T) {
		gt.Error(t, types.KnowledgeCategory("other").Validate())
	})
}
