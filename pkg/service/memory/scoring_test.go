package memory_test

import (
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
	memoryService "github.com/secmon-lab/warren/pkg/service/memory"
)

func TestDefaultScoringAlgorithm(t *testing.T) {
	t.Run("helpful memory increases score", func(t *testing.T) {
		mem1 := &memory.AgentMemory{
			ID:             types.AgentMemoryID("mem-1"),
			AgentID:        "test",
			Query:          "test query",
			QueryEmbedding: firestore.Vector32{0.1, 0.2, 0.3},
			Claim:          "test claim",
			Score:          0.0,
			CreatedAt:      time.Now(),
		}

		memories := map[types.AgentMemoryID]*memory.AgentMemory{
			mem1.ID: mem1,
		}

		reflection := &memory.Reflection{
			HelpfulMemories: []types.AgentMemoryID{mem1.ID},
		}

		updates := memoryService.DefaultScoringAlgorithm(memories, reflection)
		gt.V(t, len(updates)).Equal(1)

		// With default EMA alpha=0.3, delta=+2.0:
		// newScore = 0.3 * 2.0 + 0.7 * 0.0 = 0.6
		newScore := updates[mem1.ID]
		gt.V(t, newScore).Equal(0.6)
	})

	t.Run("harmful memory decreases score", func(t *testing.T) {
		mem1 := &memory.AgentMemory{
			ID:             types.AgentMemoryID("mem-1"),
			AgentID:        "test",
			Query:          "test query",
			QueryEmbedding: firestore.Vector32{0.1, 0.2, 0.3},
			Claim:          "test claim",
			Score:          0.0,
			CreatedAt:      time.Now(),
		}

		memories := map[types.AgentMemoryID]*memory.AgentMemory{
			mem1.ID: mem1,
		}

		reflection := &memory.Reflection{
			HarmfulMemories: []types.AgentMemoryID{mem1.ID},
		}

		updates := memoryService.DefaultScoringAlgorithm(memories, reflection)
		gt.V(t, len(updates)).Equal(1)

		// With default EMA alpha=0.3, delta=-3.0:
		// newScore = 0.3 * (-3.0) + 0.7 * 0.0 = -0.9
		newScore := updates[mem1.ID]
		// Use approximate comparison for float
		gt.True(t, newScore > -0.91 && newScore < -0.89)
	})

	t.Run("EMA smoothing with existing score", func(t *testing.T) {
		mem1 := &memory.AgentMemory{
			ID:             types.AgentMemoryID("mem-1"),
			AgentID:        "test",
			Query:          "test query",
			QueryEmbedding: firestore.Vector32{0.1, 0.2, 0.3},
			Claim:          "test claim",
			Score:          1.0, // existing positive score
			CreatedAt:      time.Now(),
		}

		memories := map[types.AgentMemoryID]*memory.AgentMemory{
			mem1.ID: mem1,
		}

		reflection := &memory.Reflection{
			HelpfulMemories: []types.AgentMemoryID{mem1.ID},
		}

		updates := memoryService.DefaultScoringAlgorithm(memories, reflection)

		// With alpha=0.3, delta=+2.0, oldScore=1.0:
		// newScore = 0.3 * 2.0 + 0.7 * 1.0 = 0.6 + 0.7 = 1.3
		newScore := updates[mem1.ID]
		gt.True(t, newScore > 1.29 && newScore < 1.31)
	})

	t.Run("score clamped at max", func(t *testing.T) {
		mem1 := &memory.AgentMemory{
			ID:             types.AgentMemoryID("mem-1"),
			AgentID:        "test",
			Query:          "test query",
			QueryEmbedding: firestore.Vector32{0.1, 0.2, 0.3},
			Claim:          "test claim",
			Score:          9.5, // near max
			CreatedAt:      time.Now(),
		}

		memories := map[types.AgentMemoryID]*memory.AgentMemory{
			mem1.ID: mem1,
		}

		reflection := &memory.Reflection{
			HelpfulMemories: []types.AgentMemoryID{mem1.ID},
		}

		// Test with score near max
		mem1.Score = 9.9
		updates := memoryService.DefaultScoringAlgorithm(memories, reflection)

		// 0.3 * 2.0 + 0.7 * 9.9 = 0.6 + 6.93 = 7.53
		newScore := updates[mem1.ID]
		gt.True(t, newScore <= 10.0)
		gt.True(t, newScore > 7.52 && newScore < 7.54)
	})

	t.Run("score clamped at min", func(t *testing.T) {
		mem1 := &memory.AgentMemory{
			ID:             types.AgentMemoryID("mem-1"),
			AgentID:        "test",
			Query:          "test query",
			QueryEmbedding: firestore.Vector32{0.1, 0.2, 0.3},
			Claim:          "test claim",
			Score:          -9.9, // near min
			CreatedAt:      time.Now(),
		}

		memories := map[types.AgentMemoryID]*memory.AgentMemory{
			mem1.ID: mem1,
		}

		reflection := &memory.Reflection{
			HarmfulMemories: []types.AgentMemoryID{mem1.ID},
		}

		updates := memoryService.DefaultScoringAlgorithm(memories, reflection)

		// 0.3 * (-3.0) + 0.7 * (-9.9) = -0.9 + (-6.93) = -7.83
		newScore := updates[mem1.ID]
		gt.True(t, newScore >= -10.0)
		gt.V(t, newScore).Equal(-7.83)
	})

	t.Run("multiple memories", func(t *testing.T) {
		mem1 := &memory.AgentMemory{
			ID:        types.AgentMemoryID("mem-1"),
			Score:     0.0,
			CreatedAt: time.Now(),
		}
		mem2 := &memory.AgentMemory{
			ID:        types.AgentMemoryID("mem-2"),
			Score:     1.0,
			CreatedAt: time.Now(),
		}
		mem3 := &memory.AgentMemory{
			ID:        types.AgentMemoryID("mem-3"),
			Score:     -2.0,
			CreatedAt: time.Now(),
		}

		memories := map[types.AgentMemoryID]*memory.AgentMemory{
			mem1.ID: mem1,
			mem2.ID: mem2,
			mem3.ID: mem3,
		}

		reflection := &memory.Reflection{
			HelpfulMemories: []types.AgentMemoryID{mem1.ID, mem2.ID},
			HarmfulMemories: []types.AgentMemoryID{mem3.ID},
		}

		updates := memoryService.DefaultScoringAlgorithm(memories, reflection)
		gt.V(t, len(updates)).Equal(3)

		// mem1: 0.3 * 2.0 + 0.7 * 0.0 = 0.6
		gt.V(t, updates[mem1.ID]).Equal(0.6)

		// mem2: 0.3 * 2.0 + 0.7 * 1.0 = 1.3
		gt.True(t, updates[mem2.ID] > 1.29 && updates[mem2.ID] < 1.31)

		// mem3: 0.3 * (-3.0) + 0.7 * (-2.0) = -0.9 + (-1.4) = -2.3
		gt.True(t, updates[mem3.ID] > -2.31 && updates[mem3.ID] < -2.29)
	})

	t.Run("nonexistent memory ID ignored", func(t *testing.T) {
		mem1 := &memory.AgentMemory{
			ID:        types.AgentMemoryID("mem-1"),
			Score:     0.0,
			CreatedAt: time.Now(),
		}

		memories := map[types.AgentMemoryID]*memory.AgentMemory{
			mem1.ID: mem1,
		}

		reflection := &memory.Reflection{
			HelpfulMemories: []types.AgentMemoryID{types.AgentMemoryID("nonexistent")},
		}

		updates := memoryService.DefaultScoringAlgorithm(memories, reflection)
		gt.V(t, len(updates)).Equal(0) // nonexistent ID should be ignored
	})
}

func TestAggressiveScoringAlgorithm(t *testing.T) {
	t.Run("more aggressive helpful feedback", func(t *testing.T) {
		mem1 := &memory.AgentMemory{
			ID:        types.AgentMemoryID("mem-1"),
			Score:     0.0,
			CreatedAt: time.Now(),
		}

		memories := map[types.AgentMemoryID]*memory.AgentMemory{
			mem1.ID: mem1,
		}

		reflection := &memory.Reflection{
			HelpfulMemories: []types.AgentMemoryID{mem1.ID},
		}

		aggressiveAlgo := memoryService.NewAggressiveScoringAlgorithm()
		updates := aggressiveAlgo(memories, reflection)

		// With aggressive: alpha=0.5, delta=+3.0:
		// newScore = 0.5 * 3.0 + 0.5 * 0.0 = 1.5
		newScore := updates[mem1.ID]
		gt.V(t, newScore).Equal(1.5)

		// Compare with default algorithm
		defaultUpdates := memoryService.DefaultScoringAlgorithm(memories, reflection)
		defaultScore := defaultUpdates[mem1.ID]

		// Aggressive should give higher score for helpful memory
		gt.True(t, newScore > defaultScore)
	})

	t.Run("more aggressive harmful feedback", func(t *testing.T) {
		mem1 := &memory.AgentMemory{
			ID:        types.AgentMemoryID("mem-1"),
			Score:     0.0,
			CreatedAt: time.Now(),
		}

		memories := map[types.AgentMemoryID]*memory.AgentMemory{
			mem1.ID: mem1,
		}

		reflection := &memory.Reflection{
			HarmfulMemories: []types.AgentMemoryID{mem1.ID},
		}

		aggressiveAlgo := memoryService.NewAggressiveScoringAlgorithm()
		updates := aggressiveAlgo(memories, reflection)

		// With aggressive: alpha=0.5, delta=-5.0:
		// newScore = 0.5 * (-5.0) + 0.5 * 0.0 = -2.5
		newScore := updates[mem1.ID]
		gt.V(t, newScore).Equal(-2.5)

		// Compare with default algorithm
		defaultUpdates := memoryService.DefaultScoringAlgorithm(memories, reflection)
		defaultScore := defaultUpdates[mem1.ID]

		// Aggressive should give lower (more negative) score for harmful memory
		gt.True(t, newScore < defaultScore)
	})
}
