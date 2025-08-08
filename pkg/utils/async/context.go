package async

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/async"
)

type ctxAsyncModeKey struct{}

// WithAsyncMode sets async mode configuration in context
func WithAsyncMode(ctx context.Context, cfg *async.Config) context.Context {
	return context.WithValue(ctx, ctxAsyncModeKey{}, cfg)
}

// GetAsyncMode gets async mode configuration from context
func GetAsyncMode(ctx context.Context) *async.Config {
	cfg, ok := ctx.Value(ctxAsyncModeKey{}).(*async.Config)
	if !ok {
		return nil
	}
	return cfg
}
