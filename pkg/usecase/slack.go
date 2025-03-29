package usecase

import (
	"context"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	session_model "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/group"
	"github.com/secmon-lab/warren/pkg/service/list"
	session_svc "github.com/secmon-lab/warren/pkg/service/session"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

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

// HandleSlackAppMention handles a slack app mention event. It will dispatch a slack action to the alert.
func (uc *UseCases) HandleSlackAppMention(ctx context.Context, user slack.User, mention slack.Mention, thread slack.Thread) error {
	uc.dispatchSlackAction(ctx, func(ctx context.Context) error {
		return uc.handleSlackAppMention(ctx, user, mention, thread)
	})
	return nil
}

func (uc *UseCases) handleSlackAppMention(ctx context.Context, user slack.User, mention slack.Mention, thread slack.Thread) error {

	logger := logging.From(ctx)
	logger.Debug("slack app mention event", "mention", mention, "slack_thread", thread)

	st := uc.slackService.NewThread(thread)
	ctx = msg.With(ctx, st.Reply, st.NewStateFunc)

	// Nothing to do
	if !uc.slackService.IsBotUser(mention.UserID) {
		return nil
	}
	if len(mention.Message) == 0 {
		msg.Notify(ctx, "🤔 No message")
		return nil
	}

	var targetAlerts alert.Alerts

	// If session is not found, starting a new session based on existing alert or list
	if v, err := uc.repository.GetAlertByThread(ctx, thread); err != nil {
		return goerr.Wrap(err, "failed to get alert by slack thread")
	} else if v != nil {
		targetAlerts = alert.Alerts{v}
	}

	if v, err := uc.repository.GetAlertListByThread(ctx, thread); err != nil {
		return goerr.Wrap(err, "failed to get alert list by slack thread")
	} else if v != nil {
		targetAlerts = v.Alerts
	}

	if err := uc.handleSlackRootCommand(ctx, st, user, targetAlerts, mention.Message); err == nil {
		// If the command is handled, return nil
		return nil
	} else if err != errUnknownCommand {
		return err
	}

	if len(targetAlerts) == 0 {
		msg.Notify(ctx, "🤔 No alerts found")
		return nil
	}

	// If session is found, dispatch the action to the existing session
	ssn, err := uc.repository.GetSessionByThread(ctx, thread)
	if err != nil {
		return goerr.Wrap(err, "failed to get session by slack thread")
	}
	if ssn == nil {
		targetAlertIDs := []types.AlertID{}
		for _, alert := range targetAlerts {
			targetAlertIDs = append(targetAlertIDs, alert.ID)
		}
		ssn = session_model.New(ctx, user, thread, targetAlertIDs)
		if err := uc.repository.PutSession(ctx, *ssn); err != nil {
			return goerr.Wrap(err, "failed to put session")
		}
	}

	// If session, alert and alert list are not found, call the command handler
	svc := session_svc.New(uc.repository, uc.llmClient, uc.slackService, ssn)
	if err := svc.Chat(ctx, mention.Message); err != nil {
		return goerr.Wrap(err, "failed to run session")
	}

	return nil
}

var (
	errUnknownCommand = goerr.New("unknown command")
)

func (uc *UseCases) handleSlackRootCommand(ctx context.Context, th *slack_svc.ThreadService, user slack.User, alerts alert.Alerts, message string) error {
	args := strings.SplitN(message, " ", 2)
	if len(args) == 0 {
		return errUnknownCommand
	}

	remaining := ""
	if len(args) == 2 {
		remaining = strings.TrimSpace(args[1])
	}
	command := strings.ToLower(strings.TrimSpace(args[0]))
	switch command {
	case "list":
		_, err := list.New(uc.repository).Run(ctx, th, &user, remaining)
		if err != nil {
			return goerr.Wrap(err, "failed to run list command")
		}
		return nil

	case "group":
		if err := group.Run(ctx, uc.repository, th, user, alerts, remaining); err != nil {
			return goerr.Wrap(err, "failed to run group command")
		}
		return nil

	default:
		return errUnknownCommand
	}
}

func (uc *UseCases) handleSlackThreadCommand(ctx context.Context, th *slack_svc.ThreadService, user slack.User, message string) error {

	return nil
}

// HandleSlackMessage handles a message from a slack user. It saves the message as an alert comment if the message is in the Alert thread.
func (uc *UseCases) HandleSlackMessage(ctx context.Context, thread slack.Thread, text string, user slack.User, ts string) error {
	logger := logging.From(ctx)
	th := uc.slackService.NewThread(thread)
	ctx = msg.With(ctx, th.Reply, th.NewStateFunc)

	// Skip if the message is from the bot
	if uc.slackService.IsBotUser(user.ID) {
		return nil
	}

	baseAlert, err := uc.repository.GetAlertByThread(ctx, thread)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert by slack thread")
	}
	if baseAlert == nil {
		logger.Info("alert not found", "thread", thread)
		return nil
	}

	comment := alert.AlertComment{
		AlertID:   baseAlert.ID,
		Comment:   text,
		Timestamp: ts,
		User:      user,
	}
	if err := uc.repository.PutAlertComment(ctx, comment); err != nil {
		msg.Trace(ctx, "💥 Failed to insert alert comment\n> %s", err.Error())
		return goerr.Wrap(err, "failed to insert alert comment", goerr.V("comment", comment))
	}

	return nil
}
