package usecase

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/errs"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/thread"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func (uc *UseCases) HandleSlackAppMention(ctx context.Context, event *slackevents.AppMentionEvent) error {
	logger := logging.From(ctx).With("event_ts", event.EventTimeStamp)
	ctx = logging.With(ctx, logger)

	logger.Debug("slack app mention event", "event", event)

	threadData := model.SlackThread{
		ChannelID: event.Channel,
		ThreadID:  event.ThreadTimeStamp,
	}
	if threadData.ThreadID == "" {
		threadData.ThreadID = event.TimeStamp
	}

	alerts, err := uc.repository.GetAlertsBySlackThread(ctx, threadData)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert by slack thread")
	}

	mention := uc.slackService.TrimMention(event.Text)
	if mention == "" {
		logger.Warn("slack app mention event is empty", "event", event)
		return nil
	}

	args := parseArgs(mention)

	logger.Info("slack app mention event", "mention", mention, "thread", threadData)
	logger.Debug("Parsed args", "args", args)

	th := uc.slackService.NewThread(threadData)
	ctx = thread.WithReplyFunc(ctx, th.Reply)

	if len(args) == 0 {
		th.Reply(ctx, "⏸️ No action specified")
		return nil
	}

	arguments := append([]string{"warren"}, args...)
	uc.dispatchSlackAction(ctx, func(ctx context.Context) error {
		return uc.RunCommand(ctx, arguments, alerts, th, &model.SlackUser{
			ID:   event.User,
			Name: event.User,
		})
	})

	return nil
}

func (uc *UseCases) dispatchSlackAction(ctx context.Context, action func(ctx context.Context) error) {
	newCtx := newBackgroundContext(ctx)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errs.Handle(newCtx, goerr.New("panic", goerr.V("recover", r)))
			}
		}()

		if err := action(newCtx); err != nil {
			errs.Handle(newCtx, err)
		}
	}()
}

func (uc *UseCases) HandleSlackMessage(ctx context.Context, event *slackevents.MessageEvent) error {
	logger := logging.From(ctx)
	logger.Debug("slack message event", "event", event)

	if event.ThreadTimeStamp == "" {
		return nil
	}

	thread := model.SlackThread{
		ChannelID: event.Channel,
		ThreadID:  event.ThreadTimeStamp,
	}
	alerts, err := uc.repository.GetAlertsBySlackThread(ctx, thread)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert by slack thread")
	}
	if len(alerts) == 0 {
		logger.Info("alert not found", "thread", thread)
		return nil
	}

	var baseAlert *model.Alert
	for _, a := range alerts {
		if a.ParentID == "" {
			baseAlert = &a
		}
	}
	if baseAlert == nil {
		logger.Warn("base alert not found", "thread", thread)
		return nil
	}

	comment := model.AlertComment{
		AlertID:   baseAlert.ID,
		Comment:   event.Text,
		Timestamp: event.EventTimeStamp,
		UserID:    event.User,
	}
	if err := uc.repository.InsertAlertComment(ctx, comment); err != nil {
		return goerr.Wrap(err, "failed to insert alert comment", goerr.V("comment", comment))
	}

	return nil
}

func (uc *UseCases) HandleSlackInteraction(ctx context.Context, interaction slack.InteractionCallback) error {
	logger := logging.From(ctx)
	logger.Info("slack interaction event", "event", interaction)

	th := uc.slackService.NewThread(model.SlackThread{
		ChannelID: interaction.Channel.ID,
		ThreadID:  interaction.Message.ThreadTimestamp,
	})
	ctx = thread.WithReplyFunc(ctx, th.Reply)

	handler := func(ctx context.Context) error {
		switch interaction.Type {
		case slack.InteractionTypeBlockActions:
			return uc.handleSlackInteractionBlockActions(ctx, interaction)
		case slack.InteractionTypeViewSubmission:
			return uc.handleSlackInteractionViewSubmission(ctx, interaction)
		}

		return nil
	}

	if IsSync(ctx) {
		if err := handler(ctx); err != nil {
			return goerr.Wrap(err, "failed to handle slack interaction")
		}
	} else {
		uc.dispatchSlackAction(ctx, handler)
	}

	return nil
}

func (uc *UseCases) handleSlackInteractionViewSubmission(ctx context.Context, interaction slack.InteractionCallback) error {
	switch model.SlackCallbackID(interaction.View.CallbackID) {
	case model.SlackCallbackSubmitResolveAlert:
		return uc.handleSlackInteractionViewSubmissionResolveAlert(ctx, interaction)
	case model.SlackCallbackSubmitResolveList:
		return uc.handleSlackInteractionViewSubmissionResolveList(ctx, interaction)
	}

	return nil
}

func (uc *UseCases) handleSlackInteractionViewSubmissionResolveAlert(ctx context.Context, interaction slack.InteractionCallback) error {
	logger := logging.From(ctx)

	alertID := model.AlertID(interaction.View.PrivateMetadata)
	alert, err := uc.repository.GetAlert(ctx, alertID)
	if err != nil {
		thread.Reply(ctx, "💥 Failed to get alert\n> "+err.Error())
		return goerr.Wrap(err, "failed to get alert")
	}
	if alert == nil {
		thread.Reply(ctx, "💥 Alert not found")
		return nil
	}

	if err := uc.handleSlackInteractionViewSubmissionResolve(ctx, interaction, []model.Alert{*alert}); err != nil {
		logger.Error("failed to resolve alert", "error", err)
		return err
	}

	return nil
}

func (uc *UseCases) handleSlackInteractionViewSubmissionResolveList(ctx context.Context, interaction slack.InteractionCallback) error {
	listID := model.AlertListID(interaction.View.PrivateMetadata)
	list, err := uc.repository.GetAlertList(ctx, listID)
	if err != nil {
		thread.Reply(ctx, "💥 Failed to get alert list\n> "+err.Error())
		return goerr.Wrap(err, "failed to get alert list")
	}

	alerts, err := uc.repository.BatchGetAlerts(ctx, list.AlertIDs)
	if err != nil {
		thread.Reply(ctx, "💥 Failed to get alerts\n> "+err.Error())
		return goerr.Wrap(err, "failed to get alerts")
	}

	if err := uc.handleSlackInteractionViewSubmissionResolve(ctx, interaction, alerts); err != nil {
		thread.Reply(ctx, "💥 Failed to resolve alerts of list\n> "+err.Error())
		return goerr.Wrap(err, "failed to resolve alerts")
	}

	return nil
}

func (uc *UseCases) handleSlackInteractionViewSubmissionResolve(ctx context.Context, interaction slack.InteractionCallback, alerts []model.Alert) error {
	thread.Reply(ctx, fmt.Sprintf("⏳ Resolving %d alerts...", len(alerts)))

	var (
		conclusion model.AlertConclusion
		reason     string
	)
	if conclusionBlock, ok := interaction.View.State.Values[model.SlackBlockIDConclusion.String()]; ok {
		if conclusionAction, ok := conclusionBlock[model.SlackActionIDConclusion.String()]; ok {
			conclusion = model.AlertConclusion(conclusionAction.SelectedOption.Value)
		}
	}
	if commentBlock, ok := interaction.View.State.Values[model.SlackBlockIDComment.String()]; ok {
		if commentAction, ok := commentBlock[model.SlackActionIDComment.String()]; ok {
			reason = commentAction.Value
		}
	}

	if err := conclusion.Validate(); err != nil {
		return goerr.Wrap(err, "invalid conclusion", goerr.V("conclusion", conclusion))
	}

	now := clock.Now(ctx)
	for _, alert := range alerts {
		alert.Status = model.AlertStatusResolved
		alert.ResolvedAt = &now
		alert.Conclusion = conclusion
		alert.Reason = reason
		if alert.Assignee == nil {
			alert.Assignee = &model.SlackUser{
				ID:   interaction.User.ID,
				Name: interaction.User.Name,
			}
		}

		if err := uc.repository.PutAlert(ctx, alert); err != nil {
			return goerr.Wrap(err, "failed to put alert")
		}

		th := uc.slackService.NewThread(*alert.SlackThread)
		ctx = thread.WithReplyFunc(ctx, th.Reply)
		th.Reply(ctx, "Alert resolved by <@"+interaction.User.ID+">")

		if alert.ParentID == "" {
			if err := th.UpdateAlert(ctx, alert); err != nil {
				return goerr.Wrap(err, "failed to update slack thread")
			}
		}
	}

	thread.Reply(ctx, fmt.Sprintf("✅️ Resolved %d alerts", len(alerts)))

	return nil
}

func (uc *UseCases) handleSlackInteractionBlockActions(ctx context.Context, interaction slack.InteractionCallback) error {
	logger := logging.From(ctx)

	action := interaction.ActionCallback.BlockActions[0]

	th := uc.slackService.NewThread(model.SlackThread{
		ChannelID: interaction.Channel.ID,
		ThreadID:  interaction.Message.ThreadTimestamp,
	})
	ctx = thread.WithReplyFunc(ctx, th.Reply)

	switch model.SlackActionID(action.ActionID) {
	case model.SlackActionIDAck:
		alert, err := uc.repository.GetAlert(ctx, model.AlertID(action.Value))
		if err != nil {
			return goerr.Wrap(err, "failed to get alert")
		} else if alert == nil {
			logger.Error("alert not found", "alert_id", action.Value)
			return nil
		}

		alert.Assignee = &model.SlackUser{
			ID:   interaction.User.ID,
			Name: interaction.User.Name,
		}
		alert.Status = model.AlertStatusAcknowledged
		if err := uc.repository.PutAlert(ctx, *alert); err != nil {
			return goerr.Wrap(err, "failed to put alert")
		}

		if alert.SlackThread != nil {
			thread := uc.slackService.NewThread(*alert.SlackThread)
			thread.Reply(ctx, "Alert acknowledged by <@"+interaction.User.ID+">")

			if err := thread.UpdateAlert(ctx, *alert); err != nil {
				return goerr.Wrap(err, "failed to update slack thread")
			}
		} else {
			logger.Warn("slack thread not found", "alert_id", alert.ID)
		}

	case model.SlackActionIDResolve:
		alert, err := uc.repository.GetAlert(ctx, model.AlertID(action.Value))
		if err != nil {
			return goerr.Wrap(err, "failed to get alert")
		} else if alert == nil {
			logger.Error("alert not found", "alert_id", action.Value)
			return nil
		}

		triggerID := interaction.TriggerID

		if svc, ok := uc.slackService.(*service.Slack); ok {
			if err := svc.ShowResolveAlertModal(ctx, *alert, triggerID); err != nil {
				return goerr.Wrap(err, "failed to show resolve alert modal")
			}
		} else {
			logger.Warn("slack service is not available")
		}

	case model.SlackActionIDInspect:
		alert, err := uc.repository.GetAlert(ctx, model.AlertID(action.Value))
		if err != nil {
			return goerr.Wrap(err, "failed to get alert")
		} else if alert == nil {
			logger.Error("alert not found", "alert_id", action.Value)
			return nil
		}

		if err := uc.RunWorkflow(ctx, *alert); err != nil {
			return err
		}

	case model.SlackActionIDIgnoreList:
		return uc.RunCommand(ctx, []string{"warren", "ignore", action.Value}, nil, th, &model.SlackUser{
			ID:   interaction.User.ID,
			Name: interaction.User.Name,
		})

	case model.SlackActionIDResolveList:
		return uc.RunCommand(ctx, []string{"warren", "resolve", action.Value}, nil, th, &model.SlackUser{
			ID:   interaction.User.ID,
			Name: interaction.User.Name,
		})

	case model.SlackActionIDCreatePR:
		th.Reply(ctx, "✏️ Creating pull request...")

		diffID := model.PolicyDiffID(action.Value)
		diff, err := uc.repository.GetPolicyDiff(ctx, diffID)
		if err != nil {
			thread.Reply(ctx, "💥 Failed to get policy diff\n> "+err.Error())
			return goerr.Wrap(err, "failed to get policy diff")
		} else if diff == nil {
			thread.Reply(ctx, "💥 Policy diff not found")
			return nil
		}

		if uc.gitHubApp == nil {
			thread.Reply(ctx, "💥 GitHub is not enabled")
			return nil
		}

		prURL, err := uc.gitHubApp.CreatePullRequest(ctx, diff)
		if err != nil {
			thread.Reply(ctx, "💥 Failed to create pull request\n> "+err.Error())
			return err
		}

		thread.Reply(ctx, fmt.Sprintf("✅️ Created: <%s|%s>", prURL.String(), diff.Title))
	}

	return nil
}

// parseArgs parses a string into arguments, handling various types of quotes
func parseArgs(input string) []string {
	var result []string
	var current []rune
	var inQuotes bool
	var quoteChar rune

	// Unicode code points for quotes
	const (
		leftDoubleQuote  = '\u201c' // "
		rightDoubleQuote = '\u201d' // "
		leftSingleQuote  = '\u2018' // '
		rightSingleQuote = '\u2019' // '
	)

	// isMatchingQuote checks if two quote characters form a matching pair
	isMatchingQuote := func(open, close rune) bool {
		return open == close || // Same quotes
			(open == leftDoubleQuote && close == rightDoubleQuote) || // Unicode double quotes
			(open == leftSingleQuote && close == rightSingleQuote) // Unicode single quotes
	}

	for i := 0; i < len(input); {
		char, size := utf8.DecodeRuneInString(input[i:])
		i += size

		switch char {
		case '\\':
			if i < len(input) {
				nextChar, size := utf8.DecodeRuneInString(input[i:])
				if nextChar == '\\' || nextChar == '"' || nextChar == '\'' ||
					nextChar == leftDoubleQuote || nextChar == rightDoubleQuote ||
					nextChar == leftSingleQuote || nextChar == rightSingleQuote ||
					nextChar == '`' {
					current = append(current, nextChar)
					i += size
				} else {
					current = append(current, char)
				}
			} else {
				current = append(current, char)
			}
		case '"', '\'', leftDoubleQuote, rightDoubleQuote, leftSingleQuote, rightSingleQuote, '`':
			if inQuotes {
				if isMatchingQuote(quoteChar, char) {
					inQuotes = false
					if len(current) > 0 {
						result = append(result, string(current))
						current = nil
					}
				} else {
					current = append(current, char)
				}
			} else {
				inQuotes = true
				quoteChar = char
				if len(current) > 0 {
					result = append(result, string(current))
					current = nil
				}
			}
		case ' ':
			if inQuotes {
				current = append(current, char)
			} else if len(current) > 0 {
				result = append(result, string(current))
				current = nil
			}
		default:
			current = append(current, char)
		}
	}

	if len(current) > 0 {
		result = append(result, string(current))
	}

	return result
}
