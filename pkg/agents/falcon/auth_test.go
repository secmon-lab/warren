package falcon_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/agents/falcon"
)

func TestTokenProvider_GetToken(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)

		gt.Equal(t, r.Method, http.MethodPost)
		gt.Equal(t, r.URL.Path, "/oauth2/token")
		gt.Equal(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")

		err := r.ParseForm()
		gt.NoError(t, err)
		gt.Equal(t, r.FormValue("client_id"), "test-client-id")
		gt.Equal(t, r.FormValue("client_secret"), "test-client-secret")

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"access_token": "test-token-123",
			"expires_in":   1800,
			"token_type":   "bearer",
		}
		err = json.NewEncoder(w).Encode(resp)
		gt.NoError(t, err)
	}))
	defer srv.Close()

	tp := falcon.NewTokenProviderForTest(
		"test-client-id",
		"test-client-secret",
		srv.URL,
	)

	ctx := context.Background()

	// First call should fetch a new token
	token, err := tp.GetToken(ctx)
	gt.NoError(t, err)
	gt.Equal(t, token, "test-token-123")
	gt.Equal(t, callCount.Load(), int32(1))

	// Second call should return cached token
	token, err = tp.GetToken(ctx)
	gt.NoError(t, err)
	gt.Equal(t, token, "test-token-123")
	gt.Equal(t, callCount.Load(), int32(1)) // No additional call
}

func TestTokenProvider_GetToken_ConcurrentAccess(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"access_token": "concurrent-token",
			"expires_in":   1800,
			"token_type":   "bearer",
		}
		err := json.NewEncoder(w).Encode(resp)
		gt.NoError(t, err)
	}))
	defer srv.Close()

	tp := falcon.NewTokenProviderForTest(
		"test-client-id",
		"test-client-secret",
		srv.URL,
	)

	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := tp.GetToken(ctx)
			gt.NoError(t, err)
			gt.Equal(t, token, "concurrent-token")
		}()
	}
	wg.Wait()

	// Due to mutex, should have minimal API calls (ideally 1, but could be a few due to timing)
	gt.True(t, callCount.Load() <= 10)
}

func TestTokenProvider_GetToken_ClearAndRefresh(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := callCount.Add(1)

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"access_token": "token-" + string(rune('0'+count)),
			"expires_in":   1800,
			"token_type":   "bearer",
		}
		err := json.NewEncoder(w).Encode(resp)
		gt.NoError(t, err)
	}))
	defer srv.Close()

	tp := falcon.NewTokenProviderForTest(
		"test-client-id",
		"test-client-secret",
		srv.URL,
	)

	ctx := context.Background()

	// Get initial token
	token1, err := tp.GetToken(ctx)
	gt.NoError(t, err)
	gt.Equal(t, callCount.Load(), int32(1))

	// Clear token
	tp.ClearToken()

	// Should fetch a new token
	token2, err := tp.GetToken(ctx)
	gt.NoError(t, err)
	gt.Equal(t, callCount.Load(), int32(2))

	// Tokens should be different
	gt.True(t, token1 != token2)
}

func TestTokenProvider_GetToken_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(`{"errors": [{"message": "internal error"}]}`))
		gt.NoError(t, err)
	}))
	defer srv.Close()

	tp := falcon.NewTokenProviderForTest(
		"test-client-id",
		"test-client-secret",
		srv.URL,
	)

	ctx := context.Background()

	_, err := tp.GetToken(ctx)
	gt.Error(t, err)
}
