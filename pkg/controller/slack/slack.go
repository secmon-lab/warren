package slack

import (
	"context"
	"runtime/debug"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	slack_model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func newBackgroundContext(ctx context.Context) context.Context {
	newCtx := context.Background()
	newCtx = logging.With(newCtx, logging.From(ctx))
	newCtx = msg.WithContext(newCtx)
	newCtx = lang.With(newCtx, lang.From(ctx))
	return newCtx
}

func dispatch(ctx context.Context, handler func(ctx context.Context) error) {
	newCtx := newBackgroundContext(ctx)

	if IsSync(ctx) {
		if err := handler(newCtx); err != nil {
			errs.Handle(newCtx, err)
		}
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				errs.Handle(newCtx, goerr.New("panic", goerr.V("recover", r), goerr.V("stack", string(stack))))
			}
		}()

		if err := handler(newCtx); err != nil {
			errs.Handle(newCtx, err)
		}
	}()
}

type Controller struct {
	event       interfaces.SlackEventUsecases
	interaction interfaces.SlackInteractionUsecases
}

func New(event interfaces.SlackEventUsecases, interaction interfaces.SlackInteractionUsecases) *Controller {
	return &Controller{
		event:       event,
		interaction: interaction,
	}
}

func (x *Controller) HandleSlackAppMention(ctx context.Context, apiEvent *slackevents.EventsAPIEvent, event *slackevents.AppMentionEvent) error {
	logger := logging.From(ctx).With("event_ts", event.EventTimeStamp)
	ctx = logging.With(ctx, logger)

	slackMsg := slack_model.NewMessage(ctx, apiEvent)
	if slackMsg == nil {
		return nil
	}

	dispatch(ctx, func(ctx context.Context) error {
		return x.event.HandleSlackAppMention(ctx, *slackMsg)
	})

	return nil
}

func (x *Controller) HandleSlackMessage(ctx context.Context, apiEvent *slackevents.EventsAPIEvent, event *slackevents.MessageEvent) error {
	logger := logging.From(ctx).With("event_ts", event.EventTimeStamp)
	ctx = logging.With(ctx, logger)

	logger.Debug("slack message event", "event", event)

	if event.ThreadTimeStamp == "" {
		return nil
	}

	slackMsg := slack_model.NewMessage(ctx, apiEvent)
	if slackMsg == nil {
		return nil
	}

	dispatch(ctx, func(ctx context.Context) error {
		return x.event.HandleSlackMessage(ctx, *slackMsg)
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
		return x.interaction.HandleSlackInteractionBlockActions(ctx, user, th, slack_model.ActionID(action.ActionID), action.Value, interaction.TriggerID)
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

	callbackID := slack_model.CallbackID(interaction.View.CallbackID)

	return x.interaction.HandleSlackInteractionViewSubmission(ctx, user, callbackID, metadata, sv)
}
