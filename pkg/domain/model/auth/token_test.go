package auth_test

import (
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
)

func TestAnonymousUser(t *testing.T) {
	t.Run("NewAnonymousUser creates valid anonymous token", func(t *testing.T) {
		token := auth.NewAnonymousUser()

		gt.NoError(t, token.Validate())
		gt.Equal(t, token.Sub, auth.AnonymousUserID)
		gt.Equal(t, token.Email, auth.AnonymousUserEmail)
		gt.Equal(t, token.Name, auth.AnonymousUserName)
		gt.False(t, token.IsExpired())
		gt.True(t, token.IsAnonymous())
	})

	t.Run("Regular user is not anonymous", func(t *testing.T) {
		token := auth.NewToken("U12345", "user@example.com", "Test User")

		gt.NoError(t, token.Validate())
		gt.False(t, token.IsAnonymous())
	})

	t.Run("Anonymous user constants", func(t *testing.T) {
		// Verify that anonymous user ID doesn't start with U (Slack user ID pattern)
		gt.True(t, auth.AnonymousUserID[0] != 'U')
		gt.Equal(t, auth.AnonymousUserID, "anonymous")
		gt.Equal(t, auth.AnonymousUserName, "Anonymous")
		gt.Equal(t, auth.AnonymousUserEmail, "anonymous@localhost")
	})

	t.Run("Anonymous token expiration", func(t *testing.T) {
		token := auth.NewAnonymousUser()

		// Token should have valid expiration time
		gt.True(t, token.ExpiresAt.After(time.Now()))
		gt.True(t, token.CreatedAt.Before(time.Now().Add(time.Second)))

		// Token should expire after TokenExpireDuration
		expectedExpiration := token.CreatedAt.Add(auth.TokenExpireDuration)
		gt.True(t, token.ExpiresAt.Equal(expectedExpiration) || token.ExpiresAt.After(expectedExpiration.Add(-time.Second)))
	})
}
