package slack

import (
	"context"
	"strings"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	slack_model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/async"
	"github.com/secmon-lab/warren/pkg/utils/authctx"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/user"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func dispatch(ctx context.Context, handler func(ctx context.Context) error) {
	if IsSync(ctx) {
		if err := handler(ctx); err != nil {
			errutil.Handle(ctx, err)
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

	// Add Subject from Slack user
	if subject, err := authctx.NewSubjectFromSlackUser(event.User); err == nil {
		ctx = authctx.WithSubject(ctx, subject)
	} else {
		logging.From(ctx).Warn("failed to create Subject from Slack user", "error", err)
	}

	slackMsg := slack_model.NewMessage(ctx, apiEvent)
	if slackMsg == nil {
		logger.Debug("slack app mention: message is nil, skipping")
		return nil
	}

	logger.Debug("slack app mention: dispatching async handler",
		"channel", event.Channel,
		"user", event.User,
	)

	dispatch(ctx, func(ctx context.Context) error {
		return x.event.HandleSlackAppMention(ctx, *slackMsg)
	})

	return nil
}

func (x *Controller) HandleSlackMessage(ctx context.Context, apiEvent *slackevents.EventsAPIEvent, event *slackevents.MessageEvent) error {
	logger := logging.From(ctx).With("event_ts", event.EventTimeStamp)
	ctx = logging.With(ctx, logger)

	// Extract user ID considering message subtypes.
	// For message_changed events, the user info is in the nested Message field.
	userID := resolveMessageEventUserID(event)

	// Set user context from Slack event
	ctx = user.WithUserID(ctx, userID)

	// Add Subject from Slack user
	if subject, err := authctx.NewSubjectFromSlackUser(userID); err == nil {
		ctx = authctx.WithSubject(ctx, subject)
	} else {
		logging.From(ctx).Warn("failed to create Subject from Slack user", "error", err)
	}

	logger.Debug("slack message event", "event", event)

	slackMsg := slack_model.NewMessage(ctx, apiEvent)
	if slackMsg == nil {
		return nil
	}

	if !slackMsg.InThread() {
		return nil
	}

	dispatch(ctx, func(ctx context.Context) error {
		return x.event.HandleSlackMessage(ctx, *slackMsg)
	})

	return nil
}

// resolveMessageEventUserID extracts the user ID from a MessageEvent,
// handling subtypes where the user info is in a nested field.
func resolveMessageEventUserID(event *slackevents.MessageEvent) string {
	if event.User != "" {
		return event.User
	}
	if event.SubType == "message_changed" && event.Message != nil {
		return event.Message.User
	}
	return ""
}

func (x *Controller) HandleSlackInteraction(ctx context.Context, interaction slack.InteractionCallback) error {
	logger := logging.From(ctx)
	logger.Info("slack interaction event", "event", interaction)

	// Set user context from Slack interaction
	ctx = user.WithUserID(ctx, interaction.User.ID)

	// Add Subject from Slack user
	if subject, err := authctx.NewSubjectFromSlackUser(interaction.User.ID); err == nil {
		ctx = authctx.WithSubject(ctx, subject)
	} else {
		logging.From(ctx).Warn("failed to create Subject from Slack user", "error", err)
	}

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
	logger := logging.From(ctx)

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

	// Process only the first action (Slack interactions typically have one action per callback)
	for _, action := range interaction.ActionCallback.BlockActions {
		actionID := action.ActionID
		value := action.Value

		// Handle HITL actions with state values for comment/answer extraction
		if actionID == slack_model.ActionIDHITLApprove.String() || actionID == slack_model.ActionIDHITLDeny.String() {
			status := hitl.StatusApproved
			if actionID == slack_model.ActionIDHITLDeny.String() {
				status = hitl.StatusDenied
			}

			response := extractHITLResponse(interaction.BlockActionState)
			return x.interaction.HandleHITLAction(ctx, user, types.HITLRequestID(value), status, response)
		}

		// Handle HITL question submit
		if actionID == slack_model.ActionIDHITLSubmitAnswer.String() {
			response := extractHITLResponse(interaction.BlockActionState)
			if _, ok := response["answer"]; !ok {
				logger.Warn("HITL submit answer without selection, ignoring")
				return nil
			}
			return x.interaction.HandleHITLAction(ctx, user, types.HITLRequestID(value), hitl.StatusApproved, response)
		}

		// Handle overflow menu actions (e.g., notice actions)
		// Overflow menu uses action.SelectedOption.Value in format "actionType:parameter"
		if actionID == "notice_actions" && action.SelectedOption.Value != "" {
			parts := strings.SplitN(action.SelectedOption.Value, ":", 2)
			if len(parts) != 2 {
				logger.Warn("invalid overflow menu value format", "value", action.SelectedOption.Value)
				continue
			}
			// Extract actual action ID and parameter from the value
			actionID = parts[0]
			value = parts[1]
			logger.Debug("parsed overflow menu action", "action_id", actionID, "value", value)
		}

		return x.interaction.HandleSlackInteractionBlockActions(ctx, user, th, slack_model.ActionID(actionID), value, interaction.TriggerID)
	}

	return nil
}

// extractHITLResponse extracts answer and comment from Slack block action state values.
func extractHITLResponse(state *slack.BlockActionStates) map[string]any {
	response := map[string]any{}
	if state == nil {
		return response
	}

	// Extract radio button answer (for questions)
	if block, ok := state.Values[slack_model.BlockIDHITLAnswer.String()]; ok {
		if answerAction, ok := block[slack_model.BlockActionIDHITLAnswer.String()]; ok {
			if answerAction.SelectedOption.Value != "" {
				response["answer"] = answerAction.SelectedOption.Value
			}
		}
	}

	// Extract comment
	if block, ok := state.Values[slack_model.BlockIDHITLComment.String()]; ok {
		if commentAction, ok := block[slack_model.BlockActionIDHITLComment.String()]; ok {
			if commentAction.Value != "" {
				response["comment"] = commentAction.Value
			}
		}
	}

	return response
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
