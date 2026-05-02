package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

// HandleSlackMessage handles a message from a slack user. It persists
// the message as a session.Message (type=user) on the Slack Session
// bound to the ticket's thread. Pre-redesign ticket.Comment writes have
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

	// chat-session-redesign Phase 7 (confinement): the legacy
	// ticket.Comment write path has been removed. Slack-origin thread
	// messages are persisted exclusively as type=user
	// session.Message(TurnID=nil) rows. Pre-redesign Comment documents
	// remain in production indefinitely (the migration never deletes
	// them); they are accessible only via the migration package.
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

	// Resolve the Slack user's display name and cache it on Author so
	// downstream readers (frontend MessageBubble, GraphQL) show a
	// human-readable name instead of the raw Slack ID. Slack events
	// only carry the user ID, so we have to hit the Slack API — the
	// slackService caches profiles to avoid per-message calls.
	if uc.slackService != nil && slackMsg.User() != nil {
		if profile, profErr := uc.slackService.GetUserProfile(ctx, slackMsg.User().ID); profErr == nil && profile != "" {
			author.DisplayName = profile
		} else if profErr != nil {
			// Profile lookup failures are non-fatal — fall back to the
			// author as constructed (DisplayName = user ID).
			logging.From(ctx).Debug("failed to resolve slack profile for message author",
				"error", profErr, "user_id", slackMsg.User().ID)
		}
	}

	tid := ticketID
	// Dedupe on (SessionID + Slack ts): Slack fires both `message` and
	// (when the message contains a URL) `message_changed` events for
	// the same logical message, and Events API retries also arrive if
	// the initial 200 OK is slow. Using a deterministic MessageID
	// collapses all of these into a single row. slack_ts is unique per
	// Slack message within a workspace, so sessionID+ts is sufficient.
	ts := slackMsg.Timestamp()
	externalKey := string(sess.ID) + "|" + ts
	m := sessModel.NewMessageV2(ctx, sess.ID, &tid, nil, sessModel.MessageTypeUser, slackMsg.Text(), author)
	if ts != "" {
		m.ID = types.DeterministicMessageID(externalKey)
	}
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
