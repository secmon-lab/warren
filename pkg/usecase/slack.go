package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
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
	logger := logging.From(ctx)
	logger.Debug("slack app mention event", "mention", mention, "slack_thread", thread)

	st := uc.slackService.NewThread(thread)
	ctx = msg.With(ctx, st.Reply, st.NewStateFunc)

	// Nothing to do
	if !uc.slackService.IsBotUser(mention.UserID) {
		return nil
	}

	alert, err := uc.repository.GetAlertByThread(ctx, thread)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert by slack thread")
	}
	session, err := uc.repository.GetSessionByThread(ctx, thread)
	if err != nil {
		return goerr.Wrap(err, "failed to get session by slack thread")
	}

	if len(mention.Args) == 0 {
		msg.Reply(ctx, "⏸️ No action specified")
		return nil
	}

	arguments := append([]string{"warren"}, mention.Args...)
	uc.dispatchSlackAction(ctx, func(ctx context.Context) error {
		// TODO: Implement

		// If alert is not fo
		if alert != nil {
			logger.Info("alert found", "alert", alert)
		}
		if session != nil {
			logger.Info("session found", "session", session)
		}

		logger.Info("dispatch slack action", "arguments", arguments)
		return nil
	})

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
		msg.State(ctx, "💥 Failed to insert alert comment\n> %s", err.Error())
		return goerr.Wrap(err, "failed to insert alert comment", goerr.V("comment", comment))
	}

	return nil
}
