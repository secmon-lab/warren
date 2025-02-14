package authctx

import (
	"context"

	"github.com/secmon-lab/warren/pkg/model"
)

type ctxGoogleIDTokenClaimsKey struct{}

func WithGoogleIDTokenClaims(ctx context.Context, claims map[string]interface{}) context.Context {
	return context.WithValue(ctx, ctxGoogleIDTokenClaimsKey{}, claims)
}

func Build(ctx context.Context) *model.AuthContext {
	var authCtx model.AuthContext
	claims, ok := ctx.Value(ctxGoogleIDTokenClaimsKey{}).(map[string]interface{})
	if ok {
		authCtx.Google = claims
	}

	return &authCtx
}
