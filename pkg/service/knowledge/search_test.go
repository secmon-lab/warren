package knowledge

import (
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gt"
	knowledgeModel "github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func makeTestEmbedding(vals ...float32) firestore.Vector32 {
	v := make(firestore.Vector32, 256)
	for i, val := range vals {
		if i < 256 {
			v[i] = val
		}
	}
	return v
}

func TestHybridSearch(t *testing.T) {
	now := time.Now()
	docs := []*knowledgeModel.Knowledge{
		{
			ID:        types.KnowledgeID("k1"),
			Category:  types.KnowledgeCategoryFact,
			Title:     "svchost.exe",
			Claim:     "Windows service host process. False positive with CrowdStrike.",
			Embedding: makeTestEmbedding(1.0, 0.0, 0.0),
			Author:    "user-1",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        types.KnowledgeID("k2"),
			Category:  types.KnowledgeCategoryFact,
			Title:     "CrowdStrike DLL injection",
			Claim:     "Known false positive pattern in CrowdStrike Falcon EDR.",
			Embedding: makeTestEmbedding(0.9, 0.1, 0.0),
			Author:    "user-1",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        types.KnowledgeID("k3"),
			Category:  types.KnowledgeCategoryFact,
			Title:     "server-analytics-01",
			Claim:     "JupyterHub environment for data analysis team.",
			Embedding: makeTestEmbedding(0.0, 0.0, 1.0),
			Author:    "user-1",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	t.Run("returns results sorted by relevance", func(t *testing.T) {
		// Query embedding close to k1 and k2
		queryEmb := makeTestEmbedding(0.95, 0.05, 0.0)
		results := hybridSearch(docs, queryEmb, "svchost.exe crowdstrike", 10)
		gt.A(t, results).Length(3)
		// k1 or k2 should be ranked higher than k3
		gt.V(t, string(results[len(results)-1].ID)).Equal("k3")
	})

	t.Run("respects limit", func(t *testing.T) {
		queryEmb := makeTestEmbedding(1.0, 0.0, 0.0)
		results := hybridSearch(docs, queryEmb, "svchost", 1)
		gt.A(t, results).Length(1)
	})

	t.Run("empty docs returns nil", func(t *testing.T) {
		results := hybridSearch(nil, makeTestEmbedding(1.0), "test", 10)
		gt.True(t, results == nil)
	})
}

func TestCosineSimilarity(t *testing.T) {
	t.Run("identical vectors", func(t *testing.T) {
		a := makeTestEmbedding(1.0, 0.0, 0.0)
		sim := cosineSimilarity(a, a)
		gt.True(t, sim > 0.99)
	})

	t.Run("orthogonal vectors", func(t *testing.T) {
		a := makeTestEmbedding(1.0, 0.0, 0.0)
		b := makeTestEmbedding(0.0, 1.0, 0.0)
		sim := cosineSimilarity(a, b)
		gt.True(t, sim < 0.01)
	})

	t.Run("empty vectors", func(t *testing.T) {
		sim := cosineSimilarity(nil, nil)
		gt.V(t, sim).Equal(0.0)
	})
}
