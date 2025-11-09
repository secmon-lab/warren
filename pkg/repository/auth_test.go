package repository_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestToken(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

		// Create test token
		token := auth.NewToken("test-sub", "test@example.com", "Test User")

		// PutToken
		gt.NoError(t, repo.PutToken(ctx, token))

		// GetToken
		gotToken, err := repo.GetToken(ctx, token.ID)
		gt.NoError(t, err)
		gt.Value(t, gotToken.ID).Equal(token.ID)
		gt.Value(t, gotToken.Secret).Equal(token.Secret)
		gt.Value(t, gotToken.Sub).Equal(token.Sub)
		gt.Value(t, gotToken.Email).Equal(token.Email)
		gt.Value(t, gotToken.Name).Equal(token.Name)
		gt.Value(t, gotToken.ExpiresAt.Unix()).Equal(token.ExpiresAt.Unix())
		gt.Value(t, gotToken.CreatedAt.Unix()).Equal(token.CreatedAt.Unix())

		// Test token validation
		gt.NoError(t, gotToken.Validate())
		gt.Value(t, gotToken.IsExpired()).Equal(false)

		// DeleteToken
		gt.NoError(t, repo.DeleteToken(ctx, token.ID))

		// Verify token is deleted
		_, err = repo.GetToken(ctx, token.ID)
		gt.Error(t, err) // Should return error for non-existent token
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})

	t.Run("Firestore", func(t *testing.T) {
		repo := newFirestoreClient(t)
		testFn(t, repo)
	})
}
