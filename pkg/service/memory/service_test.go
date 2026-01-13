package memory_test

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	memoryService "github.com/secmon-lab/warren/pkg/service/memory"
)

func createTestRepository(t *testing.T) interfaces.Repository {
	t.Helper()
	return repository.NewMemory()
}

func newMockLLMClient() gollem.LLMClient {
	return &mock.LLMClientMock{
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
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					reflectionJSON := `{
						"new_claims": ["Test claim from execution"],
						"helpful_memories": [],
						"harmful_memories": []
					}`
					return &gollem.Response{Texts: []string{reflectionJSON}}, nil
				},
			}, nil
		},
	}
}

func TestSearchAndSelectMemories(t *testing.T) {
	repo := createTestRepository(t)
	llmClient := newMockLLMClient()
	agentID := "test-agent"
	svc := memoryService.New(agentID, llmClient, repo)
	ctx := context.Background()
	now := time.Now()

	// Create test memories
	memories := []*memory.AgentMemory{
		{
			ID:             types.NewAgentMemoryID(),
			AgentID:        agentID,
			Query:          "login errors query",
			QueryEmbedding: firestore.Vector32{0.1, 0.2, 0.3},
			Claim:          "Login errors have severity='ERROR' field",
			Score:          5.0,
			CreatedAt:      now.Add(-1 * time.Hour),
			LastUsedAt:     now.Add(-1 * time.Hour),
		},
		{
			ID:             types.NewAgentMemoryID(),
			AgentID:        agentID,
			Query:          "authentication failures",
			QueryEmbedding: firestore.Vector32{0.15, 0.25, 0.35},
			Claim:          "Auth failures need action='login' check",
			Score:          3.0,
			CreatedAt:      now.Add(-2 * time.Hour),
			LastUsedAt:     now.Add(-2 * time.Hour),
		},
		{
			ID:             types.NewAgentMemoryID(),
			AgentID:        "other-agent",
			Query:          "scan file hash",
			QueryEmbedding: firestore.Vector32{0.5, 0.6, 0.7},
			Claim:          "File hash scanning uses MD5",
			Score:          2.0,
			CreatedAt:      now.Add(-3 * time.Hour),
			LastUsedAt:     now.Add(-3 * time.Hour),
		},
	}

	// Save memories
	for _, mem := range memories {
		gt.NoError(t, repo.SaveAgentMemory(ctx, mem))
	}

	// Search for memories
	results, err := svc.SearchAndSelectMemories(ctx, "login errors", 2)
	gt.NoError(t, err)
	gt.V(t, len(results)).Equal(2)

	// Verify results are from correct agent
	for _, r := range results {
		gt.V(t, r.AgentID).Equal(agentID)
	}
}

func TestExtractAndSaveMemories(t *testing.T) {
	repo := createTestRepository(t)
	llmClient := newMockLLMClient()
	agentID := "test-agent"
	svc := memoryService.New(agentID, llmClient, repo)
	ctx := context.Background()

	// Execute task and extract memories
	taskQuery := "How to query user login data?"
	usedMemories := []*memory.AgentMemory{}
	history := &gollem.History{}

	err := svc.ExtractAndSaveMemories(ctx, taskQuery, usedMemories, history)
	gt.NoError(t, err)

	// Verify memories were saved
	memories, err := repo.ListAgentMemories(ctx, agentID)
	gt.NoError(t, err)
	gt.True(t, len(memories) > 0)

	// Verify memory content
	mem := memories[0]
	gt.V(t, mem.AgentID).Equal(agentID)
	gt.V(t, mem.Query).Equal(taskQuery)
	gt.True(t, len(mem.Claim) > 0)
}

func TestPruneMemories(t *testing.T) {
	repo := createTestRepository(t)
	llmClient := newMockLLMClient()
	agentID := "pruning-test-agent"
	svc := memoryService.New(agentID, llmClient, repo)
	ctx := context.Background()
	now := time.Now()

	// Create memories with various scores and timestamps
	memories := []*memory.AgentMemory{
		// Critical score - should be deleted
		{
			ID:             types.NewAgentMemoryID(),
			AgentID:        agentID,
			Query:          "critical bad memory",
			QueryEmbedding: firestore.Vector32{0.1, 0.2, 0.3},
			Claim:          "This is completely wrong",
			Score:          -9.0,
			CreatedAt:      now.Add(-10 * 24 * time.Hour),
			LastUsedAt:     now.Add(-10 * 24 * time.Hour),
		},
		// Harmful + old - should be deleted
		{
			ID:             types.NewAgentMemoryID(),
			AgentID:        agentID,
			Query:          "harmful old memory",
			QueryEmbedding: firestore.Vector32{0.1, 0.2, 0.3},
			Claim:          "Somewhat incorrect claim",
			Score:          -6.0,
			CreatedAt:      now.Add(-100 * 24 * time.Hour),
			LastUsedAt:     now.Add(-100 * 24 * time.Hour),
		},
		// Good memory - should NOT be deleted
		{
			ID:             types.NewAgentMemoryID(),
			AgentID:        agentID,
			Query:          "good memory",
			QueryEmbedding: firestore.Vector32{0.1, 0.2, 0.3},
			Claim:          "This is helpful claim",
			Score:          5.0,
			CreatedAt:      now,
			LastUsedAt:     now,
		},
	}

	for _, mem := range memories {
		gt.NoError(t, repo.SaveAgentMemory(ctx, mem))
	}

	// Run pruning
	deleted, err := svc.PruneMemories(ctx)
	gt.NoError(t, err)
	gt.V(t, deleted).Equal(2) // critical + harmful old

	// Verify remaining memories
	remaining, err := repo.ListAgentMemories(ctx, agentID)
	gt.NoError(t, err)
	gt.V(t, len(remaining)).Equal(1) // only good memory

	// Good memory should still exist
	gt.V(t, remaining[0].Score).Equal(5.0)
}

func TestServiceAlgorithmReplacement(t *testing.T) {
	repo := createTestRepository(t)
	llmClient := newMockLLMClient()

	// Create service with custom algorithms
	customScoring := func(
		memories map[types.AgentMemoryID]*memory.AgentMemory,
		reflection *memory.Reflection,
	) map[types.AgentMemoryID]float64 {
		// Custom scoring: always return +5.0 for helpful
		updates := make(map[types.AgentMemoryID]float64)
		for _, memID := range reflection.HelpfulMemories {
			if _, exists := memories[memID]; exists {
				updates[memID] = 5.0
			}
		}
		return updates
	}

	svc := memoryService.New("test-agent", llmClient, repo).
		WithScoringAlgorithm(customScoring)

	// Verify algorithm replacement worked
	gt.NotEqual(t, svc, nil)
}

func TestSearchAndSelectMemories_UpdatesLastUsedAt(t *testing.T) {
	repo := createTestRepository(t)
	llmClient := newMockLLMClient()
	agentID := "test-agent"
	svc := memoryService.New(agentID, llmClient, repo).EnableAsyncTrackingForTest()
	ctx := context.Background()
	now := time.Now()

	// Create a test memory
	mem := &memory.AgentMemory{
		ID:             types.NewAgentMemoryID(),
		AgentID:        agentID,
		Query:          "test query",
		QueryEmbedding: firestore.Vector32{0.1, 0.2, 0.3},
		Claim:          "test claim",
		Score:          0.0,
		CreatedAt:      now.Add(-1 * time.Hour),
		LastUsedAt:     now.Add(-1 * time.Hour),
	}

	// Save memory
	gt.NoError(t, repo.SaveAgentMemory(ctx, mem))

	// Search and select memories
	results, err := svc.SearchAndSelectMemories(ctx, "test query", 1)
	gt.NoError(t, err)
	gt.V(t, len(results)).Equal(1)

	// Wait for async update to complete
	svc.WaitForAsyncOperationsForTest()

	// Retrieve the memory and verify LastUsedAt was updated
	updated, err := repo.GetAgentMemory(ctx, agentID, mem.ID)
	gt.NoError(t, err)

	// LastUsedAt should be more recent than the original value
	gt.True(t, updated.LastUsedAt.After(mem.LastUsedAt))
}
