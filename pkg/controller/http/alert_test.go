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
	"github.com/secmon-lab/warren/pkg/mock"
	"github.com/secmon-lab/warren/pkg/model"
)

//go:embed testdata/sns.har
var snsHar []byte

func TestAlertSNSHandler(t *testing.T) {
	logs, err := harlog.ParseHARData(snsHar)
	gt.NoError(t, err)
	gt.A(t, logs).Length(1)

	log := logs[0]
	var snsMessage model.SNSMessage
	bodyData, err := io.ReadAll(log.Request.Body)
	gt.NoError(t, err)
	err = json.Unmarshal(bodyData, &snsMessage)
	gt.NoError(t, err)

	t.Run("successful alert handling", func(t *testing.T) {
		mockUseCase := &mock.UseCaseMock{
			HandleAlertWithAuthFunc: func(ctx context.Context, schema string, alertData any) ([]*model.Alert, error) {
				gt.Value(t, schema).Equal("") // That's caused by calling AlertSNSHandler directly
				data, ok := alertData.(map[string]any)
				gt.True(t, ok)
				gt.Value(t, data["color"]).Equal("blue")
				return []*model.Alert{}, nil
			},
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
		gt.Value(t, len(mockUseCase.HandleAlertWithAuthCalls())).Equal(1)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockUseCase := &mock.UseCaseMock{}

		req := httptest.NewRequest(http.MethodPost, "/alert/sns/test", bytes.NewReader([]byte("invalid json")))
		rec := httptest.NewRecorder()

		server.AlertSNSHandler(mockUseCase)(rec, req)

		gt.Value(t, rec.Code).Equal(http.StatusBadRequest)
		gt.Value(t, len(mockUseCase.HandleAlertWithAuthCalls())).Equal(0)
	})

	t.Run("invalid alert data", func(t *testing.T) {
		mockUseCase := &mock.UseCaseMock{}

		invalidMessage := snsMessage
		invalidMessage.Message = "invalid json"

		body, err := json.Marshal(invalidMessage)
		gt.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/alert/sns/test", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		server.AlertSNSHandler(mockUseCase)(rec, req)

		gt.Value(t, rec.Code).Equal(http.StatusBadRequest)
		gt.Value(t, len(mockUseCase.HandleAlertWithAuthCalls())).Equal(0)
	})
}
