package http

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/utils/authctx"
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

		authReq := &model.AuthHTTPRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   string(body),
			Header: r.Header,
		}

		ctx := authctx.WithHTTPRequest(r.Context(), authReq)
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
				handleError(w, r, goerr.Wrap(err, "failed to verify slack request", goerr.T(model.ErrTagInvalidRequest)))
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

		var msg model.SNSMessage
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

		// Verify SNS message signature
		if err := verifySNSMessageSignature(r.Context(), msg); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to verify SNS message signature", goerr.T(model.ErrTagInvalidRequest)))
			return
		}

		// Inject validated message into request context
		ctx := authctx.WithSNSMessage(r.Context(), &msg)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func handleSNSSubscriptionConfirmation(ctx context.Context, msg model.SNSMessage) error {
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

func verifySNSMessageSignature(ctx context.Context, msg model.SNSMessage) error {
	parsedURL, err := url.Parse(msg.SigningCertURL)
	if err != nil {
		return goerr.Wrap(err, "failed to parse signing cert URL", goerr.T(model.ErrTagInvalidRequest), goerr.V("url", msg.SigningCertURL))
	}

	// Check if the URL is from AWS SNS
	if !strings.HasPrefix(parsedURL.Host, "sns.") || !strings.HasSuffix(parsedURL.Host, ".amazonaws.com") || !strings.HasPrefix(parsedURL.Path, "/SimpleNotificationService-") {
		return goerr.New("invalid signing cert URL", goerr.T(model.ErrTagInvalidRequest), goerr.V("url", msg.SigningCertURL))
	}

	var client httpClient = http.DefaultClient
	if c, ok := ctx.Value(httpClientKey).(httpClient); ok {
		client = c
	}

	resp, err := client.Get(msg.SigningCertURL)
	if err != nil {
		return goerr.Wrap(err, "failed to get signing cert")
	}
	defer resp.Body.Close()

	certPEM, err := io.ReadAll(resp.Body)
	if err != nil {
		return goerr.Wrap(err, "failed to read cert")
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return goerr.New("failed to decode PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return goerr.Wrap(err, "failed to parse certificate")
	}

	rsaPublicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return goerr.New("certificate does not contain an RSA public key")
	}

	// Build the message string to verify according to AWS SNS spec
	var stringToSign strings.Builder

	// Common fields for all message types
	stringToSign.WriteString("Message\n")
	stringToSign.WriteString(msg.Message + "\n")
	stringToSign.WriteString("MessageId\n")
	stringToSign.WriteString(msg.MessageId + "\n")

	// Optional Subject field
	if msg.Subject != "" {
		stringToSign.WriteString("Subject\n")
		stringToSign.WriteString(msg.Subject + "\n")
	}

	// Type-specific fields
	if msg.Type == "SubscriptionConfirmation" || msg.Type == "UnsubscribeConfirmation" {
		stringToSign.WriteString("SubscribeURL\n")
		stringToSign.WriteString(msg.SubscribeURL + "\n")
		stringToSign.WriteString("Token\n")
		stringToSign.WriteString(msg.Token + "\n")
	}

	// Common fields for all message types
	stringToSign.WriteString("Timestamp\n")
	stringToSign.WriteString(msg.Timestamp + "\n")
	stringToSign.WriteString("TopicArn\n")
	stringToSign.WriteString(msg.TopicArn + "\n")
	stringToSign.WriteString("Type\n")
	stringToSign.WriteString(msg.Type + "\n")

	signature, err := base64.StdEncoding.DecodeString(msg.Signature)
	if err != nil {
		return goerr.Wrap(err, "failed to decode signature")
	}

	var alg x509.SignatureAlgorithm
	var hash crypto.Hash
	switch msg.SignatureVersion {
	case "1":
		alg = x509.SHA1WithRSA
		hash = crypto.SHA1
	case "2":
		alg = x509.SHA256WithRSA
		hash = crypto.SHA256
	default:
		return goerr.New("invalid signature version", goerr.T(model.ErrTagInvalidRequest), goerr.V("version", msg.SignatureVersion))
	}

	if err := cert.CheckSignature(alg, []byte(stringToSign.String()), signature); err != nil {
		return goerr.Wrap(err, "signature verification failed")
	}

	hashed := hash.New()
	hashed.Write([]byte(stringToSign.String()))
	digest := hashed.Sum(nil)

	if err := rsa.VerifyPKCS1v15(rsaPublicKey, hash, digest, signature); err != nil {
		return goerr.Wrap(err, "signature verification failed")
	}

	return nil
}

type httpClient interface {
	Get(url string) (*http.Response, error)
}

func withHTTPClient(ctx context.Context, client httpClient) context.Context {
	return context.WithValue(ctx, httpClientKey, client)
}
