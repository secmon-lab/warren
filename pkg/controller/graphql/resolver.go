package graphql

import (
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver serves as dependency injection point for the application.
type Resolver struct {
	repo interfaces.Repository
}

// NewResolver creates a new resolver instance.
func NewResolver(repo interfaces.Repository) *Resolver {
	return &Resolver{
		repo: repo,
	}
}
