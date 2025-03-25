package http_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/harlog"
	"github.com/m-mizutani/opaq"
	server "github.com/secmon-lab/warren/pkg/controller/http"
	slack_ctrl "github.com/secmon-lab/warren/pkg/controller/slack"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	slack_model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"

	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

//go:embed testdata/pubsub.json
var pubsubJSON []byte

func TestValidateGoogleIDToken(t *testing.T) {
	vars := test.NewEnvVars(t, "TEST_GOOGLE_ID_TOKEN", "TEST_GOOGLE_ID_TOKEN_EMAIL")

	policyClient := &interfaces.PolicyClientMock{
		QueryFunc: func(ctx context.Context, s string, v1, v2 any, queryOptions ...opaq.QueryOption) error {
			if s == "data.alert.test" {
				return nil
			}

			m1, ok := v1.(auth.Context)
			gt.True(t, ok)
			gt.Equal(t, m1.Google["email"].(string), vars.Get("TEST_GOOGLE_ID_TOKEN_EMAIL"))
			gt.NoError(t, json.Unmarshal([]byte(`{"allow":true}`), &v2))
			return nil
		},
	}

	uc := usecase.New(usecase.WithPolicyClient(policyClient))

	server := server.New(uc)

	t.Run("with valid token", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/alert/pubsub/test", bytes.NewReader(pubsubJSON))
		req.Header.Set("Authorization", "Bearer "+vars.Get("TEST_GOOGLE_ID_TOKEN"))
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		gt.Equal(t, http.StatusOK, w.Code)
		gt.A(t, policyClient.QueryCalls()).Length(2).
			At(0, func(t testing.TB, v struct {
				ContextMoqParam context.Context
				S               string
				V1              any
				V2              any
				QueryOptions    []opaq.QueryOption
			}) {
				gt.Equal(t, v.S, "data.auth")
			}).
			At(1, func(t testing.TB, v struct {
				ContextMoqParam context.Context
				S               string
				V1              any
				V2              any
				QueryOptions    []opaq.QueryOption
			}) {
				gt.Equal(t, v.S, "data.alert.test")
			})
	})
}

//go:embed testdata/slack_interaction.json
var slackInteractionJSON []byte

func TestSlackInteractionHandler(t *testing.T) {
	signingSecret := "test_signing_secret"
	uc := &UseCaseMock{
		HandleSlackInteractionViewSubmissionResolveAlertFunc: func(ctx context.Context, user slack_model.User, metadata string, values slack_model.StateValue) error {
			return nil
		},
		HandleSlackInteractionViewSubmissionResolveListFunc: func(ctx context.Context, user slack_model.User, metadata string, values slack_model.StateValue) error {
			return nil
		},
		HandleSlackInteractionViewSubmissionIgnoreListFunc: func(ctx context.Context, metadata string, values slack_model.StateValue) error {
			return nil
		},
		HandleSlackInteractionBlockActionsFunc: func(ctx context.Context, user slack_model.User, slackThread slack_model.Thread, actionID slack_model.ActionID, value, triggerID string) error {
			return nil
		},
	}
	srv := server.New(uc, server.WithSlackVerifier(slack_model.NewPayloadVerifier(signingSecret)))

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

		req = req.WithContext(slack_ctrl.WithSync(t.Context()))
		srv.ServeHTTP(w, req)

		gt.Equal(t, http.StatusOK, w.Code)
	})
}

//go:embed testdata/slack_mention.json
var slackMentionJSON []byte

func TestSlackMentionHandler(t *testing.T) {
	signingSecret := "test_signing_secret"
	uc := &UseCaseMock{
		HandleSlackAppMentionFunc: func(ctx context.Context, user slack_model.User, mention slack_model.Mention, slackThread slack_model.Thread) error {
			gt.Equal(t, user.ID, "U8JLN34SV")
			gt.Equal(t, slackThread.ChannelID, "C07AR2FPG1F")
			gt.Equal(t, slackThread.ThreadID, "1741487414.163419")
			gt.Equal(t, mention.UserID, "U08A3TTRENS")
			gt.Equal(t, mention.Message, "kokoro")
			return nil
		},
	}
	srv := server.New(uc, server.WithSlackVerifier(slack_model.NewPayloadVerifier(signingSecret)))

	t.Run("with valid signature", func(t *testing.T) {
		ctx := slack_ctrl.WithSync(t.Context())
		ts := fmt.Sprint(time.Now().Unix())

		// Calculate signature
		signature := calculateSlackSignature(string(slackMentionJSON), ts, signingSecret)

		req := httptest.NewRequest("POST", "/slack/event", strings.NewReader(string(slackMentionJSON)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Slack-Signature", signature)
		req.Header.Set("X-Slack-Request-Timestamp", ts)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req.WithContext(ctx))

		gt.Equal(t, http.StatusOK, w.Code)

		gt.A(t, uc.HandleSlackAppMentionCalls()).Length(1)
	})

}

func calculateSlackSignature(payload string, ts string, signingSecret string) string {
	baseString := "v0:" + ts + ":" + payload
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(baseString))
	return "v0=" + hex.EncodeToString(mac.Sum(nil))
}

//go:embed testdata/sns.pem
var snsPem []byte

func TestAlertSNS(t *testing.T) {
	uc := &UseCaseMock{
		HandleAlertWithAuthFunc: func(ctx context.Context, schema types.AlertSchema, alertData any) ([]*alert.Alert, error) {
			return nil, nil
		},
	}
	srv := server.New(uc)

	ctx := server.WithHTTPClient(t.Context(), &mockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return &http.Response{
				Body: io.NopCloser(bytes.NewReader(snsPem)),
			}, nil
		},
	})

	t.Run("with valid SNS message", func(t *testing.T) {
		logs, err := harlog.ParseHARData(snsHar)
		gt.NoError(t, err)
		gt.A(t, logs).Length(1)

		log := logs[0]
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, log.Request.WithContext(ctx))

		gt.Equal(t, http.StatusOK, w.Code)
		gt.A(t, uc.HandleAlertWithAuthCalls()).Length(1)
	})
}
