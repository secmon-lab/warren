package slack

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	slack_model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/utils/async"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/user"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func dispatch(ctx context.Context, handler func(ctx context.Context) error) {
	if IsSync(ctx) {
		if err := handler(ctx); err != nil {
			errs.Handle(ctx, err)
		}
		return
	}

	// Use common async dispatch
	async.Dispatch(ctx, handler)
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

	// Set user context from Slack event
	ctx = user.WithUserID(ctx, event.User)

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

	// Set user context from Slack event
	ctx = user.WithUserID(ctx, event.User)

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

	// Set user context from Slack interaction
	ctx = user.WithUserID(ctx, interaction.User.ID)

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

	// Handle modal block actions differently (they don't have channel/thread context)
	if interaction.View.ID != "" {
		for _, action := range interaction.ActionCallback.BlockActions {
			if action.ActionID == "salvage_refresh_button" {
				return x.handleSalvageRefresh(ctx, interaction, user)
			}
		}
		return nil
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

func (x *Controller) handleSalvageRefresh(ctx context.Context, interaction slack.InteractionCallback, user slack_model.User) error {
	// Extract current values from the modal state
	values := interaction.View.State.Values
	metadata := interaction.View.PrivateMetadata

	sv := slack_model.BlockActionFromValue(values)

	return x.interaction.HandleSalvageRefresh(ctx, user, metadata, sv, interaction.View.ID)
}
