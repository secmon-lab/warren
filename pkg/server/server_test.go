package server_test

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opac"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/mock"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/server"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

//go:embed testdata/pubsub.json
var pubsubJSON []byte

func TestValidateGoogleIDToken(t *testing.T) {
	vars := test.NewEnvVars(t, "TEST_GOOGLE_ID_TOKEN", "TEST_GOOGLE_ID_TOKEN_EMAIL")
	calledAuthQuery := false
	policyMock := &mock.PolicyClientMock{
		QueryFunc: func(contextMoqParam context.Context, s string, v1, v2 any, queryOptions ...opac.QueryOption) error {
			if s == "data.auth" {
				calledAuthQuery = true
				m1 := v1.(*model.AuthContext)
				gt.Equal(t, m1.Google["email"].(string), vars.Get("TEST_GOOGLE_ID_TOKEN_EMAIL"))
				gt.NoError(t, json.Unmarshal([]byte(`{"allow":true}`), &v2))
			} else {
				gt.NoError(t, json.Unmarshal([]byte(`{}`), &v2))
			}
			return nil
		},
	}
	ssnMock := func() interfaces.GenAIChatSession {
		ssn := &mock.GenAIChatSessionMock{}
		return ssn
	}
	uc := usecase.New(ssnMock, usecase.WithPolicyClient(policyMock))

	server := server.New(uc)

	t.Run("with valid token", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/alert/pubsub/test", bytes.NewReader(pubsubJSON))
		req.Header.Set("Authorization", "Bearer "+vars.Get("TEST_GOOGLE_ID_TOKEN"))
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		gt.Equal(t, http.StatusOK, w.Code)
		gt.A(t, policyMock.QueryCalls()).Length(2)
		gt.True(t, calledAuthQuery)
	})
}
