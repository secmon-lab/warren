package memory

import (
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// ScoringAlgorithm updates memory scores based on reflection feedback
// Pure function with no side effects - receives memories and reflection, returns score updates
type ScoringAlgorithm func(
	memories map[types.AgentMemoryID]*memory.AgentMemory,
	reflection *memory.Reflection,
) map[types.AgentMemoryID]float64

// DefaultScoringAlgorithm is the default implementation of scoring algorithm using EMA
// Algorithm parameters are defined as constants within the function
var DefaultScoringAlgorithm ScoringAlgorithm = func(
	memories map[types.AgentMemoryID]*memory.AgentMemory,
	reflection *memory.Reflection,
) map[types.AgentMemoryID]float64 {
	// Algorithm parameters (const within function)
	const (
		helpfulScoreDelta = 2.0   // positive feedback for helpful memories
		harmfulScoreDelta = -3.0  // negative feedback for harmful memories
		emaAlpha          = 0.3   // EMA smoothing factor (weight for new feedback)
		scoreMin          = -10.0 // minimum score value
		scoreMax          = 10.0  // maximum score value
	)

	updates := make(map[types.AgentMemoryID]float64)

	// Process helpful memories
	for _, memID := range reflection.HelpfulMemories {
		if mem, exists := memories[memID]; exists {
			delta := helpfulScoreDelta
			newScore := emaAlpha*delta + (1-emaAlpha)*mem.Score
			newScore = clamp(newScore, scoreMin, scoreMax)
			updates[memID] = newScore
		}
	}

	// Process harmful memories
	for _, memID := range reflection.HarmfulMemories {
		if mem, exists := memories[memID]; exists {
			delta := harmfulScoreDelta
			newScore := emaAlpha*delta + (1-emaAlpha)*mem.Score
			newScore = clamp(newScore, scoreMin, scoreMax)
			updates[memID] = newScore
		}
	}

	return updates
}

// NewAggressiveScoringAlgorithm creates a scoring algorithm with more aggressive feedback
// Example of how to create custom scoring algorithms with different parameters
func NewAggressiveScoringAlgorithm() ScoringAlgorithm {
	return func(
		memories map[types.AgentMemoryID]*memory.AgentMemory,
		reflection *memory.Reflection,
	) map[types.AgentMemoryID]float64 {
		const (
			helpfulScoreDelta = 3.0  // more aggressive positive feedback
			harmfulScoreDelta = -5.0 // more aggressive negative feedback
			emaAlpha          = 0.5  // higher weight on new feedback
			scoreMin          = -10.0
			scoreMax          = 10.0
		)

		updates := make(map[types.AgentMemoryID]float64)

		for _, memID := range reflection.HelpfulMemories {
			if mem, exists := memories[memID]; exists {
				delta := helpfulScoreDelta
				newScore := emaAlpha*delta + (1-emaAlpha)*mem.Score
				newScore = clamp(newScore, scoreMin, scoreMax)
				updates[memID] = newScore
			}
		}

		for _, memID := range reflection.HarmfulMemories {
			if mem, exists := memories[memID]; exists {
				delta := harmfulScoreDelta
				newScore := emaAlpha*delta + (1-emaAlpha)*mem.Score
				newScore = clamp(newScore, scoreMin, scoreMax)
				updates[memID] = newScore
			}
		}

		return updates
	}
}

// clamp constrains a value to be within [min, max]
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
