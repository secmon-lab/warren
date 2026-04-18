package usecase

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/goerr/v2"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

// HandleSlackMessage handles a message from a slack user. It saves the
// message both as a legacy ticket.Comment and as a session.Message with
// Type=user on the Slack Session. The legacy persistence remains to keep
// pre-redesign readers (graphql / base tool) functional until Phase 3.4
// migrates callers; the session.Message path is the new source of truth
// and is what the redesigned chat prompts will consume.
func (uc *UseCases) HandleSlackMessage(ctx context.Context, slackMsg slack.Message) error {
	logger := logging.From(ctx)
	if uc.slackService == nil {
		return goerr.New("slack service not configured")
	}
	th := uc.slackService.NewThread(slackMsg.Thread())
	traceFunc := func(ctx context.Context, message string) {
		th.NewTraceMessage(ctx, message)
	}
	ctx = msg.With(ctx, th.Reply, traceFunc, createSlackWarnFunc(th))

	// Set user ID in context for activity tracking and skip if the message is from the bot
	if slackMsg.User() != nil {
		ctx = user.WithUserID(ctx, slackMsg.User().ID)
		if uc.slackService.IsBotUser(slackMsg.User().ID) {
			return nil
		}
	}

	existingTicket, err := uc.repository.GetTicketByThread(ctx, slackMsg.Thread())
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket by slack thread")
	}
	if existingTicket == nil {
		logger.Info("ticket not found", "slack_thread", slackMsg.Thread())
		return nil
	}

	// Legacy write: ticket.Comment remains the input for Slack-origin
	// comments across GraphQL, base tool, refine, and the chat prompts
	// until subsequent phases migrate all readers off it.
	comment := existingTicket.NewComment(ctx, slackMsg.Text(), slackMsg.User(), slackMsg.ID())
	if err := uc.repository.PutTicketComment(ctx, comment); err != nil {
		if data, jsonErr := json.Marshal(comment); jsonErr == nil {
			logger.Error("failed to save ticket comment", "error", err, "comment", string(data))
		}
		msg.Trace(ctx, "💥 Failed to insert alert comment\n> %s", err.Error())
		return goerr.Wrap(err, "failed to insert alert comment", goerr.V("comment", comment))
	}

	// New write: mirror the message into the Slack Session as a
	// type=user Message with TurnID=nil (non-mention Slack thread
	// messages do not belong to any AI Turn). Failures are logged but do
	// not propagate, so a migration-era issue cannot break the existing
	// Slack comment pipeline.
	uc.persistSlackThreadMessageAsSessionMessage(ctx, existingTicket.ID, slackMsg)

	return nil
}

// persistSlackThreadMessageAsSessionMessage resolves (or creates) the
// Slack Session for this thread and appends a type=user Message for the
// incoming slackMsg. Used by HandleSlackMessage to mirror legacy
// ticket.Comment writes into the new Message timeline.
func (uc *UseCases) persistSlackThreadMessageAsSessionMessage(ctx context.Context, ticketID types.TicketID, slackMsg slack.Message) {
	if uc.sessionResolver == nil {
		return
	}
	sess, _, err := uc.sessionResolver.ResolveSlackSession(ctx, &ticketID, slackMsg.Thread(), types.UserID(slackUserID(&slackMsg)))
	if err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to resolve slack session for thread message",
			goerr.V("ticket_id", ticketID),
			goerr.V("thread", slackMsg.Thread()),
		))
		return
	}

	author := authorFromSlackUser(slackMsg.User())
	if author == nil {
		// Messages without a user (rare; typically Slack system events
		// that sneak through IsBotUser) are skipped here — they have no
		// meaningful authorship in the Session timeline.
		return
	}
	tid := ticketID
	m := sessModel.NewMessageV2(ctx, sess.ID, &tid, nil, sessModel.MessageTypeUser, slackMsg.Text(), author)
	if err := uc.repository.PutSessionMessage(ctx, m); err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to persist Slack user message as session.Message",
			goerr.V("session_id", sess.ID),
			goerr.V("ticket_id", ticketID),
		))
	}
}

// authorFromSlackUser builds a session.Author from a Slack user. Returns
// nil when u is nil so callers can cheaply decide to skip persistence.
func authorFromSlackUser(u *slack.User) *sessModel.Author {
	if u == nil {
		return nil
	}
	slackID := u.ID
	display := u.Name
	if display == "" {
		display = u.ID
	}
	return &sessModel.Author{
		UserID:      types.UserID(u.ID),
		DisplayName: display,
		SlackUserID: &slackID,
	}
}
