package notify

import (
	"context"
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"

	slack_sdk "github.com/slack-go/slack"
)

type SlackThread struct {
	client   slackClient
	msgID    string
	thread   slack.Thread
	messages []string
	mutex    sync.Mutex
}

const maxContextTextLength = 3000

func countTextLength(messages []string) int {
	length := 0
	for _, msg := range messages {
		length += len([]rune(msg))
	}
	return length
}

type slackClient interface {
	UpdateMessageContext(ctx context.Context, channelID, messageID string, options ...slack_sdk.MsgOption) (string, string, string, error)
	PostMessageContext(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error)
}

func NewSlackThread(client slackClient, thread slack.Thread) *SlackThread {
	return &SlackThread{
		client: client,
		thread: thread,
	}
}

func (x *SlackThread) Notify(ctx context.Context, msg string) {
	x.mutex.Lock()
	defer x.mutex.Unlock()

	var err error
	newMsg := append(x.messages, msg)
	if x.msgID != "" && countTextLength(newMsg) < maxContextTextLength {
		_, _, _, err = x.client.UpdateMessageContext(
			ctx,
			x.thread.ChannelID,
			x.msgID,
			slack_sdk.MsgOptionTS(x.thread.ThreadID),
			slack_sdk.MsgOptionBlocks(newContextBlock(newMsg)),
		)
	} else {
		newMsg = []string{msg}
		_, x.msgID, err = x.client.PostMessageContext(
			ctx,
			x.thread.ChannelID,
			slack_sdk.MsgOptionTS(x.thread.ThreadID),
			slack_sdk.MsgOptionBlocks(newContextBlock(newMsg)),
		)
	}
	x.messages = newMsg

	if err != nil {
		errs.Handle(ctx, goerr.Wrap(err, "failed to notify slack",
			goerr.V("channelID", x.thread.ChannelID),
			goerr.V("threadID", x.thread.ThreadID),
			goerr.V("message", msg),
		))
	}
}
