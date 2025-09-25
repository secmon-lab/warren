package http_test

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/harlog"
	server "github.com/secmon-lab/warren/pkg/controller/http"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/message"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

//go:embed testdata/sns.har
var snsHar []byte

func TestAlertSNSHandler(t *testing.T) {
	logs, err := harlog.ParseHARData(snsHar)
	gt.NoError(t, err)
	gt.A(t, logs).Length(1)

	log := logs[0]
	var snsMessage message.SNS
	bodyData, err := io.ReadAll(log.Request.Body)
	gt.NoError(t, err)
	err = json.Unmarshal(bodyData, &snsMessage)
	gt.NoError(t, err)

	t.Run("successful alert handling", func(t *testing.T) {
		alertMock := &mock.AlertUsecasesMock{
			HandleAlertFunc: func(ctx context.Context, schema types.AlertSchema, alertData any) ([]*alert.Alert, error) {
				gt.Value(t, schema).Equal("") // That's caused by calling AlertSNSHandler directly
				data, ok := alertData.(map[string]any)
				gt.True(t, ok)
				gt.Value(t, data["color"]).Equal("blue")
				return []*alert.Alert{}, nil
			},
		}
		mockUseCase := &useCaseInterface{
			AlertUsecases: alertMock,
		}

		// Create request with SNS message
		body, err := json.Marshal(snsMessage)
		gt.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/alert/sns/test", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		// Call handler
		server.AlertSNSHandler(mockUseCase)(rec, req)

		// Check response
		gt.Value(t, rec.Code).Equal(http.StatusOK)
		gt.Value(t, len(alertMock.HandleAlertCalls())).Equal(1)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockUseCase := &useCaseInterface{}

		req := httptest.NewRequest(http.MethodPost, "/alert/sns/test", bytes.NewReader([]byte("invalid json")))
		rec := httptest.NewRecorder()

		server.AlertSNSHandler(mockUseCase)(rec, req)

		gt.Value(t, rec.Code).Equal(http.StatusBadRequest)
	})

	t.Run("invalid alert data", func(t *testing.T) {
		alertMock := &mock.AlertUsecasesMock{
			HandleAlertFunc: func(ctx context.Context, schema types.AlertSchema, alertData any) ([]*alert.Alert, error) {
				return []*alert.Alert{}, nil
			},
		}

		invalidMessage := snsMessage
		invalidMessage.Message = "invalid json"

		body, err := json.Marshal(invalidMessage)
		gt.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/alert/sns/test", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		server.AlertSNSHandler(&useCaseInterface{
			AlertUsecases: alertMock,
		})(rec, req)

		gt.Value(t, rec.Code).Equal(http.StatusBadRequest)
		gt.Value(t, len(alertMock.HandleAlertCalls())).Equal(0)
	})
}

func TestAlertRawHandler_ContentType(t *testing.T) {
	validJSON := `{"test": "data"}`

	testCases := []struct {
		name        string
		contentType string
		expectCode  int
		expectCalls int
	}{
		{
			name:        "valid application/json",
			contentType: "application/json",
			expectCode:  http.StatusOK,
			expectCalls: 1,
		},
		{
			name:        "valid application/json with charset",
			contentType: "application/json; charset=utf-8",
			expectCode:  http.StatusOK,
			expectCalls: 1,
		},
		{
			name:        "valid application/json with multiple params",
			contentType: "application/json; charset=utf-8; boundary=something",
			expectCode:  http.StatusOK,
			expectCalls: 1,
		},
		{
			name:        "invalid text/plain",
			contentType: "text/plain",
			expectCode:  http.StatusBadRequest,
			expectCalls: 0,
		},
		{
			name:        "invalid application/xml",
			contentType: "application/xml",
			expectCode:  http.StatusBadRequest,
			expectCalls: 0,
		},
		{
			name:        "malformed content type",
			contentType: "application/json;;invalid",
			expectCode:  http.StatusBadRequest,
			expectCalls: 0,
		},
		{
			name:        "empty content type",
			contentType: "",
			expectCode:  http.StatusBadRequest,
			expectCalls: 0,
		},
		{
			name:        "invalid parameter name (parse error)",
			contentType: "text/plain; application/json",
			expectCode:  http.StatusBadRequest,
			expectCalls: 0,
		},
		{
			name:        "valid parameter format but wrong media type",
			contentType: "multipart/form-data; type=application/json",
			expectCode:  http.StatusBadRequest,
			expectCalls: 0,
		},
		{
			name:        "reversed parameter order (parse error)",
			contentType: "charset=utf-8; application/json",
			expectCode:  http.StatusBadRequest,
			expectCalls: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create fresh mock for each test to avoid interference
			testAlertMock := &mock.AlertUsecasesMock{
				HandleAlertFunc: func(ctx context.Context, schema types.AlertSchema, alertData any) ([]*alert.Alert, error) {
					return []*alert.Alert{}, nil
				},
			}
			testMockUseCase := &useCaseInterface{
				AlertUsecases: testAlertMock,
			}

			req := httptest.NewRequest(http.MethodPost, "/alert/raw/test-schema", bytes.NewReader([]byte(validJSON)))
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}
			rec := httptest.NewRecorder()

			// Call handler
			server.AlertRawHandler(testMockUseCase)(rec, req)

			// Check response code
			gt.Value(t, rec.Code).Equal(tc.expectCode)

			// Check if handler was called expected number of times
			gt.Value(t, len(testAlertMock.HandleAlertCalls())).Equal(tc.expectCalls)
		})
	}
}
