package memory_test

import (
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
	memsvc "github.com/secmon-lab/warren/pkg/service/memory"
)

// TestCalculateDaysSinceUsed tests the days since used calculation
func TestCalculateDaysSinceUsed(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

	t.Run("uses LastUsedAt when available", func(t *testing.T) {
		mem := &memory.AgentMemory{
			CreatedAt:  now.AddDate(0, 0, -100),
			LastUsedAt: now.AddDate(0, 0, -30),
		}
		days := memsvc.CalculateDaysSinceUsed(mem, now)
		gt.V(t, days).Equal(30)
	})

	t.Run("uses CreatedAt when LastUsedAt is zero", func(t *testing.T) {
		mem := &memory.AgentMemory{
			CreatedAt:  now.AddDate(0, 0, -100),
			LastUsedAt: time.Time{},
		}
		days := memsvc.CalculateDaysSinceUsed(mem, now)
		gt.V(t, days).Equal(100)
	})

	t.Run("zero days when used today", func(t *testing.T) {
		mem := &memory.AgentMemory{
			CreatedAt:  now.AddDate(0, 0, -10),
			LastUsedAt: now,
		}
		days := memsvc.CalculateDaysSinceUsed(mem, now)
		gt.V(t, days).Equal(0)
	})

	t.Run("exact day boundaries", func(t *testing.T) {
		mem := &memory.AgentMemory{
			CreatedAt:  now.AddDate(0, 0, -10),
			LastUsedAt: now.Add(-24 * time.Hour),
		}
		days := memsvc.CalculateDaysSinceUsed(mem, now)
		gt.V(t, days).Equal(1)
	})
}

// TestDefaultPruningAlgorithm tests the full pruning algorithm
func TestDefaultPruningAlgorithm(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

	// Helper to create a test memory
	createMemory := func(id string, score float64, lastUsedDaysAgo int, createdDaysAgo int) *memory.AgentMemory {
		mem := &memory.AgentMemory{
			ID:             types.AgentMemoryID(id),
			AgentID:        "test",
			Query:          "query",
			QueryEmbedding: firestore.Vector32{1.0, 0.0, 0.0},
			Claim:          "claim " + id,
			Score:          score,
			CreatedAt:      now.AddDate(0, 0, -createdDaysAgo),
		}
		if lastUsedDaysAgo >= 0 {
			mem.LastUsedAt = now.AddDate(0, 0, -lastUsedDaysAgo)
		}
		return mem
	}

	t.Run("criterion 1: critical score deletes immediately", func(t *testing.T) {
		memories := []*memory.AgentMemory{
			createMemory("A", -9.0, 1, 10),    // Score <= -8.0, recently used
			createMemory("B", -8.0, 0, 5),     // Score <= -8.0, used today
			createMemory("C", -8.5, 200, 300), // Score <= -8.0, very old
		}

		toDelete := memsvc.DefaultPruningAlgorithm(memories, now)

		deleteMap := make(map[types.AgentMemoryID]bool)
		for _, id := range toDelete {
			deleteMap[id] = true
		}
		gt.True(t, deleteMap[types.AgentMemoryID("A")])
		gt.True(t, deleteMap[types.AgentMemoryID("B")])
		gt.True(t, deleteMap[types.AgentMemoryID("C")])
		gt.V(t, len(deleteMap)).Equal(3)
	})

	t.Run("criterion 2: harmful + stale (90+ days)", func(t *testing.T) {
		memories := []*memory.AgentMemory{
			createMemory("A", -6.0, 100, 150), // Harmful + stale (>90 days)
			createMemory("B", -5.0, 90, 120),  // Harmful + exactly 90 days
			createMemory("C", -6.0, 50, 100),  // Harmful but recently used
			createMemory("D", -5.0, 89, 120),  // Harmful but not stale enough
		}

		toDelete := memsvc.DefaultPruningAlgorithm(memories, now)

		deleteMap := make(map[types.AgentMemoryID]bool)
		for _, id := range toDelete {
			deleteMap[id] = true
		}
		gt.True(t, deleteMap[types.AgentMemoryID("A")])
		gt.True(t, deleteMap[types.AgentMemoryID("B")])
		gt.False(t, deleteMap[types.AgentMemoryID("C")])
		gt.False(t, deleteMap[types.AgentMemoryID("D")])
		gt.V(t, len(deleteMap)).Equal(2)
	})

	t.Run("criterion 3: moderate + very stale (180+ days)", func(t *testing.T) {
		memories := []*memory.AgentMemory{
			createMemory("A", -4.0, 200, 250), // Moderate + very stale (>180 days)
			createMemory("B", -3.0, 180, 220), // Moderate + exactly 180 days
			createMemory("C", -4.0, 100, 150), // Moderate but not stale enough
			createMemory("D", -3.0, 179, 220), // Moderate but not stale enough
		}

		toDelete := memsvc.DefaultPruningAlgorithm(memories, now)

		deleteMap := make(map[types.AgentMemoryID]bool)
		for _, id := range toDelete {
			deleteMap[id] = true
		}
		gt.True(t, deleteMap[types.AgentMemoryID("A")])
		gt.True(t, deleteMap[types.AgentMemoryID("B")])
		gt.False(t, deleteMap[types.AgentMemoryID("C")])
		gt.False(t, deleteMap[types.AgentMemoryID("D")])
		gt.V(t, len(deleteMap)).Equal(2)
	})

	t.Run("preserves neutral and positive scores", func(t *testing.T) {
		memories := []*memory.AgentMemory{
			createMemory("A", -2.0, 365, 400), // Above moderate threshold
			createMemory("B", 0.0, 365, 400),  // Neutral
			createMemory("C", 5.0, 365, 400),  // Positive
		}

		toDelete := memsvc.DefaultPruningAlgorithm(memories, now)
		gt.V(t, len(toDelete)).Equal(0)
	})

	t.Run("never used memories use CreatedAt", func(t *testing.T) {
		memories := []*memory.AgentMemory{
			createMemory("A", -6.0, -1, 100), // -1 means never used, created 100 days ago
			createMemory("B", -4.0, -1, 200), // Never used, created 200 days ago
		}

		toDelete := memsvc.DefaultPruningAlgorithm(memories, now)

		deleteMap := make(map[types.AgentMemoryID]bool)
		for _, id := range toDelete {
			deleteMap[id] = true
		}
		gt.True(t, deleteMap[types.AgentMemoryID("A")])
		gt.True(t, deleteMap[types.AgentMemoryID("B")])
		gt.V(t, len(deleteMap)).Equal(2)
	})

	t.Run("comprehensive scenario from spec", func(t *testing.T) {
		memories := []*memory.AgentMemory{
			createMemory("A", -9.0, 1, 10),    // Critical -> DELETE
			createMemory("B", -6.0, 100, 150), // Harmful+Stale -> DELETE
			createMemory("C", -6.0, 50, 100),  // Harmful but recent -> KEEP
			createMemory("D", -4.0, 200, 250), // Moderate+VeryStale -> DELETE
			createMemory("E", -4.0, 100, 150), // Moderate but not stale enough -> KEEP
			createMemory("F", -2.0, -1, 200),  // Above threshold -> KEEP
			createMemory("G", 0.0, -1, 365),   // Neutral -> KEEP
			createMemory("H", 5.0, 200, 250),  // Positive -> KEEP
		}

		toDelete := memsvc.DefaultPruningAlgorithm(memories, now)

		deleteMap := make(map[types.AgentMemoryID]bool)
		for _, id := range toDelete {
			deleteMap[id] = true
		}
		gt.True(t, deleteMap[types.AgentMemoryID("A")])
		gt.True(t, deleteMap[types.AgentMemoryID("B")])
		gt.False(t, deleteMap[types.AgentMemoryID("C")])
		gt.True(t, deleteMap[types.AgentMemoryID("D")])
		gt.False(t, deleteMap[types.AgentMemoryID("E")])
		gt.False(t, deleteMap[types.AgentMemoryID("F")])
		gt.False(t, deleteMap[types.AgentMemoryID("G")])
		gt.False(t, deleteMap[types.AgentMemoryID("H")])
		gt.V(t, len(deleteMap)).Equal(3)
	})

	t.Run("handles empty memories list", func(t *testing.T) {
		memories := []*memory.AgentMemory{}
		toDelete := memsvc.DefaultPruningAlgorithm(memories, now)
		gt.V(t, len(toDelete)).Equal(0)
	})

	t.Run("handles all memories kept", func(t *testing.T) {
		memories := []*memory.AgentMemory{
			createMemory("A", 5.0, 10, 20),
			createMemory("B", 3.0, 5, 15),
			createMemory("C", 0.0, 1, 10),
		}

		toDelete := memsvc.DefaultPruningAlgorithm(memories, now)
		gt.V(t, len(toDelete)).Equal(0)
	})

	t.Run("boundary conditions for thresholds", func(t *testing.T) {
		memories := []*memory.AgentMemory{
			createMemory("A1", -8.0, 0, 10),    // DELETE (<=)
			createMemory("A2", -7.9, 0, 10),    // KEEP (>)
			createMemory("B1", -5.0, 90, 100),  // DELETE (<=)
			createMemory("B2", -4.9, 90, 100),  // KEEP (>)
			createMemory("C1", -3.0, 180, 200), // DELETE (<=)
			createMemory("C2", -2.9, 180, 200), // KEEP (>)
		}

		toDelete := memsvc.DefaultPruningAlgorithm(memories, now)

		deleteMap := make(map[types.AgentMemoryID]bool)
		for _, id := range toDelete {
			deleteMap[id] = true
		}
		gt.True(t, deleteMap[types.AgentMemoryID("A1")])
		gt.False(t, deleteMap[types.AgentMemoryID("A2")])
		gt.True(t, deleteMap[types.AgentMemoryID("B1")])
		gt.False(t, deleteMap[types.AgentMemoryID("B2")])
		gt.True(t, deleteMap[types.AgentMemoryID("C1")])
		gt.False(t, deleteMap[types.AgentMemoryID("C2")])
		gt.V(t, len(deleteMap)).Equal(3)
	})

	t.Run("boundary conditions for days", func(t *testing.T) {
		memories := []*memory.AgentMemory{
			createMemory("A1", -5.0, 90, 100),  // DELETE (>=)
			createMemory("A2", -5.0, 89, 100),  // KEEP (<)
			createMemory("B1", -3.0, 180, 200), // DELETE (>=)
			createMemory("B2", -3.0, 179, 200), // KEEP (<)
		}

		toDelete := memsvc.DefaultPruningAlgorithm(memories, now)

		deleteMap := make(map[types.AgentMemoryID]bool)
		for _, id := range toDelete {
			deleteMap[id] = true
		}
		gt.True(t, deleteMap[types.AgentMemoryID("A1")])
		gt.False(t, deleteMap[types.AgentMemoryID("A2")])
		gt.True(t, deleteMap[types.AgentMemoryID("B1")])
		gt.False(t, deleteMap[types.AgentMemoryID("B2")])
		gt.V(t, len(deleteMap)).Equal(2)
	})

	t.Run("criteria priority - critical overrides all", func(t *testing.T) {
		memories := []*memory.AgentMemory{
			createMemory("A", -9.0, 0, 5), // Used today, but critical
		}

		toDelete := memsvc.DefaultPruningAlgorithm(memories, now)

		deleteMap := make(map[types.AgentMemoryID]bool)
		for _, id := range toDelete {
			deleteMap[id] = true
		}
		gt.True(t, deleteMap[types.AgentMemoryID("A")])
		gt.V(t, len(deleteMap)).Equal(1)
	})

	t.Run("multiple memories with different criteria", func(t *testing.T) {
		memories := []*memory.AgentMemory{
			createMemory("crit1", -10.0, 0, 5),    // Critical
			createMemory("crit2", -9.5, 10, 20),   // Critical
			createMemory("harm1", -6.0, 95, 120),  // Harmful+Stale
			createMemory("harm2", -5.5, 100, 150), // Harmful+Stale
			createMemory("mod1", -4.0, 185, 220),  // Moderate+VeryStale
			createMemory("mod2", -3.5, 190, 230),  // Moderate+VeryStale
			createMemory("keep1", -5.0, 80, 100),  // Harmful but not stale
			createMemory("keep2", -3.0, 170, 200), // Moderate but not very stale
			createMemory("keep3", 0.0, 200, 250),  // Neutral
		}

		toDelete := memsvc.DefaultPruningAlgorithm(memories, now)

		deleteMap := make(map[types.AgentMemoryID]bool)
		for _, id := range toDelete {
			deleteMap[id] = true
		}
		gt.True(t, deleteMap[types.AgentMemoryID("crit1")])
		gt.True(t, deleteMap[types.AgentMemoryID("crit2")])
		gt.True(t, deleteMap[types.AgentMemoryID("harm1")])
		gt.True(t, deleteMap[types.AgentMemoryID("harm2")])
		gt.True(t, deleteMap[types.AgentMemoryID("mod1")])
		gt.True(t, deleteMap[types.AgentMemoryID("mod2")])
		gt.False(t, deleteMap[types.AgentMemoryID("keep1")])
		gt.False(t, deleteMap[types.AgentMemoryID("keep2")])
		gt.False(t, deleteMap[types.AgentMemoryID("keep3")])
		gt.V(t, len(deleteMap)).Equal(6)
	})
}
