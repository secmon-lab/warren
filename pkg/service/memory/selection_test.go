package memory_test

import (
	"math"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
	memsvc "github.com/secmon-lab/warren/pkg/service/memory"
)

// TestCalculateCosineSimilarity tests the cosine similarity calculation
func TestCalculateCosineSimilarity(t *testing.T) {
	t.Run("identical vectors", func(t *testing.T) {
		v1 := []float32{1.0, 2.0, 3.0}
		v2 := []float32{1.0, 2.0, 3.0}
		sim := memsvc.CalculateCosineSimilarity(v1, v2)
		gt.V(t, sim).Equal(1.0)
	})

	t.Run("orthogonal vectors", func(t *testing.T) {
		v1 := []float32{1.0, 0.0, 0.0}
		v2 := []float32{0.0, 1.0, 0.0}
		sim := memsvc.CalculateCosineSimilarity(v1, v2)
		gt.V(t, sim).Equal(0.0)
	})

	t.Run("opposite vectors", func(t *testing.T) {
		v1 := []float32{1.0, 2.0, 3.0}
		v2 := []float32{-1.0, -2.0, -3.0}
		sim := memsvc.CalculateCosineSimilarity(v1, v2)
		gt.V(t, sim).Equal(-1.0)
	})

	t.Run("similar vectors", func(t *testing.T) {
		v1 := []float32{1.0, 0.0, 0.0}
		v2 := []float32{1.0, 1.0, 0.0}
		sim := memsvc.CalculateCosineSimilarity(v1, v2)
		expected := 1.0 / math.Sqrt(2.0) // cos(45°) ≈ 0.707
		gt.True(t, math.Abs(sim-expected) < 0.001)
	})

	t.Run("empty vectors", func(t *testing.T) {
		v1 := []float32{}
		v2 := []float32{}
		sim := memsvc.CalculateCosineSimilarity(v1, v2)
		gt.V(t, sim).Equal(0.0)
	})

	t.Run("different length vectors", func(t *testing.T) {
		v1 := []float32{1.0, 2.0}
		v2 := []float32{1.0, 2.0, 3.0}
		sim := memsvc.CalculateCosineSimilarity(v1, v2)
		gt.V(t, sim).Equal(0.0)
	})

	t.Run("zero vector", func(t *testing.T) {
		v1 := []float32{0.0, 0.0, 0.0}
		v2 := []float32{1.0, 2.0, 3.0}
		sim := memsvc.CalculateCosineSimilarity(v1, v2)
		gt.V(t, sim).Equal(0.0)
	})
}

// TestCalculateRecencyScore tests the recency score calculation
func TestCalculateRecencyScore(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	halfLifeDays := 30.0

	t.Run("never used (zero time)", func(t *testing.T) {
		lastUsedAt := time.Time{}
		score := memsvc.CalculateRecencyScore(lastUsedAt, now, halfLifeDays)
		gt.V(t, score).Equal(0.0)
	})

	t.Run("used just now", func(t *testing.T) {
		lastUsedAt := now
		score := memsvc.CalculateRecencyScore(lastUsedAt, now, halfLifeDays)
		gt.V(t, score).Equal(1.0)
	})

	t.Run("used 30 days ago (half-life)", func(t *testing.T) {
		lastUsedAt := now.AddDate(0, 0, -30)
		score := memsvc.CalculateRecencyScore(lastUsedAt, now, halfLifeDays)
		gt.V(t, score).Equal(0.5)
	})

	t.Run("used 60 days ago (2x half-life)", func(t *testing.T) {
		lastUsedAt := now.AddDate(0, 0, -60)
		score := memsvc.CalculateRecencyScore(lastUsedAt, now, halfLifeDays)
		gt.V(t, score).Equal(0.25)
	})

	t.Run("used 90 days ago (3x half-life)", func(t *testing.T) {
		lastUsedAt := now.AddDate(0, 0, -90)
		score := memsvc.CalculateRecencyScore(lastUsedAt, now, halfLifeDays)
		gt.V(t, score).Equal(0.125)
	})

	t.Run("used 15 days ago", func(t *testing.T) {
		lastUsedAt := now.AddDate(0, 0, -15)
		score := memsvc.CalculateRecencyScore(lastUsedAt, now, halfLifeDays)
		expected := math.Pow(0.5, 15.0/30.0) // 0.5^0.5 ≈ 0.707
		gt.True(t, math.Abs(score-expected) < 0.001)
	})
}

// TestDefaultSelectionAlgorithm tests the full selection algorithm
func TestDefaultSelectionAlgorithm(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

	// Create test query embedding
	queryEmbedding := []float32{1.0, 0.0, 0.0}

	t.Run("filters out low quality memories", func(t *testing.T) {
		candidates := []*memory.AgentMemory{
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        "test",
				Query:          "query1",
				QueryEmbedding: firestore.Vector32{1.0, 0.0, 0.0}, // Perfect similarity
				Claim:          "claim1",
				Score:          -6.0, // Below minQualityScore (-5.0)
				CreatedAt:      now.AddDate(0, 0, -10),
				LastUsedAt:     now.AddDate(0, 0, -5),
			},
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        "test",
				Query:          "query2",
				QueryEmbedding: firestore.Vector32{1.0, 0.0, 0.0},
				Claim:          "claim2",
				Score:          -4.0, // Above minQualityScore
				CreatedAt:      now.AddDate(0, 0, -10),
				LastUsedAt:     now.AddDate(0, 0, -5),
			},
		}

		results := memsvc.DefaultSelectionAlgorithm(candidates, queryEmbedding, 10)

		// Only the second memory should pass the quality filter
		gt.A(t, results).Length(1)
		gt.V(t, results[0].Score).Equal(-4.0)
	})

	t.Run("ranks by weighted score (similarity + quality + recency)", func(t *testing.T) {
		candidates := []*memory.AgentMemory{
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        "test",
				Query:          "memory A",
				QueryEmbedding: firestore.Vector32{0.92, 0.39, 0.0}, // High similarity
				Claim:          "claim A",
				Score:          4.0, // Quality: (4.0 - (-10.0)) / 20.0 = 0.7
				CreatedAt:      now.AddDate(0, 0, -20),
				LastUsedAt:     now.AddDate(0, 0, -10), // Recency: 0.5^(10/30) ≈ 0.81
			},
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        "test",
				Query:          "memory B",
				QueryEmbedding: firestore.Vector32{0.65, 0.76, 0.0}, // Medium similarity
				Claim:          "claim B",
				Score:          8.0, // Quality: (8.0 - (-10.0)) / 20.0 = 0.9
				CreatedAt:      now.AddDate(0, 0, -70),
				LastUsedAt:     now.AddDate(0, 0, -60), // Recency: 0.5^(60/30) = 0.25
			},
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        "test",
				Query:          "memory C",
				QueryEmbedding: firestore.Vector32{0.45, 0.89, 0.0}, // Lower similarity
				Claim:          "claim C",
				Score:          6.0, // Quality: (6.0 - (-10.0)) / 20.0 = 0.8
				CreatedAt:      now.AddDate(0, 0, -10),
				LastUsedAt:     now.AddDate(0, 0, -5), // Recency: 0.5^(5/30) ≈ 0.89
			},
		}

		results := memsvc.DefaultSelectionAlgorithm(candidates, queryEmbedding, 10)

		// All should pass quality filter
		gt.A(t, results).Length(3)

		// Memory A should rank highest due to balanced high scores
		// Expected final scores:
		// A: 0.5*0.92 + 0.3*0.7 + 0.2*0.81 = 0.46 + 0.21 + 0.162 = 0.832
		// B: 0.5*0.65 + 0.3*0.9 + 0.2*0.25 = 0.325 + 0.27 + 0.05 = 0.645
		// C: 0.5*0.45 + 0.3*0.8 + 0.2*0.89 = 0.225 + 0.24 + 0.178 = 0.643
		gt.V(t, results[0].Claim).Equal("claim A")
		gt.V(t, results[1].Claim).Equal("claim B")
		gt.V(t, results[2].Claim).Equal("claim C")
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		candidates := []*memory.AgentMemory{
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        "test",
				Query:          "query1",
				QueryEmbedding: firestore.Vector32{1.0, 0.0, 0.0},
				Claim:          "claim1",
				Score:          5.0,
				CreatedAt:      now.AddDate(0, 0, -10),
				LastUsedAt:     now.AddDate(0, 0, -5),
			},
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        "test",
				Query:          "query2",
				QueryEmbedding: firestore.Vector32{1.0, 0.0, 0.0},
				Claim:          "claim2",
				Score:          4.0,
				CreatedAt:      now.AddDate(0, 0, -10),
				LastUsedAt:     now.AddDate(0, 0, -5),
			},
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        "test",
				Query:          "query3",
				QueryEmbedding: firestore.Vector32{1.0, 0.0, 0.0},
				Claim:          "claim3",
				Score:          3.0,
				CreatedAt:      now.AddDate(0, 0, -10),
				LastUsedAt:     now.AddDate(0, 0, -5),
			},
		}

		results := memsvc.DefaultSelectionAlgorithm(candidates, queryEmbedding, 2)

		// Should return only top 2
		gt.A(t, results).Length(2)
		gt.V(t, results[0].Score).Equal(5.0)
		gt.V(t, results[1].Score).Equal(4.0)
	})

	t.Run("handles empty candidates", func(t *testing.T) {
		candidates := []*memory.AgentMemory{}
		results := memsvc.DefaultSelectionAlgorithm(candidates, queryEmbedding, 10)
		gt.A(t, results).Length(0)
	})

	t.Run("handles limit greater than candidates", func(t *testing.T) {
		candidates := []*memory.AgentMemory{
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        "test",
				Query:          "query1",
				QueryEmbedding: firestore.Vector32{1.0, 0.0, 0.0},
				Claim:          "claim1",
				Score:          5.0,
				CreatedAt:      now.AddDate(0, 0, -10),
				LastUsedAt:     now.AddDate(0, 0, -5),
			},
		}

		results := memsvc.DefaultSelectionAlgorithm(candidates, queryEmbedding, 10)

		// Should return all candidates
		gt.A(t, results).Length(1)
	})

	t.Run("never used memory gets zero recency", func(t *testing.T) {
		candidates := []*memory.AgentMemory{
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        "test",
				Query:          "query1",
				QueryEmbedding: firestore.Vector32{1.0, 0.0, 0.0},
				Claim:          "claim1",
				Score:          5.0,
				CreatedAt:      now.AddDate(0, 0, -10),
				LastUsedAt:     time.Time{}, // Never used
			},
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        "test",
				Query:          "query2",
				QueryEmbedding: firestore.Vector32{1.0, 0.0, 0.0},
				Claim:          "claim2",
				Score:          5.0, // Same quality
				CreatedAt:      now.AddDate(0, 0, -10),
				LastUsedAt:     now.AddDate(0, 0, -5), // Recently used
			},
		}

		results := memsvc.DefaultSelectionAlgorithm(candidates, queryEmbedding, 10)

		// Recently used should rank higher due to recency bonus
		gt.A(t, results).Length(2)
		gt.V(t, results[0].Claim).Equal("claim2")
		gt.V(t, results[1].Claim).Equal("claim1")
	})

	t.Run("quality normalization boundary values", func(t *testing.T) {
		candidates := []*memory.AgentMemory{
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        "test",
				Query:          "min",
				QueryEmbedding: firestore.Vector32{1.0, 0.0, 0.0},
				Claim:          "min score",
				Score:          -10.0, // Quality: 0.0
				CreatedAt:      now,
				LastUsedAt:     now,
			},
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        "test",
				Query:          "zero",
				QueryEmbedding: firestore.Vector32{1.0, 0.0, 0.0},
				Claim:          "zero score",
				Score:          0.0, // Quality: 0.5
				CreatedAt:      now,
				LastUsedAt:     now,
			},
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        "test",
				Query:          "max",
				QueryEmbedding: firestore.Vector32{1.0, 0.0, 0.0},
				Claim:          "max score",
				Score:          10.0, // Quality: 1.0
				CreatedAt:      now,
				LastUsedAt:     now,
			},
		}

		results := memsvc.DefaultSelectionAlgorithm(candidates, queryEmbedding, 10)

		// Min score (-10.0) is below minQualityScore (-5.0), so filtered out
		gt.A(t, results).Length(2)
		// Max score should rank first, then zero
		gt.V(t, results[0].Claim).Equal("max score")
		gt.V(t, results[1].Claim).Equal("zero score")
	})
}
