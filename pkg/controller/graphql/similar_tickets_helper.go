package graphql

import "math"

const (
	// Constants for similar tickets pagination
	defaultSimilarTicketsLimit  = 5
	maxSimilarTicketsCandidates = 1000 // Fixed large number of candidates to fetch

	// Constants for comments pagination
	defaultCommentsLimit = 20
)

func cosineSimilarity(a, b []float32) float32 {
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
