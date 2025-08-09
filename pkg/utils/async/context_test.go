package async_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	asyncModel "github.com/secmon-lab/warren/pkg/domain/model/async"
	"github.com/secmon-lab/warren/pkg/utils/async"
)

func TestAsyncModeContext(t *testing.T) {
	t.Run("stores and retrieves async mode config", func(t *testing.T) {
		ctx := context.Background()
		cfg := &asyncModel.Config{
			Raw:    true,
			PubSub: false,
			SNS:    true,
		}

		ctx = async.WithAsyncMode(ctx, cfg)
		retrieved := async.GetAsyncMode(ctx)

		gt.NotNil(t, retrieved)
		gt.Equal(t, retrieved.Raw, true)
		gt.Equal(t, retrieved.PubSub, false)
		gt.Equal(t, retrieved.SNS, true)
	})

	t.Run("returns nil when no config in context", func(t *testing.T) {
		ctx := context.Background()
		retrieved := async.GetAsyncMode(ctx)
		gt.Nil(t, retrieved)
	})

	t.Run("handles nil config", func(t *testing.T) {
		ctx := context.Background()
		ctx = async.WithAsyncMode(ctx, nil)
		retrieved := async.GetAsyncMode(ctx)
		gt.Nil(t, retrieved)
	})
}
