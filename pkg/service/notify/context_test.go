package notify_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/service/notify"
)

func TestContext(t *testing.T) {
	ctx := context.Background()
	called := false
	send := func(ctx context.Context, msg string) {
		called = true
		gt.Value(t, msg).Equal("test")
	}
	ctx = notify.With(ctx, send)
	notify.Send(ctx, "test")
	gt.True(t, called)
}

func TestContext_NoSend(t *testing.T) {
	ctx := context.Background()
	notify.Send(ctx, "test")
	// no panic
}
