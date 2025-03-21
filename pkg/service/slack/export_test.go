package slack

import (
	"context"

	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/slack-go/slack"
)

// PostMessage posts a message to the channel and returns the thread. It's just for testing.
func (x *Service) PostMessage(ctx context.Context, message string) (*ThreadService, error) {
	ch, thread, err := x.client.PostMessageContext(ctx, x.channelID, slack.MsgOptionText(message, false))
	if err != nil {
		return nil, err
	}

	return x.NewThread(model.Thread{
		ChannelID: ch,
		ThreadID:  thread,
	}), nil
}

// BotUserID returns the bot user ID. It's just for testing.
func (x *Service) BotUserID() string {
	return x.botID
}
