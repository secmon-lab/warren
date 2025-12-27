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

	ctx := msg.With(context.Background(), replyFunc, nil, nil)
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
	traceFunc := func(ctx context.Context, msg string) {
		// TraceFunc should post the trace message
		callCount++
		lastMsg = msg
	}

	ctx := msg.With(context.Background(), replyFunc, traceFunc, nil)
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

func TestWithWarnFunc(t *testing.T) {
	var calledWarn bool
	var warnMsg string
	warnFunc := func(ctx context.Context, msg string) {
		calledWarn = true
		warnMsg = msg
	}

	ctx := msg.With(context.Background(), nil, nil, warnFunc)
	msg.Warn(ctx, "test warning message")

	gt.True(t, calledWarn)
	gt.Equal(t, "test warning message", warnMsg)
}

func TestWarn_Nil(t *testing.T) {
	ctx := context.Background()

	// Should not panic when WarnFunc is not set
	msg.Warn(ctx, "test message")
}

func TestWarn_Format(t *testing.T) {
	var warnMsg string
	warnFunc := func(ctx context.Context, msg string) {
		warnMsg = msg
	}

	ctx := msg.With(context.Background(), nil, nil, warnFunc)
	msg.Warn(ctx, "warning: %s, code: %d", "test", 123)

	gt.Equal(t, "warning: test, code: 123", warnMsg)
}
