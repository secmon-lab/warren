package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/message"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/secmon-lab/warren/pkg/utils/user"
	"google.golang.org/api/idtoken"
)

type contextKey string

const (
	httpClientKey contextKey = "http_client"
)

type SNSMessage struct {
	Type             string    `json:"Type"`
	MessageId        string    `json:"MessageId"`
	Token            string    `json:"Token"`
	TopicArn         string    `json:"TopicArn"`
	Message          string    `json:"Message"`
	Timestamp        time.Time `json:"Timestamp"`
	SignatureVersion string    `json:"SignatureVersion"`
	Signature        string    `json:"Signature"`
	SigningCertURL   string    `json:"SigningCertURL"`
	SubscribeURL     string    `json:"SubscribeURL"`
}

func withAuthHTTPRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		// Restore the body for next handlers
		r.Body = io.NopCloser(bytes.NewBuffer(body))

		copiedHeader := make(map[string][]string)
		for k, v := range r.Header {
			copiedHeader[k] = v[:]
		}

		authReq := &auth.HTTPRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   string(body),
			Header: copiedHeader,
		}

		ctx := auth.WithHTTPRequest(r.Context(), authReq)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validateGoogleIAPToken validates Google IAP JWT from x-goog-iap-jwt-assertion header
// and injects the verified claims into request context if valid
// If validation fails, it logs a warning and continues processing
func validateGoogleIAPToken(next http.Handler) http.Handler {
	return validateGoogleIAPTokenWithJWKURL(next, "https://www.gstatic.com/iap/verify/public_key-jwk")
}

// validateGoogleIAPTokenWithJWKURL validates Google IAP JWT with a configurable JWK URL
func validateGoogleIAPTokenWithJWKURL(next http.Handler, jwkURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		iapJWTHeader := r.Header.Get("x-goog-iap-jwt-assertion")
		if iapJWTHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Fetch IAP public keys
		keySet, err := jwk.Fetch(r.Context(), jwkURL)
		if err != nil {
			logging.From(r.Context()).Warn("failed to fetch IAP public keys, continuing without validation", "error", err)
			next.ServeHTTP(w, r)
			return
		}

		// Parse and verify the JWT token
		token, err := jwt.Parse([]byte(iapJWTHeader), jwt.WithKeySet(keySet), jwt.WithValidate(true))
		if err != nil {
			logging.From(r.Context()).Warn("invalid IAP JWT token, continuing without validation", "error", err)
			next.ServeHTTP(w, r)
			return
		}

		// Verify issuer
		if token.Issuer() != "https://cloud.google.com/iap" {
			logging.From(r.Context()).Warn("invalid JWT issuer, continuing without validation", "issuer", token.Issuer())
			next.ServeHTTP(w, r)
			return
		}

		// Verify expiration and issued-at time
		now := time.Now()
		if token.Expiration().Before(now) {
			logging.From(r.Context()).Warn("JWT token expired, continuing without validation", "expiration", token.Expiration(), "now", now)
			next.ServeHTTP(w, r)
			return
		}

		if token.IssuedAt().After(now) {
			logging.From(r.Context()).Warn("JWT token used before issued, continuing without validation", "issued_at", token.IssuedAt(), "now", now)
			next.ServeHTTP(w, r)
			return
		}

		// Verify audience format (should be /projects/{project_number}/apps/{project_id})
		aud := token.Audience()
		if len(aud) == 0 {
			logging.From(r.Context()).Warn("JWT missing audience, continuing without validation")
			next.ServeHTTP(w, r)
			return
		}

		// Extract claims as map for context
		claimsMap := make(map[string]interface{})
		for iter := token.Iterate(r.Context()); iter.Next(r.Context()); {
			pair := iter.Pair()
			claimsMap[pair.Key.(string)] = pair.Value
		}

		// Log successful validation for debugging
		logging.From(r.Context()).Info("IAP JWT validated successfully",
			"sub", token.Subject(),
			"email", claimsMap["email"],
			"aud", aud[0])

		// Inject validated claims into request context
		ctx := auth.WithGoogleIAPJWTClaims(r.Context(), claimsMap)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

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
		ctx := auth.WithGoogleIDTokenClaims(r.Context(), payload.Claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func verifySlackRequest(verifier slack.PayloadVerifier) func(http.Handler) http.Handler {
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
				handleError(w, r, goerr.Wrap(err, "failed to verify slack request", goerr.T(errs.TagInvalidRequest)))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getDetailedStackTrace returns a detailed stack trace with function names and line numbers
func getDetailedStackTrace() string {
	var buf strings.Builder
	buf.WriteString("Detailed Stack Trace:\n")

	// Get callers (skip the first few frames that are in the panic recovery code)
	callers := make([]uintptr, 64)
	n := runtime.Callers(3, callers)
	frames := runtime.CallersFrames(callers[:n])

	for {
		frame, more := frames.Next()
		buf.WriteString(fmt.Sprintf("  %s\n    %s:%d\n", frame.Function, frame.File, frame.Line))
		if !more {
			break
		}
	}

	return buf.String()
}

func panicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Get both debug.Stack() and detailed stack trace
				debugStack := debug.Stack()
				detailedStack := getDetailedStackTrace()

				// Create error with stack trace information
				panicErr := goerr.New("panic recovered",
					goerr.V("panic", fmt.Sprintf("%v", err)),
					goerr.V("debug_stack", string(debugStack)),
					goerr.V("detailed_stack", detailedStack),
					goerr.V("method", r.Method),
					goerr.V("path", r.URL.Path),
				)

				handleError(w, r, panicErr)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// verifySNSRequest validates AWS SNS HTTP request
// and injects the message into request context if valid
func verifySNSRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to read request body"))
			return
		}
		r.Body = io.NopCloser(bytes.NewBuffer(body))

		var msg message.SNS
		if err := json.Unmarshal(body, &msg); err != nil {
			next.ServeHTTP(w, r) // Not SNS message, pass through
			return
		}

		// Handle subscription confirmation
		if msg.Type == "SubscriptionConfirmation" {
			if err := handleSNSSubscriptionConfirmation(r.Context(), msg); err != nil {
				handleError(w, r, goerr.Wrap(err, "failed to handle SNS subscription confirmation"))
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		var client httpClient = http.DefaultClient
		if c, ok := r.Context().Value(httpClientKey).(httpClient); ok {
			client = c
		}

		// Verify SNS message signature
		if err := msg.Verify(r.Context(), client); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to verify SNS message", goerr.T(errs.TagInvalidRequest)))
			return
		}

		// Inject validated message into request context
		ctx := auth.WithSNSMessage(r.Context(), &msg)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func handleSNSSubscriptionConfirmation(ctx context.Context, msg message.SNS) error {
	logger := logging.From(ctx)

	logger.Info("handling SNS subscription confirmation", "msg", msg)

	var client httpClient = http.DefaultClient
	if c, ok := ctx.Value(httpClientKey).(httpClient); ok {
		client = c
	}

	resp, err := client.Get(msg.SubscribeURL)
	if err != nil {
		return goerr.Wrap(err, "failed to access SubscribeURL")
	}
	defer safe.Close(ctx, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return goerr.New("failed to confirm subscription", goerr.V("status", resp.StatusCode))
	}

	logger.Info("SNS subscription confirmed")

	return nil
}

type httpClient interface {
	Get(url string) (*http.Response, error)
}

func withHTTPClient(ctx context.Context, client httpClient) context.Context {
	return context.WithValue(ctx, httpClientKey, client)
}

// authMiddleware validates authentication for GraphQL requests
func authMiddleware(authUC AuthUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// For NoAuthn mode, always use anonymous user
			if authUC.IsNoAuthn() {
				token := auth.NewAnonymousUser()
				ctx := auth.ContextWithToken(r.Context(), token)
				ctx = user.WithUserID(ctx, token.Sub)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Get tokens from cookies
			tokenIDCookie, err := r.Cookie("token_id")
			if err != nil {
				http.Error(w, `{"errors": [{"message": "Authentication required"}]}`, http.StatusUnauthorized)
				return
			}

			tokenSecretCookie, err := r.Cookie("token_secret")
			if err != nil {
				http.Error(w, `{"errors": [{"message": "Authentication required"}]}`, http.StatusUnauthorized)
				return
			}

			tokenID := auth.TokenID(tokenIDCookie.Value)
			tokenSecret := auth.TokenSecret(tokenSecretCookie.Value)

			// Validate token
			token, err := authUC.ValidateToken(r.Context(), tokenID, tokenSecret)
			if err != nil {
				http.Error(w, `{"errors": [{"message": "Invalid authentication token"}]}`, http.StatusUnauthorized)
				return
			}

			// Add user context to request with Slack User ID
			ctx := auth.ContextWithToken(r.Context(), token)
			ctx = user.WithUserID(ctx, token.Sub) // Use Slack User ID
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func authorizeWithPolicy(policy interfaces.PolicyClient, noAuthorization bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Bypass authorization check if --no-authorization flag is set
			if noAuthorization {
				logging.From(r.Context()).Debug("authorization check bypassed due to --no-authorization flag")
				next.ServeHTTP(w, r)
				return
			}

			if policy == nil {
				next.ServeHTTP(w, r)
				return
			}

			var result struct {
				Allow bool `json:"allow"`
			}

			ctx := r.Context()
			authCtx := auth.BuildContext(ctx)
			if err := policy.Query(ctx, "data.auth", authCtx, &result); err != nil {
				handleError(w, r, goerr.Wrap(err, "failed to authorize request"))
				return
			}

			logging.From(ctx).Debug("authorization result", "input", authCtx, "output", result)

			if !result.Allow {
				logging.From(ctx).Warn("authorization failed", "auth", authCtx)
				http.Error(w, `Authorization failed. Check your policy.`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthorizeWithPolicyForTest is exported for testing purposes
func AuthorizeWithPolicyForTest(policy interfaces.PolicyClient, noAuthorization bool) func(http.Handler) http.Handler {
	return authorizeWithPolicy(policy, noAuthorization)
}
