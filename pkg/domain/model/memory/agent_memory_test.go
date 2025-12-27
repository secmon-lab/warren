package memory_test

import (
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func TestAgentMemory_Validate(t *testing.T) {
	validMemory := &memory.AgentMemory{
		ID:             types.AgentMemoryID("mem-123"),
		AgentID:        "bigquery",
		Query:          "Find login failures",
		QueryEmbedding: firestore.Vector32{0.1, 0.2, 0.3},
		Claim:          "Login failures are identified by severity='ERROR'",
		Score:          5.0,
		CreatedAt:      time.Now(),
		LastUsedAt:     time.Now(),
	}

	t.Run("valid memory", func(t *testing.T) {
		err := validMemory.Validate()
		gt.NoError(t, err)
	})

	t.Run("missing ID", func(t *testing.T) {
		m := *validMemory
		m.ID = ""
		err := m.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("agent memory ID is required")
	})

	t.Run("missing AgentID", func(t *testing.T) {
		m := *validMemory
		m.AgentID = ""
		err := m.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("agent ID is required")
	})

	t.Run("missing Query", func(t *testing.T) {
		m := *validMemory
		m.Query = ""
		err := m.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("query is required")
	})

	t.Run("missing Claim", func(t *testing.T) {
		m := *validMemory
		m.Claim = ""
		err := m.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("claim is required")
	})

	t.Run("score too low", func(t *testing.T) {
		m := *validMemory
		m.Score = -11.0
		err := m.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("score must be between -10.0 and +10.0")
	})

	t.Run("score too high", func(t *testing.T) {
		m := *validMemory
		m.Score = 11.0
		err := m.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("score must be between -10.0 and +10.0")
	})

	t.Run("score at boundaries", func(t *testing.T) {
		m1 := *validMemory
		m1.Score = -10.0
		gt.NoError(t, m1.Validate())

		m2 := *validMemory
		m2.Score = 10.0
		gt.NoError(t, m2.Validate())

		m3 := *validMemory
		m3.Score = 0.0
		gt.NoError(t, m3.Validate())
	})

	t.Run("missing CreatedAt", func(t *testing.T) {
		m := *validMemory
		m.CreatedAt = time.Time{}
		err := m.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("created_at is required")
	})

	t.Run("zero LastUsedAt is allowed", func(t *testing.T) {
		m := *validMemory
		m.LastUsedAt = time.Time{}
		err := m.Validate()
		gt.NoError(t, err)
	})
}

func TestAgentMemory_Structure(t *testing.T) {
	now := time.Now()
	embedding := firestore.Vector32{0.1, 0.2, 0.3}

	mem := &memory.AgentMemory{
		ID:             types.NewAgentMemoryID(),
		AgentID:        "bigquery",
		Query:          "get login errors",
		QueryEmbedding: embedding,
		Claim:          "Login errors have severity='ERROR' AND action='login'",
		Score:          3.5,
		CreatedAt:      now,
		LastUsedAt:     now.Add(-1 * time.Hour),
	}

	gt.V(t, mem.AgentID).Equal("bigquery")
	gt.V(t, mem.Query).Equal("get login errors")
	gt.V(t, len(mem.QueryEmbedding)).Equal(3)
	gt.V(t, mem.QueryEmbedding[0]).Equal(float32(0.1))
	gt.V(t, mem.Claim).Equal("Login errors have severity='ERROR' AND action='login'")
	gt.V(t, mem.Score).Equal(3.5)
	gt.True(t, mem.CreatedAt.Equal(now))
	gt.True(t, mem.LastUsedAt.Before(now))
}
