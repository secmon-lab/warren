package memory

import (
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// PruningAlgorithm selects memories to be deleted based on score and usage patterns
// Takes all memories for an agent and current time
// Returns list of memory IDs that should be deleted
type PruningAlgorithm func(
	memories []*memory.AgentMemory,
	now time.Time,
) []types.AgentMemoryID

// DefaultPruningAlgorithm is the default implementation of pruning algorithm
// Parameters are defined as constants within the function
// Uses conservative approach with 3-tier criteria:
// 1. Critical: Score <= -8.0 (immediate deletion)
// 2. Harmful + Stale: Score <= -5.0 AND unused for 90+ days
// 3. Moderate + Very Stale: Score <= -3.0 AND unused for 180+ days
var DefaultPruningAlgorithm PruningAlgorithm = func(
	memories []*memory.AgentMemory,
	now time.Time,
) []types.AgentMemoryID {
	// Algorithm parameters (const within function)
	const (
		criticalThreshold = -8.0 // Critical score threshold for immediate deletion
		harmfulThreshold  = -5.0 // Harmful score threshold
		harmfulMaxDays    = 90   // Max days unused for harmful memories
		moderateThreshold = -3.0 // Moderate score threshold
		moderateMaxDays   = 180  // Max days unused for moderate memories
	)

	var toDelete []types.AgentMemoryID

	for _, mem := range memories {
		shouldDelete := false

		// Criterion 1: Critical score
		if mem.Score <= criticalThreshold {
			shouldDelete = true
		}

		// Criterion 2: Harmful + Stale
		if !shouldDelete && mem.Score <= harmfulThreshold {
			daysSinceUsed := calculateDaysSinceUsed(mem, now)
			if daysSinceUsed >= harmfulMaxDays {
				shouldDelete = true
			}
		}

		// Criterion 3: Moderate + Very Stale
		if !shouldDelete && mem.Score <= moderateThreshold {
			daysSinceUsed := calculateDaysSinceUsed(mem, now)
			if daysSinceUsed >= moderateMaxDays {
				shouldDelete = true
			}
		}

		if shouldDelete {
			toDelete = append(toDelete, mem.ID)
		}
	}

	return toDelete
}

// calculateDaysSinceUsed computes the number of days since a memory was last used
// If LastUsedAt is zero, uses CreatedAt as the reference point
func calculateDaysSinceUsed(mem *memory.AgentMemory, now time.Time) int {
	if !mem.LastUsedAt.IsZero() {
		return int(now.Sub(mem.LastUsedAt).Hours() / 24)
	}
	return int(now.Sub(mem.CreatedAt).Hours() / 24)
}
