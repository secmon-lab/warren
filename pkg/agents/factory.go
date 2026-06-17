package agents

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/urfave/cli/v3"
)

// ToolSetFactory is the interface that all agent packages must implement.
// This interface provides a unified way for the CLI layer to interact with agents.
type ToolSetFactory interface {
	// Flags returns CLI flags for this agent
	Flags() []cli.Flag

	// Configure creates and initializes the agent, returning a ToolSet.
	// Returns (nil, nil) if the agent is not configured.
	Configure(ctx context.Context) (interfaces.ToolSet, error)
}

// StorageAware is an optional interface for factories that need the
// warren-wide storage client and prefix (e.g. for snapshotting large
// result sets to shared object storage). ConfigureAll injects these
// before calling Configure. Factories that do not need storage simply
// omit this interface.
type StorageAware interface {
	SetStorage(client interfaces.StorageClient, prefix string)
}
