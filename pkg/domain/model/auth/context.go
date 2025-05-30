package auth

import (
	"context"
	"os"
	"strings"

	"github.com/secmon-lab/warren/pkg/domain/model/message"
)

type contextKey string

const (
	googleIDTokenClaimsKey contextKey = "google_id_token_claims"
	snsMessageKey          contextKey = "sns_message"
	httpRequestKey         contextKey = "http_request"
)

func WithGoogleIDTokenClaims(ctx context.Context, claims map[string]interface{}) context.Context {
	return context.WithValue(ctx, googleIDTokenClaimsKey, claims)
}

func WithSNSMessage(ctx context.Context, msg *message.SNS) context.Context {
	return context.WithValue(ctx, snsMessageKey, msg)
}

func WithHTTPRequest(ctx context.Context, req *HTTPRequest) context.Context {
	return context.WithValue(ctx, httpRequestKey, req)
}

func BuildContext(ctx context.Context) Context {
	var authCtx Context
	claims, ok := ctx.Value(googleIDTokenClaimsKey).(map[string]interface{})
	if ok {
		authCtx.Google = claims
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
