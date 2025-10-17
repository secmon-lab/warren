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

func TestWithNewStateMsg(t *testing.T) {
	var calledReply bool
	var callCount int
	var lastMsg string
	replyFunc := func(ctx context.Context, msg string) {
		calledReply = true
	}
	newStateFunc := func(ctx context.Context, msg string) func(ctx context.Context, msg string) {
		// NewTraceFunc should post the initial message immediately (like NewTraceMessage does)
		callCount++
		lastMsg = msg

		// Return a function for potential updates (though with new behavior, this won't be used)
		return func(ctx context.Context, updateMsg string) {
			callCount++
			lastMsg = updateMsg
		}
	}

	ctx := msg.With(context.Background(), replyFunc, newStateFunc)
	ctx = msg.NewTrace(ctx, "test new state")
	msg.Trace(ctx, "test state messsage")

	gt.False(t, calledReply)
	// Trace now always creates a new trace, so newStateFunc is called twice:
	// once for NewTrace("test new state") and once for Trace("test state messsage")
	gt.Equal(t, callCount, 2)
	gt.Equal(t, lastMsg, "test state messsage")
}

func TestNewStateMsg_Nil(t *testing.T) {
	ctx := context.Background()
	ctx = msg.NewTrace(ctx, "test state")

	// Should not panic
	msg.Trace(ctx, "test message")
}
