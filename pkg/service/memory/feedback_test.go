package memory_test

import (
	"context"
	"math"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
	memoryRepo "github.com/secmon-lab/warren/pkg/repository/memory"
	memoryService "github.com/secmon-lab/warren/pkg/service/memory"
)

func TestCollectAndApplyFeedback_EmptyMemories(t *testing.T) {
	ctx := context.Background()
	repo := memoryRepo.New()
	llm := &mockLLMClient{}
	svc := memoryService.New(llm, repo)

	// No error when no memories provided
	err := svc.CollectAndApplyFeedback(ctx, "test-agent", nil, "query", nil, nil, nil)
	gt.NoError(t, err)

	err = svc.CollectAndApplyFeedback(ctx, "test-agent", []*memory.AgentMemory{}, "query", nil, nil, nil)
	gt.NoError(t, err)
}

func TestMemoryFeedback_ScoreCalculation(t *testing.T) {
	testCases := []struct {
		name      string
		relevance int
		support   int
		impact    int
		expected  float64 // normalized score
	}{
		{
			name:      "all zeros -> -10",
			relevance: 0,
			support:   0,
			impact:    0,
			expected:  -10.0,
		},
		{
			name:      "neutral (5) -> 0",
			relevance: 2,
			support:   2,
			impact:    1,
			expected:  0.0,
		},
		{
			name:      "max score (10) -> +10",
			relevance: 3,
			support:   4,
			impact:    3,
			expected:  10.0,
		},
		{
			name:      "above neutral",
			relevance: 2,
			support:   3,
			impact:    2,
			expected:  4.0, // (7 - 5) * 2 = 4
		},
		{
			name:      "below neutral",
			relevance: 1,
			support:   1,
			impact:    1,
			expected:  -4.0, // (3 - 5) * 2 = -4
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			feedback := &memory.MemoryFeedback{
				MemoryID:  types.NewAgentMemoryID(),
				Relevance: tc.relevance,
				Support:   tc.support,
				Impact:    tc.impact,
			}

			gt.Equal(t, feedback.NormalizedScore(), tc.expected)
		})
	}
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

func TestRecencyScoreCalculation(t *testing.T) {
	// Test recency score decay with different half-lives
	testCases := []struct {
		name         string
		halfLifeDays float64
		daysAgo      float64
		expected     float64 // approximate
	}{
		{
			name:         "just used (0 days)",
			halfLifeDays: 30,
			daysAgo:      0,
			expected:     1.0,
		},
		{
			name:         "half-life (30 days)",
			halfLifeDays: 30,
			daysAgo:      30,
			expected:     0.5,
		},
		{
			name:         "double half-life (60 days)",
			halfLifeDays: 30,
			daysAgo:      60,
			expected:     0.25,
		},
		{
			name:         "longer half-life (60 days), 30 days ago",
			halfLifeDays: 60,
			daysAgo:      30,
			expected:     0.707, // 2^(-30/60) ≈ 0.707
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Calculate: 0.5^(daysAgo / halfLife)
			result := math.Pow(0.5, tc.daysAgo/tc.halfLifeDays)

			// Allow small floating point difference
			diff := math.Abs(result - tc.expected)
			if diff >= 0.01 {
				t.Errorf("expected ~%f, got %f", tc.expected, result)
			}
		})
	}
}
