package slack

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func (a *Agent) saveExecutionMemory(
	ctx context.Context,
	query string,
	resp *gollem.ExecuteResponse,
	execErr error,
	duration time.Duration,
	session gollem.Session,
) error {
	if a.memoryService == nil {
		return nil // Memory service not available
	}

	mem := &memory.AgentMemory{
		ID:        types.NewAgentMemoryID(),
		AgentID:   a.ID(),
		TaskQuery: query,
		Timestamp: time.Now(),
		Duration:  duration,
	}

	// Add result to successes or problems
	if execErr != nil {
		mem.Problems = []string{"error: " + execErr.Error()}
	} else if resp != nil && !resp.IsEmpty() {
		mem.Successes = []string{resp.String()}
	}

	if err := a.memoryService.SaveAgentMemory(ctx, mem); err != nil {
		return goerr.Wrap(err, "failed to save agent memory")
	}

	return nil
}
