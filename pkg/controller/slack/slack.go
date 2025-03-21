package slack

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	slack_model "github.com/secmon-lab/warren/pkg/domain/model/slack"
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

	if IsSync(ctx) {
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

	slackThread := slack_model.Thread{
		ChannelID: event.Channel,
		ThreadID:  event.ThreadTimeStamp,
	}
	if slackThread.ThreadID == "" {
		slackThread.ThreadID = event.TimeStamp
	}

	mentions := slack_model.ParseMention(event.Text)
	user := slack_model.User{
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

	slackThread := slack_model.Thread{
		ChannelID: event.Channel,
		ThreadID:  event.ThreadTimeStamp,
	}
	user := slack_model.User{
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

	user := slack_model.User{
		ID:   interaction.User.ID,
		Name: interaction.User.Name,
	}
	th := slack_model.Thread{
		ChannelID: interaction.Channel.ID,
		ThreadID:  interaction.Message.ThreadTimestamp,
	}

	for _, action := range interaction.ActionCallback.BlockActions {
		return x.uc.HandleSlackInteractionBlockActions(ctx, user, th, slack_model.ActionID(action.ActionID), action.Value, interaction.TriggerID)
	}

	return nil
}

func (x *Controller) handleSlackInteractionViewSubmission(ctx context.Context, interaction slack.InteractionCallback) error {
	values := interaction.View.State.Values
	metadata := interaction.View.PrivateMetadata
	user := slack_model.User{
		ID:   interaction.User.ID,
		Name: interaction.User.Name,
	}

	sv := slack_model.BlockActionFromValue(values)

	switch slack_model.CallbackID(interaction.View.CallbackID) {
	case slack_model.CallbackSubmitResolveAlert:
		return x.uc.HandleSlackInteractionViewSubmissionResolveAlert(ctx, user, metadata, sv)
	case slack_model.CallbackSubmitResolveList:
		return x.uc.HandleSlackInteractionViewSubmissionResolveList(ctx, user, metadata, sv)
	case slack_model.CallbackSubmitIgnoreList:
		return x.uc.HandleSlackInteractionViewSubmissionIgnoreList(ctx, metadata, sv)
	}

	return nil
}
