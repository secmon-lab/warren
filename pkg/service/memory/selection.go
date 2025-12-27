package memory

import (
	"math"
	"sort"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/memory"
)

// SelectionAlgorithm ranks and filters memories for prompt selection
// Takes candidates retrieved from vector search, query embedding, and desired limit
// Returns top-ranked memories based on multi-dimensional scoring
type SelectionAlgorithm func(
	candidates []*memory.AgentMemory,
	queryEmbedding []float32,
	limit int,
) []*memory.AgentMemory

// DefaultSelectionAlgorithm is the default implementation of selection algorithm
// Parameters are defined as constants within the function
var DefaultSelectionAlgorithm SelectionAlgorithm = func(
	candidates []*memory.AgentMemory,
	queryEmbedding []float32,
	limit int,
) []*memory.AgentMemory {
	// Algorithm parameters (const within function)
	const (
		minQualityScore     = -5.0 // Minimum quality score threshold
		similarityWeight    = 0.5  // Weight for similarity score
		qualityWeight       = 0.3  // Weight for quality score
		recencyWeight       = 0.2  // Weight for recency score
		recencyHalfLifeDays = 30.0 // Half-life for recency decay in days
		scoreMin            = -10.0
		scoreMax            = 10.0
	)

	now := time.Now()
	type rankedMemory struct {
		memory     *memory.AgentMemory
		finalScore float64
	}
	var ranked []rankedMemory

	// Filter and rank
	for _, mem := range candidates {
		if mem.Score < minQualityScore {
			continue // Filter out low-quality memories
		}

		similarity := calculateCosineSimilarity(queryEmbedding, mem.QueryEmbedding)
		quality := (mem.Score - scoreMin) / (scoreMax - scoreMin)
		recency := calculateRecencyScore(mem.LastUsedAt, now, recencyHalfLifeDays)

		finalScore := similarityWeight*similarity +
			qualityWeight*quality +
			recencyWeight*recency

		ranked = append(ranked, rankedMemory{
			memory:     mem,
			finalScore: finalScore,
		})
	}

	// Sort by finalScore descending
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].finalScore > ranked[j].finalScore
	})

	// Take top N
	resultSize := limit
	if len(ranked) < resultSize {
		resultSize = len(ranked)
	}

	results := make([]*memory.AgentMemory, resultSize)
	for i := 0; i < resultSize; i++ {
		results[i] = ranked[i].memory
	}

	return results
}

// calculateCosineSimilarity computes the cosine similarity between two vectors
// Returns a value between 0.0 (orthogonal/unrelated) and 1.0 (identical direction)
func calculateCosineSimilarity(v1, v2 []float32) float64 {
	if len(v1) != len(v2) || len(v1) == 0 {
		return 0.0
	}

	var dotProduct, norm1, norm2 float64
	for i := range v1 {
		dotProduct += float64(v1[i]) * float64(v2[i])
		norm1 += float64(v1[i]) * float64(v1[i])
		norm2 += float64(v2[i]) * float64(v2[i])
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// calculateRecencyScore computes a recency score based on exponential decay
// Returns:
// - 0.0 if lastUsedAt is zero (never used)
// - 1.0 if used very recently
// - 0.5 after halfLifeDays
// - Exponential decay over time: 0.5^(daysSinceUsed / halfLifeDays)
func calculateRecencyScore(lastUsedAt time.Time, now time.Time, halfLifeDays float64) float64 {
	if lastUsedAt.IsZero() {
		return 0.0
	}

	daysSince := now.Sub(lastUsedAt).Hours() / 24.0
	return math.Pow(0.5, daysSince/halfLifeDays)
}
