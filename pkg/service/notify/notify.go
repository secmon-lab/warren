package notify

import (
	"context"
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"

	slack_sdk "github.com/slack-go/slack"
)

// const maxContextTextLength = 3000
/*
func countTextLength(messages []string) int {
	length := 0
	for _, msg := range messages {
		length += len([]rune(msg))
	}
	return length
}
*/
type SlackThread struct {
	client slackClient
	thread slack.Thread
}

func NewSlackThread(client slackClient, thread slack.Thread) *SlackThread {
	return &SlackThread{
		client: client,
		thread: thread,
	}
}

type slackClient interface {
	UpdateMessageContext(ctx context.Context, channelID, messageID string, options ...slack_sdk.MsgOption) (string, string, string, error)
	PostMessageContext(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error)
}

func (x *SlackThread) Notify(ctx context.Context, msg string) {
	_, _, err := x.client.PostMessageContext(ctx,
		x.thread.ChannelID,
		slack_sdk.MsgOptionBlocks(
			slack_sdk.NewSectionBlock(
				slack_sdk.NewTextBlockObject(slack_sdk.MarkdownType, msg, false, false),
				nil,
				nil,
			),
		),
		slack_sdk.MsgOptionTS(x.thread.ThreadID),
	)

	if err != nil {
		errs.Handle(ctx, goerr.Wrap(err, "failed to post message to slack",
			goerr.V("channelID", x.thread.ChannelID),
			goerr.V("threadID", x.thread.ThreadID),
			goerr.V("message", msg),
		))
	}
}

func (x *SlackThread) NewMessageContext(ctx context.Context, base string) *MessageContext {
	blocks := newContextBlocks(base, []string{})
	_, msgID, err := x.client.PostMessageContext(ctx,
		x.thread.ChannelID,
		slack_sdk.MsgOptionBlocks(blocks...),
		slack_sdk.MsgOptionTS(x.thread.ThreadID),
	)

	if err != nil {
		errs.Handle(ctx, goerr.Wrap(err, "failed to post message to slack",
			goerr.V("channelID", x.thread.ChannelID),
			goerr.V("threadID", x.thread.ThreadID),
			goerr.V("blocks", blocks),
		))

		// Return dummy context
		return newDummyMessageContext()
	}

	return &MessageContext{
		client:  x.client,
		thread:  x.thread,
		msgID:   msgID,
		baseMsg: base,
	}
}

type MessageContext struct {
	client   slackClient
	msgID    string
	thread   slack.Thread
	baseMsg  string
	messages []string
	mutex    sync.Mutex
}

func newDummyMessageContext() *MessageContext {
	return &MessageContext{}
}

func (x *MessageContext) Append(ctx context.Context, msg string) {
	// If the message is not posted yet, do nothing
	if x.msgID == "" {
		return
	}

	x.mutex.Lock()
	defer x.mutex.Unlock()

	newMsg := append(x.messages, msg)

	blocks := newContextBlocks(x.baseMsg, newMsg)
	_, _, _, err := x.client.UpdateMessageContext(
		ctx,
		x.thread.ChannelID,
		x.msgID,
		slack_sdk.MsgOptionTS(x.thread.ThreadID),
		slack_sdk.MsgOptionBlocks(blocks...),
	)
	if err != nil {
		errs.Handle(ctx, goerr.Wrap(err, "failed to notify slack",
			goerr.V("channelID", x.thread.ChannelID),
			goerr.V("threadID", x.thread.ThreadID),
			goerr.V("blocks", blocks),
		))
	}

	x.messages = newMsg
}
