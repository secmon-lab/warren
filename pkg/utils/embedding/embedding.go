package embedding

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
)

const (
	EmbeddingDimension = 256
)

func Generate(ctx context.Context, client gollem.LLMClient, data any) (firestore.Vector32, error) {
	value, ok := data.(string)
	if !ok {
		raw, err := json.Marshal(data)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to marshal data")
		}
		value = string(raw)
	}

	embedding, err := client.GenerateEmbedding(ctx, EmbeddingDimension, []string{value})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate embedding")
	}

	vector32 := make(firestore.Vector32, len(embedding[0]))
	for i, v := range embedding[0] { // nocheck: S1001
		vector32[i] = float32(v)
	}
	return vector32, nil
}

func Average(embeddings []firestore.Vector32) firestore.Vector32 {
	if len(embeddings) == 0 {
		return firestore.Vector32{}
	}

	// Get dimension from first embedding
	dim := len(embeddings[0])
	sum := make([]float32, dim)

	// Sum up all embeddings
	for _, embedding := range embeddings {
		for i := 0; i < dim; i++ {
			sum[i] += embedding[i]
		}
	}

	// Calculate average
	avg := make([]float32, dim)
	n := float32(len(embeddings))
	for i := range avg {
		avg[i] = sum[i] / n
	}

	return avg
}

// WeightedAverage calculates the weighted average of embeddings.
// embeddings and weights must have the same length.
// weights should sum to 1.0 for proper normalization.
func WeightedAverage(embeddings []firestore.Vector32, weights []float32) (firestore.Vector32, error) {
	if len(embeddings) == 0 {
		return firestore.Vector32{}, goerr.New("embeddings cannot be empty")
	}

	if len(embeddings) != len(weights) {
		return firestore.Vector32{}, goerr.New("embeddings and weights must have the same length",
			goerr.V("embeddings_len", len(embeddings)),
			goerr.V("weights_len", len(weights)),
		)
	}

	// Get dimension from first embedding
	dim := len(embeddings[0])
	if dim == 0 {
		return firestore.Vector32{}, goerr.New("embeddings cannot have zero dimensions")
	}

	// Validate that all embeddings have the same dimension
	for i, embedding := range embeddings {
		if len(embedding) != dim {
			return firestore.Vector32{}, goerr.New("all embeddings must have the same dimension",
				goerr.V("expected_dim", dim),
				goerr.V("embedding_index", i),
				goerr.V("actual_dim", len(embedding)),
			)
		}
	}

	weightedSum := make([]float32, dim)

	// Calculate weighted sum
	for i, embedding := range embeddings {
		weight := weights[i]
		for j := 0; j < dim; j++ {
			weightedSum[j] += embedding[j] * weight
		}
	}

	return weightedSum, nil
}
