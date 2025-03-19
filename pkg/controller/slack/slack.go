package slack

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model"
	"github.com/secmon-lab/warren/pkg/utils/errs"
	"github.com/secmon-lab/warren/pkg/utils/lang"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/thread"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func dispatch(ctx context.Context, handler func(ctx context.Context) error) {
	newCtx := context.Background()

	newCtx = logging.With(newCtx, logging.From(ctx))
	newCtx = thread.WithReplyFunc(newCtx, thread.ReplyFuncFrom(ctx))
	newCtx = lang.With(newCtx, lang.From(ctx))

	if model.IsSync(ctx) {
		if err := handler(newCtx); err != nil {
			errs.Handle(newCtx, err)
		}
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errs.Handle(newCtx, goerr.New("panic", goerr.V("recover", r)))
			}
		}()

		if err := handler(newCtx); err != nil {
			errs.Handle(newCtx, err)
		}
	}()
}

type Controller struct {
	uc interfaces.UseCase
}

func New(uc interfaces.UseCase) *Controller {
	return &Controller{uc: uc}
}

func (x *Controller) HandleSlackAppMention(ctx context.Context, event *slackevents.AppMentionEvent) error {
	logger := logging.From(ctx).With("event_ts", event.EventTimeStamp)
	ctx = logging.With(ctx, logger)

	slackThread := model.SlackThread{
		ChannelID: event.Channel,
		ThreadID:  event.ThreadTimeStamp,
	}
	if slackThread.ThreadID == "" {
		slackThread.ThreadID = event.TimeStamp
	}

	mentions := parseMention(event.Text)
	user := model.SlackUser{
		ID:   event.User,
		Name: event.User,
	}

	logger.Info("slack app mention event", "mentions", mentions, "user", user)

	if len(mentions) == 0 {
		// nothing to do
		return nil
	}

	for _, mention := range mentions {
		dispatch(ctx, func(ctx context.Context) error {
			return x.uc.HandleSlackAppMention(ctx, user, mention, slackThread)
		})
	}

	return nil
}

func (x *Controller) HandleSlackMessage(ctx context.Context, event *slackevents.MessageEvent) error {
	logger := logging.From(ctx).With("event_ts", event.EventTimeStamp)
	ctx = logging.With(ctx, logger)

	logger.Debug("slack message event", "event", event)

	if event.ThreadTimeStamp == "" {
		return nil
	}

	slackThread := model.SlackThread{
		ChannelID: event.Channel,
		ThreadID:  event.ThreadTimeStamp,
	}
	user := model.SlackUser{
		ID:   event.User,
		Name: event.User,
	}

	dispatch(ctx, func(ctx context.Context) error {
		return x.uc.HandleSlackMessage(ctx, slackThread, event.Text, user, event.EventTimeStamp)
	})

	return nil
}

func (x *Controller) HandleSlackInteraction(ctx context.Context, interaction slack.InteractionCallback) error {
	logger := logging.From(ctx)
	logger.Info("slack interaction event", "event", interaction)

	dispatch(ctx, func(ctx context.Context) error {
		switch interaction.Type {
		case slack.InteractionTypeBlockActions:
			return x.handleSlackInteractionBlockActions(ctx, interaction)
		case slack.InteractionTypeViewSubmission:
			return x.handleSlackInteractionViewSubmission(ctx, interaction)
		}

		return nil
	})

	return nil
}

func (x *Controller) handleSlackInteractionBlockActions(ctx context.Context, interaction slack.InteractionCallback) error {

	user := model.SlackUser{
		ID:   interaction.User.ID,
		Name: interaction.User.Name,
	}
	th := model.SlackThread{
		ChannelID: interaction.Channel.ID,
		ThreadID:  interaction.Message.ThreadTimestamp,
	}

	for _, action := range interaction.ActionCallback.BlockActions {
		return x.uc.HandleSlackInteractionBlockActions(ctx, user, th, model.SlackActionID(action.ActionID), action.Value, interaction.TriggerID)
	}

	return nil
}

func (x *Controller) handleSlackInteractionViewSubmission(ctx context.Context, interaction slack.InteractionCallback) error {
	values := interaction.View.State.Values
	metadata := interaction.View.PrivateMetadata
	user := model.SlackUser{
		ID:   interaction.User.ID,
		Name: interaction.User.Name,
	}

	switch model.SlackCallbackID(interaction.View.CallbackID) {
	case model.SlackCallbackSubmitResolveAlert:
		return x.uc.HandleSlackInteractionViewSubmissionResolveAlert(ctx, user, metadata, values)
	case model.SlackCallbackSubmitResolveList:
		return x.uc.HandleSlackInteractionViewSubmissionResolveList(ctx, user, metadata, values)
	case model.SlackCallbackSubmitIgnoreList:
		return x.uc.HandleSlackInteractionViewSubmissionIgnoreList(ctx, metadata, values)
	}

	return nil
}
