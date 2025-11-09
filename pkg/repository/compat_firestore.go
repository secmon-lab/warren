package repository

import (
	"context"

	"github.com/secmon-lab/warren/pkg/repository/firestore"
)

// Firestore is a type alias for backward compatibility
type Firestore = firestore.Firestore

// NewFirestore creates a new Firestore repository client
func NewFirestore(ctx context.Context, projectID, databaseID string) (*Firestore, error) {
	return firestore.New(ctx, projectID, databaseID)
}
