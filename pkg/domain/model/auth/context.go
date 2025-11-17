package auth

import (
	"context"
	"os"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/message"
	"github.com/secmon-lab/warren/pkg/utils/authctx"
)

type contextKey string

const (
	googleIDTokenClaimsKey contextKey = "google_id_token_claims"
	googleIAPJWTClaimsKey  contextKey = "google_iap_jwt_claims"
	snsMessageKey          contextKey = "sns_message"
	httpRequestKey         contextKey = "http_request"
)

func WithGoogleIDTokenClaims(ctx context.Context, claims map[string]interface{}) context.Context {
	return context.WithValue(ctx, googleIDTokenClaimsKey, claims)
}

func WithGoogleIAPJWTClaims(ctx context.Context, claims map[string]interface{}) context.Context {
	return context.WithValue(ctx, googleIAPJWTClaimsKey, claims)
}

func WithSNSMessage(ctx context.Context, msg *message.SNS) context.Context {
	return context.WithValue(ctx, snsMessageKey, msg)
}

func WithHTTPRequest(ctx context.Context, req *HTTPRequest) context.Context {
	return context.WithValue(ctx, httpRequestKey, req)
}

// GetGoogleIDTokenClaims retrieves Google ID token claims from context
func GetGoogleIDTokenClaims(ctx context.Context) (map[string]interface{}, error) {
	claims, ok := ctx.Value(googleIDTokenClaimsKey).(map[string]interface{})
	if !ok {
		return nil, goerr.New("Google ID token claims not found in context")
	}
	return claims, nil
}

// GetGoogleIAPJWTClaims retrieves Google IAP JWT claims from context
func GetGoogleIAPJWTClaims(ctx context.Context) (map[string]interface{}, error) {
	claims, ok := ctx.Value(googleIAPJWTClaimsKey).(map[string]interface{})
	if !ok {
		return nil, goerr.New("Google IAP JWT claims not found in context")
	}
	return claims, nil
}

func BuildContext(ctx context.Context) Context {
	var authCtx Context
	claims, ok := ctx.Value(googleIDTokenClaimsKey).(map[string]interface{})
	if ok {
		authCtx.Google = claims
	}

	iapClaims, ok := ctx.Value(googleIAPJWTClaimsKey).(map[string]interface{})
	if ok {
		authCtx.IAP = iapClaims
	}

	msg, ok := ctx.Value(snsMessageKey).(*message.SNS)
	if ok {
		authCtx.SNS = msg
	}

	req, ok := ctx.Value(httpRequestKey).(*HTTPRequest)
	if ok {
		authCtx.Req = req
	}

	envVars := os.Environ()
	authCtx.Env = make(map[string]string)
	for _, v := range envVars {
		parts := strings.Split(v, "=")
		if len(parts) == 2 {
			authCtx.Env[parts[0]] = parts[1]
		}
	}

	return authCtx
}

// BuildAgentContext creates AgentContext from Go context and message
func BuildAgentContext(ctx context.Context, message string) AgentContext {
	agentCtx := AgentContext{
		Message: message,
	}

	// Get Slack user from authctx
	subjects := authctx.GetSubjects(ctx)
	for _, subject := range subjects {
		if subject.Type == authctx.SubjectTypeSlack {
			agentCtx.Auth = &AgentAuthInfo{
				Slack: &SlackAuthInfo{
					ID: subject.UserID,
				},
			}
			break
		}
	}

	// Get environment variables (same pattern as BuildContext for HTTP)
	envVars := os.Environ()
	agentCtx.Env = make(map[string]string)
	for _, v := range envVars {
		parts := strings.Split(v, "=")
		if len(parts) == 2 {
			agentCtx.Env[parts[0]] = parts[1]
		}
	}

	return agentCtx
}
