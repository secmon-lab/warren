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
	msg.Reply(ctx, "test message")

	gt.True(t, called)
	gt.Equal(t, "test message", gotMsg)
}

func TestWithNewStateMsg(t *testing.T) {
	var calledReply bool
	var calledState bool
	var gotMsgState string
	replyFunc := func(ctx context.Context, msg string) {
		calledReply = true
	}
	stateMsgFunc := func(ctx context.Context, msg string) {
		calledState = true
		gotMsgState = msg
	}
	newStateFunc := func(ctx context.Context, msg string) func(ctx context.Context, msg string) {
		return stateMsgFunc
	}

	ctx := msg.With(context.Background(), replyFunc, newStateFunc)
	ctx = msg.NewState(ctx, "test new state")
	msg.State(ctx, "test state messsage")

	gt.False(t, calledReply)
	gt.True(t, calledState)
	gt.Equal(t, "test state messsage", gotMsgState)
}

func TestNewStateMsg_Nil(t *testing.T) {
	ctx := context.Background()
	ctx = msg.NewState(ctx, "test state")

	// Should not panic
	msg.State(ctx, "test message")
}
