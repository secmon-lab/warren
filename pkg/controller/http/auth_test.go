package http_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	server "github.com/secmon-lab/warren/pkg/controller/http"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase"
)

func TestAuthEndpointsWithNoAuthn(t *testing.T) {
	// Setup server with NoAuthnUseCase
	repo := repository.NewMemory()
	noAuthnUC := usecase.NewNoAuthnUseCase(repo)
	uc := usecase.New()

	srv := server.New(uc, server.WithAuthUseCase(noAuthnUC))

	t.Run("GET /api/auth/me returns anonymous user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		gt.Equal(t, rec.Code, http.StatusOK)

		var user map[string]string
		err := json.NewDecoder(rec.Body).Decode(&user)
		gt.NoError(t, err)

		gt.Equal(t, user["sub"], auth.AnonymousUserID)
		gt.Equal(t, user["email"], auth.AnonymousUserEmail)
		gt.Equal(t, user["name"], auth.AnonymousUserName)
	})

	t.Run("GET /api/auth/login redirects to root", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		gt.Equal(t, rec.Code, http.StatusTemporaryRedirect)
		gt.Equal(t, rec.Header().Get("Location"), "/")
	})

	t.Run("POST /api/auth/logout redirects to root", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		gt.Equal(t, rec.Code, http.StatusOK)

		body, err := io.ReadAll(rec.Body)
		gt.NoError(t, err)
		gt.Equal(t, string(body), `{"success": true}`)
	})

	t.Run("GET /api/auth/callback redirects to root", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/callback?code=test&state=test", nil)
		// Set state cookie to pass validation
		req.AddCookie(&http.Cookie{
			Name:  "oauth_state",
			Value: "test",
		})
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		gt.Equal(t, rec.Code, http.StatusTemporaryRedirect)
		gt.Equal(t, rec.Header().Get("Location"), "/")

		// Should set authentication cookies
		cookies := rec.Result().Cookies()
		gt.True(t, len(cookies) >= 2) // token_id and token_secret
	})
}

func TestAuthMiddlewareWithNoAuthn(t *testing.T) {
	// Setup server with NoAuthnUseCase
	repo := repository.NewMemory()
	noAuthnUC := usecase.NewNoAuthnUseCase(repo)
	uc := usecase.New()

	// Enable GraphQL to test auth middleware
	srv := server.New(uc,
		server.WithAuthUseCase(noAuthnUC),
		server.WithGraphQLRepo(repo),
	)

	t.Run("GraphQL endpoint accessible without cookies", func(t *testing.T) {
		query := `{"query": "{ __typename }"}`
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(query))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		// Should not return 401 Unauthorized
		gt.True(t, rec.Code != http.StatusUnauthorized)
	})
}

func TestAuthWithRegularAuth(t *testing.T) {
	// Setup server with regular AuthUseCase
	repo := repository.NewMemory()
	authUC := usecase.NewAuthUseCase(repo, nil, "test-client-id", "test-client-secret", "http://localhost/api/auth/callback")
	uc := usecase.New()

	srv := server.New(uc, server.WithAuthUseCase(authUC))

	t.Run("GET /api/auth/me requires authentication", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		gt.Equal(t, rec.Code, http.StatusUnauthorized)

		body, err := io.ReadAll(rec.Body)
		gt.NoError(t, err)
		gt.True(t, strings.Contains(string(body), "Not authenticated"))
	})

	t.Run("GET /api/auth/login redirects to Slack OAuth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		gt.Equal(t, rec.Code, http.StatusTemporaryRedirect)
		location := rec.Header().Get("Location")
		gt.True(t, strings.Contains(location, "slack.com/openid/connect/authorize"))
		gt.True(t, strings.Contains(location, "client_id=test-client-id"))
	})

	t.Run("POST /api/auth/logout clears cookies", func(t *testing.T) {
		// First, set up a valid token
		token := auth.NewToken("U12345", "test@example.com", "Test User")
		err := repo.PutToken(context.Background(), token)
		gt.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		req.AddCookie(&http.Cookie{
			Name:  "token_id",
			Value: token.ID.String(),
		})
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		gt.Equal(t, rec.Code, http.StatusOK)

		// Check that cookies are cleared
		cookies := rec.Result().Cookies()
		for _, cookie := range cookies {
			if cookie.Name == "token_id" || cookie.Name == "token_secret" {
				gt.Equal(t, cookie.MaxAge, -1) // Cookie should be deleted
			}
		}
	})
}

func TestAuthMiddlewareProtection(t *testing.T) {
	// Setup server with regular AuthUseCase
	repo := repository.NewMemory()
	authUC := usecase.NewAuthUseCase(repo, nil, "test-client-id", "test-client-secret", "http://localhost/api/auth/callback")
	uc := usecase.New()

	// Enable GraphQL to test auth middleware
	srv := server.New(uc,
		server.WithAuthUseCase(authUC),
		server.WithGraphQLRepo(repo),
	)

	t.Run("GraphQL endpoint requires authentication", func(t *testing.T) {
		query := `{"query": "{ __typename }"}`
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(query))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		gt.Equal(t, rec.Code, http.StatusUnauthorized)

		body, err := io.ReadAll(rec.Body)
		gt.NoError(t, err)
		gt.True(t, strings.Contains(string(body), "Authentication required"))
	})

	t.Run("GraphQL endpoint accessible with valid token", func(t *testing.T) {
		// Create a valid token
		token := auth.NewToken("U12345", "test@example.com", "Test User")
		err := repo.PutToken(context.Background(), token)
		gt.NoError(t, err)

		query := `{"query": "{ __typename }"}`
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(query))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "token_id",
			Value: token.ID.String(),
		})
		req.AddCookie(&http.Cookie{
			Name:  "token_secret",
			Value: token.Secret.String(),
		})
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		// Should not return 401 Unauthorized
		gt.True(t, rec.Code != http.StatusUnauthorized)
	})
}

// TestE2ENoAuthnFlow tests the complete flow with no-authentication mode
func TestE2ENoAuthnFlow(t *testing.T) {
	// Setup server with NoAuthnUseCase
	repo := repository.NewMemory()
	noAuthnUC := usecase.NewNoAuthnUseCase(repo)
	uc := usecase.New()

	srv := server.New(uc,
		server.WithAuthUseCase(noAuthnUC),
		server.WithGraphQLRepo(repo), // Enable GraphQL for testing
	)

	// Create test server
	ts := httptest.NewServer(srv)
	defer ts.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	t.Run("Complete anonymous user flow", func(t *testing.T) {
		// Step 1: Access /api/auth/me - should return anonymous user
		resp, err := client.Get(ts.URL + "/api/auth/me")
		gt.NoError(t, err)
		defer resp.Body.Close()

		gt.Equal(t, resp.StatusCode, http.StatusOK)

		var user map[string]string
		err = json.NewDecoder(resp.Body).Decode(&user)
		gt.NoError(t, err)

		gt.Equal(t, user["sub"], auth.AnonymousUserID)
		gt.Equal(t, user["name"], auth.AnonymousUserName)

		// Step 2: Access GraphQL endpoint - should work without authentication
		query := `{"query": "{ __typename }"}`
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/graphql", strings.NewReader(query))
		gt.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp2, err := client.Do(req)
		gt.NoError(t, err)
		defer resp2.Body.Close()

		// Should not be 401
		gt.True(t, resp2.StatusCode != http.StatusUnauthorized)

		// Step 3: Try to login - should redirect to root
		resp3, err := client.Get(ts.URL + "/api/auth/login")
		gt.NoError(t, err)
		defer resp3.Body.Close()

		gt.Equal(t, resp3.StatusCode, http.StatusTemporaryRedirect)
		gt.Equal(t, resp3.Header.Get("Location"), "/")

		// Step 4: Logout - should complete successfully
		req4, err := http.NewRequest(http.MethodPost, ts.URL+"/api/auth/logout", nil)
		gt.NoError(t, err)

		resp4, err := client.Do(req4)
		gt.NoError(t, err)
		defer resp4.Body.Close()

		gt.Equal(t, resp4.StatusCode, http.StatusOK)

		body, err := io.ReadAll(resp4.Body)
		gt.NoError(t, err)
		gt.Equal(t, string(body), `{"success": true}`)
	})
}

// TestE2ERegularAuthFlow tests that regular authentication still works correctly
func TestE2ERegularAuthFlow(t *testing.T) {
	// Setup server with regular AuthUseCase
	repo := repository.NewMemory()
	authUC := usecase.NewAuthUseCase(repo, nil, "test-client-id", "test-client-secret", "http://localhost/api/auth/callback")
	uc := usecase.New()

	srv := server.New(uc,
		server.WithAuthUseCase(authUC),
		server.WithGraphQLRepo(repo),
	)

	// Create test server
	ts := httptest.NewServer(srv)
	defer ts.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	t.Run("Regular authentication flow protection", func(t *testing.T) {
		// Step 1: Access /api/auth/me without auth - should fail
		resp, err := client.Get(ts.URL + "/api/auth/me")
		gt.NoError(t, err)
		defer resp.Body.Close()

		gt.Equal(t, resp.StatusCode, http.StatusUnauthorized)

		// Step 2: Access GraphQL without auth - should fail
		query := `{"query": "{ __typename }"}`
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/graphql", strings.NewReader(query))
		gt.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp2, err := client.Do(req)
		gt.NoError(t, err)
		defer resp2.Body.Close()

		gt.Equal(t, resp2.StatusCode, http.StatusUnauthorized)

		// Step 3: Login redirects to Slack
		resp3, err := client.Get(ts.URL + "/api/auth/login")
		gt.NoError(t, err)
		defer resp3.Body.Close()

		gt.Equal(t, resp3.StatusCode, http.StatusTemporaryRedirect)
		location := resp3.Header.Get("Location")
		gt.True(t, strings.Contains(location, "slack.com"))

		// Step 4: Simulate authenticated request
		token := auth.NewToken("U12345", "test@example.com", "Test User")
		err = repo.PutToken(context.Background(), token)
		gt.NoError(t, err)

		// Create request with auth cookies
		req5, err := http.NewRequest(http.MethodGet, ts.URL+"/api/auth/me", nil)
		gt.NoError(t, err)
		req5.AddCookie(&http.Cookie{
			Name:  "token_id",
			Value: token.ID.String(),
		})
		req5.AddCookie(&http.Cookie{
			Name:  "token_secret",
			Value: token.Secret.String(),
		})

		resp5, err := client.Do(req5)
		gt.NoError(t, err)
		defer resp5.Body.Close()

		gt.Equal(t, resp5.StatusCode, http.StatusOK)

		var user map[string]string
		err = json.NewDecoder(resp5.Body).Decode(&user)
		gt.NoError(t, err)

		gt.Equal(t, user["sub"], "U12345")
		gt.Equal(t, user["email"], "test@example.com")
		gt.Equal(t, user["name"], "Test User")
	})
}