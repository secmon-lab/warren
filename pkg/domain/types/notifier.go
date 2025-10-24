package types

import "context"

// ProcessNotifier is a function type for notifying process events throughout the pipeline
// Implementations should format messages appropriately for their output destination
type ProcessNotifier func(ctx context.Context, message string)
