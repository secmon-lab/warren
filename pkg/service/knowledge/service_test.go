package knowledge_test

import (
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	knowledgeModel "github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/knowledge"
)

func TestGetKnowledges(t *testing.T) {
	repo := repository.NewMemory()
	svc := knowledge.New(repo, nil)

	now := time.Now()

	// Create test knowledges
	k1 := &knowledgeModel.Knowledge{
		ID:        types.NewKnowledgeID(),
		Category:  types.KnowledgeCategoryFact,
		Title:     "Test Knowledge 1",
		Claim:     "Test claim 1",
		Tags:      []types.KnowledgeTagID{},
		Author:    types.SystemUserID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	k2 := &knowledgeModel.Knowledge{
		ID:        types.NewKnowledgeID(),
		Category:  types.KnowledgeCategoryFact,
		Title:     "Test Knowledge 2",
		Claim:     "Test claim 2",
		Tags:      []types.KnowledgeTagID{},
		Author:    types.SystemUserID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	k3 := &knowledgeModel.Knowledge{
		ID:        types.NewKnowledgeID(),
		Category:  types.KnowledgeCategoryFact,
		Title:     "Test Knowledge 3",
		Claim:     "Test claim 3",
		Tags:      []types.KnowledgeTagID{},
		Author:    types.SystemUserID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	gt.NoError(t, repo.PutKnowledge(t.Context(), k1))
	gt.NoError(t, repo.PutKnowledge(t.Context(), k2))
	gt.NoError(t, repo.PutKnowledge(t.Context(), k3))

	t.Run("retrieve multiple knowledges", func(t *testing.T) {
		ids := []types.KnowledgeID{k1.ID, k2.ID, k3.ID}
		results, err := svc.GetKnowledges(t.Context(), ids)
		gt.NoError(t, err)
		gt.A(t, results).Length(3)

		idMap := make(map[types.KnowledgeID]bool)
		for _, k := range results {
			idMap[k.ID] = true
		}
		gt.True(t, idMap[k1.ID])
		gt.True(t, idMap[k2.ID])
		gt.True(t, idMap[k3.ID])
	})

	t.Run("skip non-existent IDs", func(t *testing.T) {
		nonExistentID := types.NewKnowledgeID()
		ids := []types.KnowledgeID{k1.ID, nonExistentID, k2.ID}
		results, err := svc.GetKnowledges(t.Context(), ids)
		gt.NoError(t, err)
		gt.A(t, results).Length(2)

		idMap := make(map[types.KnowledgeID]bool)
		for _, k := range results {
			idMap[k.ID] = true
		}
		gt.True(t, idMap[k1.ID])
		gt.True(t, idMap[k2.ID])
		gt.True(t, !idMap[nonExistentID])
	})

	t.Run("empty IDs returns empty slice", func(t *testing.T) {
		results, err := svc.GetKnowledges(t.Context(), []types.KnowledgeID{})
		gt.NoError(t, err)
		gt.A(t, results).Length(0)
	})
}
