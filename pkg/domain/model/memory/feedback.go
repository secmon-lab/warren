package memory

import "github.com/secmon-lab/warren/pkg/domain/types"

// MemoryFeedback represents evaluation of how useful a memory was
// This feedback is used to update the quality score of agent memories
// Based on three dimensions: Relevance, Support, and Impact
type MemoryFeedback struct {
	MemoryID types.AgentMemoryID

	// Relevance: Was the memory relevant to the task? (0-3)
	// 0 = Not relevant, 1 = Somewhat relevant, 2 = Relevant, 3 = Highly relevant
	Relevance int

	// Support: Did the memory help find the solution? (0-4)
	// 0 = No help, 1 = Minor help, 2 = Moderate help, 3 = Significant help, 4 = Critical help
	Support int

	// Impact: Did the memory contribute to the final result? (0-3)
	// 0 = No impact, 1 = Minor impact, 2 = Moderate impact, 3 = Major impact
	Impact int

	// Reasoning: LLM's explanation for the scores
	Reasoning string
}

// TotalScore returns the sum of all feedback dimensions (0-10)
func (f *MemoryFeedback) TotalScore() float64 {
	return float64(f.Relevance + f.Support + f.Impact)
}

// NormalizedScore returns score normalized to -10 to +10 range
// This normalization maps:
//   - 0 (worst) -> -10
//   - 5 (neutral) -> 0
//   - 10 (best) -> +10
func (f *MemoryFeedback) NormalizedScore() float64 {
	// Convert 0-10 scale to -10 to +10 scale
	return (f.TotalScore() - 5.0) * 2.0
}
