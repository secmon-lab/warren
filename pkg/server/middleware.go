package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/authctx"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"google.golang.org/api/idtoken"
)

type contextKey string

const (
	GoogleIDTokenClaimsKey contextKey = "google_id_token_claims"
)

// validateGoogleIDToken validates Google ID token in Authorization header
// and injects the claims into request context if valid
func validateGoogleIDToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Validate token
		validator, err := idtoken.NewValidator(r.Context())
		if err != nil {
			http.Error(w, "Failed to create token validator", http.StatusInternalServerError)
			return
		}

		payload, err := validator.Validate(r.Context(), token, "")
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Inject validated claims into request context
		ctx := authctx.WithGoogleIDTokenClaims(r.Context(), payload.Claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetGoogleIDTokenClaims retrieves Google ID token claims from context
func GetGoogleIDTokenClaims(ctx context.Context) (map[string]interface{}, error) {
	claims, ok := ctx.Value(GoogleIDTokenClaimsKey).(map[string]interface{})
	if !ok {
		return nil, goerr.New("Google ID token claims not found in context")
	}
	return claims, nil
}

func verifySlackRequest(verifier interfaces.SlackPayloadVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if verifier == nil {
				logging.From(r.Context()).Warn("slack signing secret is not set, skipping verification")
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				handleError(w, r, goerr.Wrap(err, "failed to read request body"))
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(body))

			if err := verifier(r.Context(), r.Header, body); err != nil {
				handleError(w, r, goerr.Wrap(err, "failed to verify slack request", goerr.T(errBadRequest)))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func panicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				handleError(w, r, goerr.New(fmt.Sprintf("%v", err)))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
