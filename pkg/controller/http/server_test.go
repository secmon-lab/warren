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

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/harlog"
	"github.com/m-mizutani/opaq"
	server "github.com/secmon-lab/warren/pkg/controller/http"
	slack_ctrl "github.com/secmon-lab/warren/pkg/controller/slack"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	slack_model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"

	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

//go:embed testdata/pubsub.json
var pubsubJSON []byte

//go:embed testdata/slack_interaction.json
var slackInteractionJSON []byte

//go:embed testdata/slack_mention.json
var slackMentionJSON []byte

//go:embed testdata/sns.pem
var snsPem []byte

func TestValidateGoogleIDToken(t *testing.T) {
	vars := test.NewEnvVars(t, "TEST_GOOGLE_ID_TOKEN", "TEST_GOOGLE_ID_TOKEN_EMAIL")

	// Check if the token is expired
	tokenString := vars.Get("TEST_GOOGLE_ID_TOKEN")

	// Parse JWT without verification to check expiration
	token, err := jwt.ParseInsecure([]byte(tokenString))
	if err != nil {
		t.Skipf("Failed to parse JWT token: %v", err)
	}

	// Check expiration
	if token.Expiration().Before(time.Now()) {
		t.Skipf("Google ID token has expired at %v, skipping test", token.Expiration())
	}

	policyClient := &mock.PolicyClientMock{
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

	server := server.New(uc, server.WithPolicy(policyClient))

	t.Run("with valid token", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/hooks/alert/pubsub/test", bytes.NewReader(pubsubJSON))
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

func TestSlackInteractionHandler(t *testing.T) {
	signingSecret := "test_signing_secret"
	uc := &useCaseInterface{
		SlackInteractionUsecases: &mock.SlackInteractionUsecasesMock{
			HandleSlackInteractionViewSubmissionFunc: func(ctx context.Context, user slack_model.User, callbackID slack_model.CallbackID, metadata string, values slack_model.StateValue) error {
				return nil
			},
			HandleSlackInteractionBlockActionsFunc: func(ctx context.Context, user slack_model.User, slackThread slack_model.Thread, actionID slack_model.ActionID, value, triggerID string) error {
				return nil
			},
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

		req := httptest.NewRequest("POST", "/hooks/slack/interaction", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Slack-Signature", signature)
		req.Header.Set("X-Slack-Request-Timestamp", ts)
		w := httptest.NewRecorder()

		req = req.WithContext(slack_ctrl.WithSync(t.Context()))
		srv.ServeHTTP(w, req)

		gt.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSlackMentionHandler(t *testing.T) {
	signingSecret := "test_signing_secret"
	slackEventMock := &mock.SlackEventUsecasesMock{
		HandleSlackAppMentionFunc: func(ctx context.Context, slackMsg slack_model.Message) error {
			gt.Equal(t, slackMsg.User().ID, "U8JLN34SV")
			gt.Equal(t, slackMsg.ChannelID(), "C07AR2FPG1F")
			gt.Equal(t, slackMsg.ThreadID(), "1741487414.163419")
			gt.Equal(t, slackMsg.Mention()[0].UserID, "U08A3TTRENS")
			gt.Equal(t, slackMsg.Mention()[0].Message, "kokoro")
			return nil
		},
	}

	// Create a real usecase with minimal dependencies for testing
	uc := usecase.New(
		usecase.WithRepository(repository.NewMemory()),
	)

	// Override the SlackEventUsecases to use our mock
	ucInterface := &useCaseInterface{
		AlertUsecases:            uc,
		SlackEventUsecases:       slackEventMock,
		SlackInteractionUsecases: uc,
		ApiUsecases:              uc,
	}

	srv := server.New(ucInterface, server.WithSlackVerifier(slack_model.NewPayloadVerifier(signingSecret)))

	t.Run("with valid signature", func(t *testing.T) {
		ctx := slack_ctrl.WithSync(t.Context())
		ts := fmt.Sprint(time.Now().Unix())

		// Calculate signature
		signature := calculateSlackSignature(string(slackMentionJSON), ts, signingSecret)

		req := httptest.NewRequest("POST", "/hooks/slack/event", strings.NewReader(string(slackMentionJSON)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Slack-Signature", signature)
		req.Header.Set("X-Slack-Request-Timestamp", ts)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req.WithContext(ctx))

		gt.Equal(t, http.StatusOK, w.Code)

		gt.A(t, slackEventMock.HandleSlackAppMentionCalls()).Length(1)
	})

}

func calculateSlackSignature(payload string, ts string, signingSecret string) string {
	baseString := "v0:" + ts + ":" + payload
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(baseString))
	return "v0=" + hex.EncodeToString(mac.Sum(nil))
}

func TestAlertSNS(t *testing.T) {
	alertUsecasesMock := &mock.AlertUsecasesMock{
		HandleAlertFunc: func(ctx context.Context, schema types.AlertSchema, alertData any) ([]*alert.Alert, error) {
			return nil, nil
		},
	}

	// Create a real usecase with minimal dependencies for testing
	uc := usecase.New(
		usecase.WithRepository(repository.NewMemory()),
	)

	// Override the AlertUsecases to use our mock
	ucInterface := &useCaseInterface{
		AlertUsecases:            alertUsecasesMock,
		SlackEventUsecases:       uc,
		SlackInteractionUsecases: uc,
		ApiUsecases:              uc,
	}

	srv := server.New(ucInterface)

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
		// Update the URL to use the new path with /hooks prefix
		log.Request.URL.Path = "/hooks/alert/sns/test"
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, log.Request.WithContext(ctx))

		gt.Value(t, w.Code).Equal(http.StatusOK)
		gt.A(t, alertUsecasesMock.HandleAlertCalls()).Length(1)
	})
}

func TestGraphQLHandler(t *testing.T) {
	repo := repository.NewMemory()
	server := server.New(nil, server.WithGraphQLRepo(repo))

	// Add test data
	ticketID := types.TicketID("test-ticket-1")
	alertID := types.AlertID("test-alert-1")
	gt.NoError(t, repo.PutTicket(context.Background(), ticket.Ticket{
		ID:     ticketID,
		Status: types.TicketStatusOpen,
		Metadata: ticket.Metadata{
			Title: "Test Ticket",
		},
	}))
	gt.NoError(t, repo.PutAlert(context.Background(), alert.Alert{
		ID: alertID,
		Metadata: alert.Metadata{
			Title: "Test Alert",
		},
	}))

	t.Run("query ticket", func(t *testing.T) {
		body := map[string]interface{}{
			"query":     "query { ticket(id: \"test-ticket-1\") { id status createdAt } }",
			"variables": nil,
		}
		b, err := json.Marshal(body)
		gt.NoError(t, err)

		req := httptest.NewRequest("POST", "/graphql", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		gt.Value(t, w.Code).Equal(http.StatusOK)

		var response map[string]interface{}
		err = json.NewDecoder(w.Body).Decode(&response)
		gt.NoError(t, err)

		if errs, ok := response["errors"]; ok {
			t.Logf("GraphQL errors: %v", errs)
		}

		data := response["data"].(map[string]interface{})
		ticket := data["ticket"].(map[string]interface{})
		gt.Value(t, ticket["id"]).Equal("test-ticket-1")
		gt.Value(t, ticket["status"]).Equal("open")
	})

	t.Run("query alert", func(t *testing.T) {
		body := map[string]interface{}{
			"query":     "query { alert(id: \"test-alert-1\") { id title } }",
			"variables": nil,
		}
		b, err := json.Marshal(body)
		gt.NoError(t, err)

		req := httptest.NewRequest("POST", "/graphql", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		gt.Value(t, w.Code).Equal(http.StatusOK)

		var response map[string]interface{}
		err = json.NewDecoder(w.Body).Decode(&response)
		gt.NoError(t, err)

		if errs, ok := response["errors"]; ok {
			t.Logf("GraphQL errors: %v", errs)
		}

		data := response["data"].(map[string]interface{})
		alert := data["alert"].(map[string]interface{})
		gt.Value(t, alert["id"]).Equal("test-alert-1")
		gt.Value(t, alert["title"]).Equal("Test Alert")
	})
}

func TestServerWithNoAuthorization(t *testing.T) {
	t.Run("WithNoAuthorization option sets server field", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.New(usecase.WithRepository(repo))

		// Create a policy that would normally deny the request
		denyingPolicy := &mock.PolicyClientMock{
			QueryFunc: func(ctx context.Context, path string, input any, output any, opts ...opaq.QueryOption) error {
				// Always deny
				if result, ok := output.(*struct {
					Allow bool `json:"allow"`
				}); ok {
					result.Allow = false
				}
				return nil
			},
		}

		// Create server with both policy and no-authorization
		srvWithPolicy := server.New(uc,
			server.WithPolicy(denyingPolicy),
			server.WithNoAuthorization(true),
		)

		// Test that request succeeds despite denying policy
		testData := `{"test": "data"}`
		req := httptest.NewRequest("POST", "/hooks/alert/raw/test", strings.NewReader(testData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srvWithPolicy.ServeHTTP(w, req)

		// Should succeed because authorization is bypassed
		// The endpoint will return 200 or another success code when processing the alert
		gt.Value(t, w.Code).Equal(http.StatusOK)
	})

	t.Run("Without NoAuthorization option authorization is enforced", func(t *testing.T) {
		repo := repository.NewMemory()
		uc := usecase.New(usecase.WithRepository(repo))

		// Create a policy that denies the request
		denyingPolicy := &mock.PolicyClientMock{
			QueryFunc: func(ctx context.Context, path string, input any, output any, opts ...opaq.QueryOption) error {
				// Always deny
				if result, ok := output.(*struct {
					Allow bool `json:"allow"`
				}); ok {
					result.Allow = false
				}
				return nil
			},
		}

		// Create server WITHOUT no-authorization
		srv := server.New(uc, server.WithPolicy(denyingPolicy))

		// Test that request is denied
		testData := `{"test": "data"}`
		req := httptest.NewRequest("POST", "/hooks/alert/raw/test", strings.NewReader(testData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.ServeHTTP(w, req)

		// Should be forbidden because authorization is enforced
		gt.Value(t, w.Code).Equal(http.StatusForbidden)
	})

	t.Run("E2E flow with no-authorization flag", func(t *testing.T) {
		// Setup
		repo := repository.NewMemory()
		uc := usecase.New(usecase.WithRepository(repo))

		// Create a policy that would normally deny all requests
		denyAllPolicy := &mock.PolicyClientMock{
			QueryFunc: func(ctx context.Context, path string, input any, output any, opts ...opaq.QueryOption) error {
				if result, ok := output.(*struct {
					Allow bool `json:"allow"`
				}); ok {
					result.Allow = false
				}
				return nil
			},
		}

		// Create server with no-authorization flag
		srv := server.New(uc,
			server.WithPolicy(denyAllPolicy),
			server.WithNoAuthorization(true), // This should bypass the policy
		)

		// Test various endpoints
		testCases := []struct {
			name     string
			method   string
			path     string
			body     string
			wantCode int
		}{
			{
				name:     "alert endpoint accessible",
				method:   "POST",
				path:     "/hooks/alert/raw/test",
				body:     `{"test": "data"}`,
				wantCode: http.StatusOK,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var body io.Reader
				if tc.body != "" {
					body = strings.NewReader(tc.body)
				}
				req := httptest.NewRequest(tc.method, tc.path, body)
				if tc.body != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				// Add minimal auth context
				ctx := auth.WithHTTPRequest(req.Context(), &auth.HTTPRequest{
					Method: tc.method,
					Path:   tc.path,
					Body:   tc.body,
					Header: req.Header,
				})
				req = req.WithContext(ctx)

				w := httptest.NewRecorder()
				srv.ServeHTTP(w, req)

				gt.Value(t, w.Code).Equal(tc.wantCode)
			})
		}
	})

	t.Run("E2E flow with authorization enforced", func(t *testing.T) {
		// Setup
		repo := repository.NewMemory()
		uc := usecase.New(usecase.WithRepository(repo))

		// Create a policy that denies all requests
		denyAllPolicy := &mock.PolicyClientMock{
			QueryFunc: func(ctx context.Context, path string, input any, output any, opts ...opaq.QueryOption) error {
				if result, ok := output.(*struct {
					Allow bool `json:"allow"`
				}); ok {
					result.Allow = false
				}
				return nil
			},
		}

		// Create server WITHOUT no-authorization flag
		srv := server.New(uc, server.WithPolicy(denyAllPolicy))

		// Test that requests are denied
		testData := `{"test": "data"}`
		req := httptest.NewRequest("POST", "/hooks/alert/raw/test", strings.NewReader(testData))
		req.Header.Set("Content-Type", "application/json")
		ctx := auth.WithHTTPRequest(req.Context(), &auth.HTTPRequest{
			Method: "POST",
			Path:   "/hooks/alert/raw/test",
			Body:   testData,
			Header: req.Header,
		})
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		// Should be forbidden
		gt.Value(t, w.Code).Equal(http.StatusForbidden)
	})
}
