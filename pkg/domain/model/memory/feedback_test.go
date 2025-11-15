package memory_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func TestMemoryFeedback_TotalScore(t *testing.T) {
	feedback := &memory.MemoryFeedback{
		MemoryID:  types.NewAgentMemoryID(),
		Relevance: 2,
		Support:   3,
		Impact:    1,
		Reasoning: "Test reasoning",
	}

	gt.Equal(t, feedback.TotalScore(), 6.0)
}

func TestMemoryFeedback_NormalizedScore(t *testing.T) {
	testCases := []struct {
		name      string
		relevance int
		support   int
		impact    int
		expected  float64
	}{
		{
			name:      "minimum score (0-10 scale)",
			relevance: 0,
			support:   0,
			impact:    0,
			expected:  -10.0, // (0 - 5) * 2 = -10
		},
		{
			name:      "neutral score (0-10 scale)",
			relevance: 2,
			support:   2,
			impact:    1,
			expected:  0.0, // (5 - 5) * 2 = 0
		},
		{
			name:      "maximum score (0-10 scale)",
			relevance: 3,
			support:   4,
			impact:    3,
			expected:  10.0, // (10 - 5) * 2 = 10
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
