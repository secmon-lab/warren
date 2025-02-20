package server_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/mock"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/server"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/service/policy"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/slack-go/slack"
)

//go:embed testdata/pubsub.json
var pubsubJSON []byte

func TestValidateGoogleIDToken(t *testing.T) {
	vars := test.NewEnvVars(t, "TEST_GOOGLE_ID_TOKEN", "TEST_GOOGLE_ID_TOKEN_EMAIL")
	calledAuthQuery := false
	policyMock := &mock.PolicyClientMock{
		QueryFunc: func(contextMoqParam context.Context, s string, v1, v2 any, queryOptions ...opaq.QueryOption) error {
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
		SourcesFunc: func() map[string]string {
			return map[string]string{
				"auth": "package auth",
			}
		},
	}
	ssnMock := func() interfaces.GenAIChatSession {
		ssn := &mock.GenAIChatSessionMock{}
		return ssn
	}
	policyService := policy.New(repository.NewMemory(), policyMock, &model.TestDataSet{}, policy.WithFactory(func(data policy.PolicyData) (interfaces.PolicyClient, error) {
		return policyMock, nil
	}))
	uc := usecase.New(ssnMock, usecase.WithPolicyService(policyService))

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

//go:embed testdata/slack_interaction.json
var slackInteractionJSON []byte

func TestSlackInteractionHandler(t *testing.T) {
	signingSecret := "test_signing_secret"
	uc := &mock.UseCaseMock{
		HandleSlackInteractionFunc: func(ctx context.Context, interaction slack.InteractionCallback) error {
			return nil
		},
	}
	server := server.New(uc, server.WithSlackVerifier(service.NewSlackPayloadVerifier(signingSecret)))

	t.Run("with valid signature", func(t *testing.T) {
		ts := fmt.Sprint(time.Now().Unix())
		payload := string(slackInteractionJSON)

		// Convert payload to form value format
		form := url.Values{}
		form.Add("payload", payload)
		body := form.Encode()

		// Calculate signature
		signature := calculateSlackSignature(body, ts, signingSecret)

		req := httptest.NewRequest("POST", "/slack/interaction", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Slack-Signature", signature)
		req.Header.Set("X-Slack-Request-Timestamp", ts)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		gt.Equal(t, http.StatusOK, w.Code)
	})
}

func calculateSlackSignature(payload string, ts string, signingSecret string) string {
	baseString := "v0:" + ts + ":" + payload
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(baseString))
	return "v0=" + hex.EncodeToString(mac.Sum(nil))
}
