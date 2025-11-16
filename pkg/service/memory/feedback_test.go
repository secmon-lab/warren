package memory_test

import (
	"context"
	"math"
	"testing"

	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	memoryRepo "github.com/secmon-lab/warren/pkg/repository/memory"
	memoryService "github.com/secmon-lab/warren/pkg/service/memory"
)

func TestCollectAndApplyFeedback_EmptyMemories(t *testing.T) {
	ctx := context.Background()
	repo := memoryRepo.New()
	llm := &mock.LLMClientMock{
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, input []string) ([][]float64, error) {
			embeddings := make([][]float64, len(input))
			for i := range input {
				vec := make([]float64, dimension)
				for j := 0; j < dimension; j++ {
					vec[j] = 0.1 * float64(i+j)
				}
				embeddings[i] = vec
			}
			return embeddings, nil
		},
	}
	svc := memoryService.New(llm, repo)

	// No error when no memories provided
	err := svc.CollectAndApplyFeedback(ctx, "test-agent", nil, "query", nil, nil, nil)
	gt.NoError(t, err)

	err = svc.CollectAndApplyFeedback(ctx, "test-agent", []*memory.AgentMemory{}, "query", nil, nil, nil)
	gt.NoError(t, err)
}

func TestEMAScoreUpdate(t *testing.T) {
	// Test EMA calculation with different alpha values
	testCases := []struct {
		name          string
		alpha         float64
		oldScore      float64
		feedbackScore float64
		expected      float64
	}{
		{
			name:          "default alpha (0.3)",
			alpha:         0.3,
			oldScore:      0.0,
			feedbackScore: 6.0,
			expected:      1.8, // 0.3 * 6.0 + 0.7 * 0.0 = 1.8
		},
		{
			name:          "high alpha (0.8) - more weight on new feedback",
			alpha:         0.8,
			oldScore:      2.0,
			feedbackScore: 8.0,
			expected:      6.8, // 0.8 * 8.0 + 0.2 * 2.0 = 6.8
		},
		{
			name:          "low alpha (0.1) - more weight on old score",
			alpha:         0.1,
			oldScore:      5.0,
			feedbackScore: -5.0,
			expected:      4.0, // 0.1 * -5.0 + 0.9 * 5.0 = 4.0
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.alpha*tc.feedbackScore + (1-tc.alpha)*tc.oldScore
			// Allow small floating point difference
			diff := math.Abs(result - tc.expected)
			if diff >= 0.001 {
				t.Errorf("expected ~%f, got %f", tc.expected, result)
			}
		})
	}
}
