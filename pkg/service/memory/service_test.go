package memory_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
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
	return nil, nil
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
