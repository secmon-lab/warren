package http_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/harlog"
	server "github.com/secmon-lab/warren/pkg/controller/http"
	"github.com/secmon-lab/warren/pkg/model"
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
	var snsMessage model.SNSMessage
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
	var snsMessage model.SNSMessage
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
