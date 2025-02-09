package usecase

import (
	"context"
	"net/http"

	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func (uc *UseCases) VerifySlackRequest(ctx context.Context, header http.Header, body []byte) error {
	return uc.slackService.VerifyRequest(header, body)
}

func (uc *UseCases) HandleSlackAppMention(ctx context.Context, event *slackevents.AppMentionEvent) error {
	logger := logging.From(ctx)
	logger.Info("slack app mention event", "event", event)
	return nil
}

func (uc *UseCases) HandleSlackMessage(ctx context.Context, event *slackevents.MessageEvent) error {
	logger := logging.From(ctx)
	logger.Info("slack message event", "event", event)
	return nil
}

func (uc *UseCases) HandleSlackInteraction(ctx context.Context, interaction slack.InteractionCallback) error {
	return nil
}
