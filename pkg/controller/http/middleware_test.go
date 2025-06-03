package http_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/harlog"
	server "github.com/secmon-lab/warren/pkg/controller/http"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/message"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

type mockHTTPClient struct {
	GetFunc func(url string) (*http.Response, error)
}

func (m *mockHTTPClient) Get(url string) (*http.Response, error) {
	return m.GetFunc(url)
}

func TestVerifySNSRequest(t *testing.T) {
	logs, err := harlog.ParseHARData(snsHar)
	gt.NoError(t, err)
	gt.A(t, logs).Length(1)

	log := logs[0]
	var snsMessage message.SNS
	bodyData, err := io.ReadAll(log.Request.Body)
	gt.NoError(t, err)
	err = json.Unmarshal(bodyData, &snsMessage)
	gt.NoError(t, err)

	t.Run("invalid signing cert URL", func(t *testing.T) {
		invalidMessage := snsMessage
		invalidMessage.SigningCertURL = "https://example.com/SimpleNotificationService-9c6465fa7f48f5cacd23014631ec1136.pem"

		body, err := json.Marshal(invalidMessage)
		gt.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/alert/sns", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := server.VerifySNSRequest(handler)
		middleware.ServeHTTP(rec, req)

		gt.Value(t, rec.Code).Equal(http.StatusBadRequest)
	})

	t.Run("subscription confirmation", func(t *testing.T) {
		certPath := filepath.Join("testdata", "sns.pem")
		certData, err := os.ReadFile(certPath)
		gt.NoError(t, err)

		mockClient := &mockHTTPClient{
			GetFunc: func(url string) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(certData)),
				}, nil
			},
		}

		confirmMessage := snsMessage
		confirmMessage.Type = "SubscriptionConfirmation"
		confirmMessage.SubscribeURL = "https://sns.ap-northeast-1.amazonaws.com/subscribe"

		body, err := json.Marshal(confirmMessage)
		gt.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/alert/sns", bytes.NewReader(body))
		req = req.WithContext(server.WithHTTPClient(req.Context(), mockClient))
		rec := httptest.NewRecorder()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := server.VerifySNSRequest(handler)
		middleware.ServeHTTP(rec, req)

		gt.Value(t, rec.Code).Equal(http.StatusOK)
	})

	t.Run("non SNS message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/alert/sns", bytes.NewReader([]byte("not a SNS message")))
		rec := httptest.NewRecorder()

		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		middleware := server.VerifySNSRequest(handler)
		middleware.ServeHTTP(rec, req)

		gt.Value(t, called).Equal(true)
		gt.Value(t, rec.Code).Equal(http.StatusOK)
	})

	t.Run("with valid SNS message", func(t *testing.T) {
		certPath := filepath.Join("testdata", "sns.pem")
		certData, err := os.ReadFile(certPath)
		gt.NoError(t, err)

		mockClient := &mockHTTPClient{
			GetFunc: func(url string) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(certData)),
				}, nil
			},
		}

		body, err := json.Marshal(snsMessage)
		gt.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/alert/sns", bytes.NewReader(body))
		req = req.WithContext(server.WithHTTPClient(req.Context(), mockClient))
		rec := httptest.NewRecorder()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := server.VerifySNSRequest(handler)
		middleware.ServeHTTP(rec, req)

		gt.Value(t, rec.Code).Equal(http.StatusOK)
	})
}

func TestPanicRecoveryMiddleware(t *testing.T) {
	t.Run("recover from panic", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(server.PanicRecoveryMiddleware)

		r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		req := httptest.NewRequest(http.MethodGet, "/panic", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		gt.Value(t, rec.Code).Equal(http.StatusInternalServerError)
	})
}

func TestLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer
	r := chi.NewRouter()
	r.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := logging.New(&buf, slog.LevelDebug, logging.FormatJSON, false)
			h.ServeHTTP(w, r.WithContext(logging.With(r.Context(), logger)))
		})
	})
	r.Use(server.LoggingMiddleware)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer test_token")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	gt.S(t, buf.String()).Contains(`"method":"GET"`)
	gt.S(t, buf.String()).Contains(`"path":"/"`)
	gt.S(t, buf.String()).Contains(`"status":200`)
	gt.S(t, buf.String()).NotContains(`test_token`)
	gt.S(t, buf.String()).NotContains(`"REDACTED"`)
}

func TestAlertSNSVerification(t *testing.T) {
	logs, err := harlog.ParseHARData(snsHar)
	gt.NoError(t, err)
	gt.A(t, logs).Length(1)

	log := logs[0]
	var snsMessage message.SNS
	bodyData, err := io.ReadAll(log.Request.Body)
	gt.NoError(t, err)
	err = json.Unmarshal(bodyData, &snsMessage)
	gt.NoError(t, err)

	t.Run("with valid SNS message", func(t *testing.T) {
		certPath := filepath.Join("testdata", "sns.pem")
		certData, err := os.ReadFile(certPath)
		gt.NoError(t, err)

		mockClient := &mockHTTPClient{
			GetFunc: func(url string) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(certData)),
				}, nil
			},
		}

		body, err := json.Marshal(snsMessage)
		gt.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/alert/sns", bytes.NewReader(body))
		req = req.WithContext(server.WithHTTPClient(req.Context(), mockClient))
		rec := httptest.NewRecorder()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := server.VerifySNSRequest(handler)
		middleware.ServeHTTP(rec, req)

		gt.Value(t, rec.Code).Equal(http.StatusOK)
	})
}

func TestValidateGoogleIAPToken(t *testing.T) {
	t.Run("no IAP header - should pass through", func(t *testing.T) {
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			// Verify no IAP claims in context
			claims, err := auth.GetGoogleIAPJWTClaims(r.Context())
			gt.Error(t, err) // Should error since no IAP header was provided
			gt.Value(t, claims).Equal(nil)
			w.WriteHeader(http.StatusOK)
		})

		middleware := server.ValidateGoogleIAPToken(handler)
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		gt.Value(t, called).Equal(true)
		gt.Value(t, w.Code).Equal(http.StatusOK)
	})

	t.Run("invalid IAP token - should log warning and continue", func(t *testing.T) {
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			// Verify no IAP claims in context since validation failed
			claims, err := auth.GetGoogleIAPJWTClaims(r.Context())
			gt.Error(t, err) // Should error since validation failed
			gt.Value(t, claims).Equal(nil)
			w.WriteHeader(http.StatusOK)
		})

		middleware := server.ValidateGoogleIAPToken(handler)
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("x-goog-iap-jwt-assertion", "invalid.jwt.token")
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		// Handler should be called even with invalid token (now logs warning and continues)
		gt.Value(t, called).Equal(true)
		gt.Value(t, w.Code).Equal(http.StatusOK)
	})

	t.Run("malformed IAP token - should log warning and continue", func(t *testing.T) {
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			// Verify no IAP claims in context since validation failed
			claims, err := auth.GetGoogleIAPJWTClaims(r.Context())
			gt.Error(t, err) // Should error since validation failed
			gt.Value(t, claims).Equal(nil)
			w.WriteHeader(http.StatusOK)
		})

		middleware := server.ValidateGoogleIAPToken(handler)
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("x-goog-iap-jwt-assertion", "not-a-jwt-token")
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		// Handler should be called even with malformed token (now logs warning and continues)
		gt.Value(t, called).Equal(true)
		gt.Value(t, w.Code).Equal(http.StatusOK)
	})

	t.Run("valid IAP token - should validate and inject claims", func(t *testing.T) {
		// Generate ECDSA key pair for ES256 algorithm
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		gt.NoError(t, err)

		// Create JWK from the private key
		privKey, err := jwk.FromRaw(privateKey)
		gt.NoError(t, err)

		// Set key ID and algorithm
		err = privKey.Set(jwk.KeyIDKey, "test-key-id")
		gt.NoError(t, err)
		err = privKey.Set(jwk.AlgorithmKey, jwa.ES256)
		gt.NoError(t, err)

		// Create public key for the key set
		pubKey, err := privKey.PublicKey()
		gt.NoError(t, err)

		// Create JWK Set
		keySet := jwk.NewSet()
		err = keySet.AddKey(pubKey)
		gt.NoError(t, err)

		// Create JWT token with valid IAP claims
		now := time.Now()
		token := jwt.New()

		// Set required claims for Google IAP
		err = token.Set(jwt.IssuerKey, "https://cloud.google.com/iap")
		gt.NoError(t, err)
		err = token.Set(jwt.AudienceKey, "/projects/123456789/apps/test-project")
		gt.NoError(t, err)
		err = token.Set(jwt.SubjectKey, "user-123")
		gt.NoError(t, err)
		err = token.Set(jwt.IssuedAtKey, now)
		gt.NoError(t, err)
		err = token.Set(jwt.ExpirationKey, now.Add(time.Hour))
		gt.NoError(t, err)

		// Add custom claims
		err = token.Set("email", "test@example.com")
		gt.NoError(t, err)
		err = token.Set("google", map[string]interface{}{
			"compute_engine": map[string]interface{}{
				"project_id": "test-project",
			},
		})
		gt.NoError(t, err)

		// Sign the token
		signedToken, err := jwt.Sign(token, jwt.WithKey(jwa.ES256, privKey))
		gt.NoError(t, err)

		// Mock the Google JWK endpoint by creating a test server
		// that serves our test keys at the expected Google URL path
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			keySetJSON, err := json.Marshal(keySet)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(keySetJSON)
		}))
		defer testServer.Close()

		called := false

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true

			// Verify IAP claims are available in context
			claims, err := auth.GetGoogleIAPJWTClaims(r.Context())
			gt.NoError(t, err)
			gt.Value(t, claims).NotEqual(nil)

			// Verify specific claims
			gt.Value(t, claims["iss"]).Equal("https://cloud.google.com/iap")
			gt.Value(t, claims["aud"]).Equal([]string{"/projects/123456789/apps/test-project"})
			gt.Value(t, claims["sub"]).Equal("user-123")
			gt.Value(t, claims["email"]).Equal("test@example.com")

			// Verify google claim structure
			googleClaim, ok := claims["google"].(map[string]interface{})
			gt.Value(t, ok).Equal(true)
			computeEngine, ok := googleClaim["compute_engine"].(map[string]interface{})
			gt.Value(t, ok).Equal(true)
			gt.Value(t, computeEngine["project_id"]).Equal("test-project")

			w.WriteHeader(http.StatusOK)
		})

		// Use the actual exported function with our test JWK URL
		middleware := server.ValidateGoogleIAPTokenWithJWKURL(handler, testServer.URL)

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("x-goog-iap-jwt-assertion", string(signedToken))
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		gt.Value(t, called).Equal(true)
		gt.Value(t, w.Code).Equal(http.StatusOK)
	})
}
