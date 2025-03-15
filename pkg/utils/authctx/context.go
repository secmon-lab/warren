package authctx

import (
	"context"

	"github.com/secmon-lab/warren/pkg/model"
)

type ctxGoogleIDTokenClaimsKey struct{}

type contextKey string

const (
	googleIDTokenClaimsKey contextKey = "google_id_token_claims"
	snsMessageKey          contextKey = "sns_message"
)

func WithGoogleIDTokenClaims(ctx context.Context, claims map[string]interface{}) context.Context {
	return context.WithValue(ctx, googleIDTokenClaimsKey, claims)
}

func WithSNSMessage(ctx context.Context, msg *model.SNSMessage) context.Context {
	return context.WithValue(ctx, snsMessageKey, msg)
}

func Build(ctx context.Context) *model.AuthContext {
	var authCtx model.AuthContext
	claims, ok := ctx.Value(ctxGoogleIDTokenClaimsKey{}).(map[string]interface{})
	if ok {
		authCtx.Google = claims
	}

	msg, ok := ctx.Value(snsMessageKey).(*model.SNSMessage)
	if ok {
		authCtx.SNS = msg
	}

	return &authCtx
}
