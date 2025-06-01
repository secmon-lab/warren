package http_test

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

type useCaseInterface struct {
	interfaces.AlertUsecases
	interfaces.SlackEventUsecases
	interfaces.SlackInteractionUsecases
	interfaces.UserUsecases
}

// GetUserIcon implements the UseCase interface
func (u *useCaseInterface) GetUserIcon(ctx context.Context, userID string) ([]byte, string, error) {
	// Mock implementation for testing
	return []byte("mock-icon-data"), "image/png", nil
}
