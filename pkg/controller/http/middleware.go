package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/message"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"google.golang.org/api/idtoken"
)

type contextKey string

const (
	GoogleIDTokenClaimsKey contextKey = "google_id_token_claims"
	httpClientKey          contextKey = "http_client"
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

// GetGoogleIDTokenClaims retrieves Google ID token claims from context
func GetGoogleIDTokenClaims(ctx context.Context) (map[string]interface{}, error) {
	claims, ok := ctx.Value(GoogleIDTokenClaimsKey).(map[string]interface{})
	if !ok {
		return nil, goerr.New("Google ID token claims not found in context")
	}
	return claims, nil
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
	defer resp.Body.Close()

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
