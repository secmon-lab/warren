package msg_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

func TestWithReply(t *testing.T) {
	var called bool
	var gotMsg string
	replyFunc := func(ctx context.Context, msg string) {
		called = true
		gotMsg = msg
	}

	ctx := msg.With(context.Background(), replyFunc, nil)
	msg.Notify(ctx, "test message")

	gt.True(t, called)
	gt.Equal(t, "test message", gotMsg)
}

func TestWithTraceFunc(t *testing.T) {
	var calledReply bool
	var callCount int
	var lastMsg string
	replyFunc := func(ctx context.Context, msg string) {
		calledReply = true
	}
	traceFunc := func(ctx context.Context, msg string) func(ctx context.Context, msg string) {
		// TraceFunc should post the trace message
		callCount++
		lastMsg = msg
		// Return a no-op function (not used in current implementation)
		return func(ctx context.Context, updateMsg string) {}
	}

	ctx := msg.With(context.Background(), replyFunc, traceFunc)
	msg.Trace(ctx, "test trace message 1")
	msg.Trace(ctx, "test trace message 2")

	gt.False(t, calledReply)
	// Trace is called twice
	gt.Equal(t, callCount, 2)
	gt.Equal(t, lastMsg, "test trace message 2")
}

func TestTrace_Nil(t *testing.T) {
	ctx := context.Background()

	// Should not panic when TraceFunc is not set
	msg.Trace(ctx, "test message")
}
