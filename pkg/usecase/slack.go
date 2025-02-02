package usecase

import (
	"context"
	"net/http"

	"github.com/slack-go/slack"
)

func (uc *UseCases) VerifySlackRequest(ctx context.Context, header http.Header, body []byte) error {
	return uc.slackService.VerifyRequest(header, body)
}

func (uc *UseCases) HandleSlackEvent(ctx context.Context, event slack.Event) error {
	return nil
}

func (uc *UseCases) HandleSlackInteraction(ctx context.Context, interaction slack.InteractionCallback) error {
	return nil
}
