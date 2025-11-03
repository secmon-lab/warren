package memory_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/llm/claude"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	memoryService "github.com/secmon-lab/warren/pkg/service/memory"
)

// mockLLMClient is a simple mock for testing
type mockLLMClient struct{}

func (m *mockLLMClient) NewSession(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
	return nil, nil
}

func (m *mockLLMClient) GenerateEmbedding(ctx context.Context, dimension int, input []string) ([][]float64, error) {
	// Return dummy embeddings for testing
	embeddings := make([][]float64, len(input))
	for i := range input {
		vec := make([]float64, dimension)
		for j := 0; j < dimension; j++ {
			vec[j] = 0.1 * float64(i+j)
		}
		embeddings[i] = vec
	}
	return embeddings, nil
}

func createTestRepository(t *testing.T) interfaces.Repository {
	t.Helper()
	return repository.NewMemory()
}

func TestFormatMemoriesForPrompt(t *testing.T) {
	t.Run("both memories present", func(t *testing.T) {
		svc := memoryService.New(nil, nil)

		execMem := &memory.ExecutionMemory{
			SchemaID:  types.AlertSchema("test-schema"),
			Keep:      "Use BigQuery for log analysis",
			Change:    "VirusTotal API often times out, use local cache",
			Notes:     "Data is in UTC timezone",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   1,
		}

		ticketMem := &memory.TicketMemory{
			SchemaID:  types.AlertSchema("test-schema"),
			Insights:  "This organization uses Cloudflare. IP blocks from Cloudflare are usually false positives.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   1,
		}

		result := svc.FormatMemoriesForPrompt(execMem, ticketMem)

		gt.V(t, result).NotEqual("")
		gt.True(t, strings.Contains(result, "# Accumulated Knowledge"))
		gt.True(t, strings.Contains(result, "## Execution Learnings"))
		gt.True(t, strings.Contains(result, "Use BigQuery for log analysis"))
		gt.True(t, strings.Contains(result, "VirusTotal API often times out"))
		gt.True(t, strings.Contains(result, "Data is in UTC timezone"))
		gt.True(t, strings.Contains(result, "## Organizational Security Knowledge"))
		gt.True(t, strings.Contains(result, "Cloudflare"))
	})

	t.Run("only execution memory", func(t *testing.T) {
		svc := memoryService.New(nil, nil)

		execMem := &memory.ExecutionMemory{
			SchemaID:  types.AlertSchema("test-schema"),
			Keep:      "Use BigQuery for log analysis",
			Change:    "",
			Notes:     "",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   1,
		}

		result := svc.FormatMemoriesForPrompt(execMem, nil)

		gt.V(t, result).NotEqual("")
		gt.True(t, strings.Contains(result, "# Accumulated Knowledge"))
		gt.True(t, strings.Contains(result, "Use BigQuery for log analysis"))
		gt.False(t, strings.Contains(result, "## Organizational Security Knowledge"))
	})

	t.Run("only ticket memory", func(t *testing.T) {
		svc := memoryService.New(nil, nil)

		ticketMem := &memory.TicketMemory{
			SchemaID:  types.AlertSchema("test-schema"),
			Insights:  "Organization uses Cloudflare",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   1,
		}

		result := svc.FormatMemoriesForPrompt(nil, ticketMem)

		gt.V(t, result).NotEqual("")
		gt.True(t, strings.Contains(result, "# Accumulated Knowledge"))
		gt.True(t, strings.Contains(result, "Organization uses Cloudflare"))
		gt.False(t, strings.Contains(result, "## Execution Learnings"))
	})

	t.Run("both memories empty", func(t *testing.T) {
		svc := memoryService.New(nil, nil)

		result := svc.FormatMemoriesForPrompt(nil, nil)

		gt.V(t, result).Equal("")
	})

	t.Run("memories present but empty content", func(t *testing.T) {
		svc := memoryService.New(nil, nil)

		execMem := &memory.ExecutionMemory{
			SchemaID:  types.AlertSchema("test-schema"),
			Keep:      "",
			Change:    "",
			Notes:     "",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   1,
		}

		ticketMem := &memory.TicketMemory{
			SchemaID:  types.AlertSchema("test-schema"),
			Insights:  "",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   1,
		}

		result := svc.FormatMemoriesForPrompt(execMem, ticketMem)

		gt.V(t, result).Equal("")
	})
}

func TestLoadMemoriesForPrompt(t *testing.T) {
	// This test requires a repository, so we'll create a simple integration test
	// that verifies the basic flow works with the Memory repository
	ctx := context.Background()

	t.Run("load non-existent memories returns nil", func(t *testing.T) {
		repo := createTestRepository(t)
		llmClient := &mockLLMClient{}
		svc := memoryService.New(llmClient, repo)

		schemaID := types.AlertSchema("non-existent")
		execMem, ticketMem, err := svc.LoadMemoriesForPrompt(ctx, schemaID)

		gt.NoError(t, err)
		gt.V(t, execMem).Nil()
		gt.V(t, ticketMem).Nil()
	})

	t.Run("load existing memories", func(t *testing.T) {
		repo := createTestRepository(t)
		llmClient := &mockLLMClient{}
		svc := memoryService.New(llmClient, repo)

		schemaID := types.AlertSchema("existing")

		// Store some memories first
		execMem := &memory.ExecutionMemory{
			SchemaID:  schemaID,
			Keep:      "Keep this",
			Change:    "Change that",
			Notes:     "Note something",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   1,
		}
		err := repo.PutExecutionMemory(ctx, execMem)
		gt.NoError(t, err)

		ticketMem := &memory.TicketMemory{
			SchemaID:  schemaID,
			Insights:  "Some insights",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   1,
		}
		err = repo.PutTicketMemory(ctx, ticketMem)
		gt.NoError(t, err)

		// Load them back
		loadedExec, loadedTicket, err := svc.LoadMemoriesForPrompt(ctx, schemaID)

		gt.NoError(t, err)
		gt.V(t, loadedExec).NotNil()
		gt.V(t, loadedExec.Keep).Equal("Keep this")
		gt.V(t, loadedExec.Change).Equal("Change that")
		gt.V(t, loadedExec.Notes).Equal("Note something")

		gt.V(t, loadedTicket).NotNil()
		gt.V(t, loadedTicket.Insights).Equal("Some insights")
	})
}

// TestGenerateExecutionMemoryWithRealLLM tests ExecutionMemory generation with real Claude LLM
// This test requires the following environment variables:
// - TEST_CLAUDE_PROJECT_ID: Google Cloud project ID
// - TEST_CLAUDE_LOCATION: Google Cloud location (e.g., us-central1)
// - TEST_CLAUDE_MODEL: (optional) Claude model name
func TestGenerateExecutionMemoryWithRealLLM(t *testing.T) {
	ctx := context.Background()

	projectID, ok := os.LookupEnv("TEST_CLAUDE_PROJECT_ID")
	if !ok {
		t.Skip("TEST_CLAUDE_PROJECT_ID is not set")
	}
	location, ok := os.LookupEnv("TEST_CLAUDE_LOCATION")
	if !ok {
		t.Skip("TEST_CLAUDE_LOCATION is not set")
	}
	model := os.Getenv("TEST_CLAUDE_MODEL")
	if model == "" {
		model = "claude-sonnet-4@20250514"
	}

	// Create real Claude client
	claudeClient, err := claude.NewWithVertex(ctx, location, projectID,
		claude.WithVertexModel(model),
	)
	gt.NoError(t, err).Required()

	repo := createTestRepository(t)
	svc := memoryService.New(claudeClient, repo)

	schemaID := types.AlertSchema("test-schema-" + time.Now().Format("20060102150405"))

	// Create a simple history for testing
	userContent, err := gollem.NewTextContent("Analyze the security alert for suspicious login activity")
	gt.NoError(t, err).Required()
	assistantContent, err := gollem.NewTextContent("I checked the IP address and found it's from a known VPN provider. Recommended blocking.")
	gt.NoError(t, err).Required()

	history := &gollem.History{
		Messages: []gollem.Message{
			{
				Role:     gollem.RoleUser,
				Contents: []gollem.MessageContent{userContent},
			},
			{
				Role:     gollem.RoleAssistant,
				Contents: []gollem.MessageContent{assistantContent},
			},
		},
	}

	// Generate execution memory
	execMem, err := svc.GenerateExecutionMemory(ctx, schemaID, history, nil)
	gt.NoError(t, err).Required()
	gt.V(t, execMem).NotNil()
	gt.V(t, execMem.SchemaID).Equal(schemaID)

	// Verify that at least one field has content (LLM should generate something)
	hasContent := execMem.Keep != "" || execMem.Change != "" || execMem.Notes != ""
	if !hasContent {
		t.Fatal("ExecutionMemory should have at least one field with content")
	}

	t.Logf("Generated ExecutionMemory:")
	t.Logf("  Keep: %s", execMem.Keep)
	t.Logf("  Change: %s", execMem.Change)
	t.Logf("  Notes: %s", execMem.Notes)
}

// TestGenerateTicketMemoryWithRealLLM tests TicketMemory generation with real Claude LLM
func TestGenerateTicketMemoryWithRealLLM(t *testing.T) {
	ctx := context.Background()

	projectID, ok := os.LookupEnv("TEST_CLAUDE_PROJECT_ID")
	if !ok {
		t.Skip("TEST_CLAUDE_PROJECT_ID is not set")
	}
	location, ok := os.LookupEnv("TEST_CLAUDE_LOCATION")
	if !ok {
		t.Skip("TEST_CLAUDE_LOCATION is not set")
	}
	model := os.Getenv("TEST_CLAUDE_MODEL")
	if model == "" {
		model = "claude-sonnet-4@20250514"
	}

	// Create real Claude client
	claudeClient, err := claude.NewWithVertex(ctx, location, projectID,
		claude.WithVertexModel(model),
	)
	gt.NoError(t, err).Required()

	repo := createTestRepository(t)
	svc := memoryService.New(claudeClient, repo)

	schemaID := types.AlertSchema("test-schema-" + time.Now().Format("20060102150405"))

	// Create a test ticket with resolution
	ticketData := &ticket.Ticket{
		ID:     types.NewTicketID(),
		Status: types.TicketStatusResolved,
		Finding: &ticket.Finding{
			Severity:       types.AlertSeverityHigh,
			Summary:        "Confirmed credential stuffing attack. User password was compromised.",
			Recommendation: "Force password reset, enable 2FA, and block attacker IP range",
		},
		CreatedAt: time.Now().Add(-2 * time.Hour),
		UpdatedAt: time.Now(),
	}
	ticketData.Title = "Suspicious Login from Unknown Location"
	ticketData.Description = "Multiple failed login attempts followed by successful login from new IP address"

	// Create mock comments representing the investigation
	comments := []ticket.Comment{
		{
			ID:        types.CommentID("comment-" + time.Now().Format("20060102150405") + "-1"),
			TicketID:  ticketData.ID,
			Comment:   "Investigating the login pattern. Checking if IP is from a known bad actor.",
			CreatedAt: time.Now().Add(-90 * time.Minute),
		},
		{
			ID:        types.CommentID("comment-" + time.Now().Format("20060102150405") + "-2"),
			TicketID:  ticketData.ID,
			Comment:   "IP is from a residential proxy network. Multiple accounts affected. This appears to be credential stuffing.",
			CreatedAt: time.Now().Add(-60 * time.Minute),
		},
		{
			ID:        types.CommentID("comment-" + time.Now().Format("20060102150405") + "-3"),
			TicketID:  ticketData.ID,
			Comment:   "Password reset initiated for affected user. Monitoring for similar patterns from same IP range.",
			CreatedAt: time.Now().Add(-30 * time.Minute),
		},
	}

	// Generate ticket memory
	ticketMem, err := svc.GenerateTicketMemory(ctx, schemaID, ticketData, comments)
	gt.NoError(t, err).Required()
	gt.V(t, ticketMem).NotNil()
	gt.V(t, ticketMem.SchemaID).Equal(schemaID)

	// Verify insights field has content (required field in schema)
	if ticketMem.Insights == "" {
		t.Fatal("TicketMemory insights should not be empty")
	}

	t.Logf("Generated TicketMemory:")
	t.Logf("  Insights: %s", ticketMem.Insights)
}

func TestAgentMemory_SaveAndSearch(t *testing.T) {
	repo := createTestRepository(t)
	llmClient := &mockLLMClient{}
	svc := memoryService.New(llmClient, repo)
	ctx := context.Background()

	// Create test memory
	mem1 := &memory.AgentMemory{
		ID:             types.NewAgentMemoryID(),
		AgentID:        "bigquery",
		TaskQuery:      "get login errors",
		QueryEmbedding: []float32{0.1, 0.2, 0.3},
		Timestamp:      time.Now(),
		Duration:       time.Second,
		Successes:      []string{"Successfully executed query"},
		Problems:       []string{},
		Improvements:   []string{},
	}

	mem2 := &memory.AgentMemory{
		ID:             types.NewAgentMemoryID(),
		AgentID:        "bigquery",
		TaskQuery:      "get authentication failures",
		QueryEmbedding: []float32{0.15, 0.25, 0.35},
		Timestamp:      time.Now(),
		Duration:       2 * time.Second,
		Problems:       []string{"Query timeout"},
		Improvements:   []string{"Add index"},
	}

	// Different agent
	mem3 := &memory.AgentMemory{
		ID:             types.NewAgentMemoryID(),
		AgentID:        "virustotal",
		TaskQuery:      "scan file hash",
		QueryEmbedding: []float32{0.5, 0.6, 0.7},
		Timestamp:      time.Now(),
		Duration:       time.Second,
	}

	// Save memories
	gt.NoError(t, svc.SaveAgentMemory(ctx, mem1))
	gt.NoError(t, svc.SaveAgentMemory(ctx, mem2))
	gt.NoError(t, svc.SaveAgentMemory(ctx, mem3))

	// Search by similar embedding (should find mem1 and mem2, not mem3)
	results, err := svc.SearchRelevantAgentMemories(ctx, "bigquery", "login errors", 2)
	gt.NoError(t, err)
	gt.V(t, len(results)).Equal(2)

	// Verify results are from correct agent
	for _, r := range results {
		gt.V(t, r.AgentID).Equal("bigquery")
	}
}
