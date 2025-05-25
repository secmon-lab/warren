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
