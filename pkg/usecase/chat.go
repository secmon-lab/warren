package usecase

import (
	"context"
	_ "embed"
	"errors"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	hitlModel "github.com/secmon-lab/warren/pkg/domain/model/hitl"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/tool/base"
	knowledgeTool "github.com/secmon-lab/warren/pkg/tool/knowledge"
	chatpkg "github.com/secmon-lab/warren/pkg/usecase/chat"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/slackctx"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

// EnsureWebSession creates a Session scoped to the current WebSocket
// connection. Callers (controller/websocket.Handler) invoke this once
// when the client connects and pass the returned Session back into
// ChatFromWebSocket for every subsequent user message, so the whole
// connection shares one Session and one gollem working-memory slot.
func (x *UseCases) EnsureWebSession(ctx context.Context, ticketID types.TicketID, userID types.UserID) (*sessModel.Session, error) {
	if x.sessionResolver == nil {
		return nil, goerr.New("session resolver not configured")
	}
	return x.sessionResolver.CreateFreshSession(ctx, ticketID, sessModel.SessionSourceWeb, userID)
}

// EnsureCLISession is the CLI-side counterpart to EnsureWebSession.
// The CLI main creates one Session per launch (non-interactive: one
// Turn; interactive: many Turns under the same Session).
func (x *UseCases) EnsureCLISession(ctx context.Context, ticketID types.TicketID, userID types.UserID) (*sessModel.Session, error) {
	if x.sessionResolver == nil {
		return nil, goerr.New("session resolver not configured")
	}
	return x.sessionResolver.CreateFreshSession(ctx, ticketID, sessModel.SessionSourceCLI, userID)
}

// ChatTurnEvent describes a lifecycle event produced by a Web chat
// invocation so the WebSocket handler can publish envelope-format
// updates to the bound client. Emitted for "turn_started",
// "session_message_added", "session_message_updated", "turn_ended",
// "hitl_request_pending", and "hitl_request_resolved"; the payload
// field set depends on the event kind.
type ChatTurnEvent struct {
	Kind        string
	Turn        *sessModel.Turn
	Message     *sessModel.Message
	HITLRequest *hitlModel.Request
	// HITLMessageID binds a HITL prompt to the progress message that
	// hosts it, so the frontend can render approval UI in-place rather
	// than as a floating banner.
	HITLMessageID string
}

// ChatFromWebSocket processes a single user message inside an already-
// established Web Session. Each invocation creates a fresh Turn, stamps
// the user message with the resulting TurnID, runs the agent, and
// closes the Turn — guaranteeing that every WS message appears in the
// conversation timeline with a stable TurnID rather than as orphaned
// nil-TurnID rows. `onEvent` is invoked for each lifecycle event so
// the WebSocket handler can fan the Envelope out to the bound client;
// pass nil to disable.
func (x *UseCases) ChatFromWebSocket(ctx context.Context, ticketID types.TicketID, message string, sess *sessModel.Session, onEvent func(ChatTurnEvent)) error {
	t, err := x.repository.GetTicket(ctx, ticketID)
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket")
	}
	if t == nil {
		return goerr.New("ticket not found")
	}
	var hook func(event string, payload any)
	if onEvent != nil {
		hook = func(event string, payload any) {
			ev := ChatTurnEvent{Kind: event}
			switch v := payload.(type) {
			case *sessModel.Turn:
				ev.Turn = v
			case *sessModel.Message:
				ev.Message = v
			case hitlEventPayload:
				ev.HITLRequest = v.Request
				ev.HITLMessageID = v.MessageID
			}
			onEvent(ev)
		}
	}
	return x.executeChatTurn(ctx, t, sess, message, hook)
}

// ChatFromSlack processes a chat message from a Slack app mention.
//
// chat-session-redesign Phase 2 wrapping: before delegating to ChatUC.Execute,
// the Slack Session is resolved (or created with a deterministic ID so
// repeated mentions on the same thread reuse it), the Session activity lock
// is acquired so simultaneous mentions are serialized, and a Turn is
// recorded for the mention. The existing Execute code path is retained; it
// still creates its internal "session.Session" (legacy) for backwards
// compatibility with aster/bluebell, and Phase 3+ will unify the two.
//
// If the Session lock is already held when a new mention arrives, the
// function posts a short "please wait" context block to the thread and
// returns without starting a second AI run.
func (x *UseCases) ChatFromSlack(ctx context.Context, slackMsg *slack.Message, mentionMessage string) error {
	logger := logging.From(ctx)

	// Get Slack history for conversation context
	var history []slack.HistoryMessage
	if x.slackService != nil {
		var err error
		history, err = x.slackService.GetMessageHistory(ctx, slackMsg)
		if err != nil {
			logger.Warn("failed to get slack history", "error", err)
		}
	}

	// Try to find existing ticket by thread
	existingTicket, err := x.repository.GetTicketByThread(ctx, slackMsg.Thread())
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket by slack thread")
	}

	var ticketIDPtr *types.TicketID
	if existingTicket != nil {
		tid := existingTicket.ID
		ticketIDPtr = &tid
	}

	// Resolve (or create) the Slack Session + Turn + Lock. This wraps the
	// existing Execute call so that lock contention is enforced; the legacy
	// Session handling inside Execute is untouched.
	slackSess, _, err := x.sessionResolver.ResolveSlackSession(ctx, ticketIDPtr, slackMsg.Thread(), types.UserID(slackUserID(slackMsg)))
	if err != nil {
		logger.Warn("failed to resolve slack session; continuing without session lock", "error", err)
	}

	var releaseLock func()
	if slackSess != nil {
		lock, acquired, lockErr := x.lockService.TryAcquire(ctx, slackSess.ID)
		if lockErr != nil {
			logger.Warn("failed to acquire slack session lock; continuing", "error", lockErr)
		} else if !acquired {
			// Another mention is already running on this thread.
			if x.slackService != nil {
				threadSvc := x.slackService.NewThread(slackMsg.Thread())
				if postErr := threadSvc.PostContextBlock(ctx, "⏳ Another chat is still running on this thread. Please wait for it to finish."); postErr != nil {
					errutil.Handle(ctx, goerr.Wrap(postErr, "failed to post busy notice"))
				}
			}
			return nil
		} else if lock != nil {
			turn := sessModel.NewTurn(ctx, slackSess.ID)
			if putErr := x.repository.PutTurn(ctx, turn); putErr != nil {
				errutil.Handle(ctx, goerr.Wrap(putErr, "failed to persist slack turn"))
			}
			releaseLock = func() {
				// Close the Turn first so observers see terminal state
				// before the lock is released.
				turn.Close(ctx, sessModel.TurnStatusCompleted)
				if updErr := x.repository.UpdateTurnStatus(ctx, turn.ID, turn.Status, turn.EndedAt); updErr != nil {
					errutil.Handle(ctx, goerr.Wrap(updErr, "failed to update slack turn status"))
				}
				if relErr := lock.Release(ctx); relErr != nil {
					if !errors.Is(relErr, interfaces.ErrLockNotHeld) {
						errutil.Handle(ctx, goerr.Wrap(relErr, "failed to release slack session lock"))
					}
				}
			}
		}
	}
	if releaseLock != nil {
		defer releaseLock()
	}

	if existingTicket != nil {
		chatCtx, err := x.buildChatContext(ctx, existingTicket, history, mentionMessage, slackSess)
		if err != nil {
			return err
		}
		return x.ChatUC.Execute(ctx, mentionMessage, chatCtx)
	}

	// Ticketless chat
	ctx = slackctx.WithThread(ctx, slackMsg.Thread())
	thread := slackMsg.Thread()
	placeholderTicket := &ticket.Ticket{SlackThread: &thread}

	chatCtx, err := x.buildChatContext(ctx, placeholderTicket, history, mentionMessage, slackSess)
	if err != nil {
		return err
	}
	return x.ChatUC.Execute(ctx, mentionMessage, chatCtx)
}

// slackUserID extracts the sender's Slack user id for Session attribution.
// Returns an empty string when the message carries no user (rare, e.g.
// system-generated events).
func slackUserID(m *slack.Message) string {
	if m == nil {
		return ""
	}
	if u := m.User(); u != nil {
		return u.ID
	}
	return ""
}

// executeChatTurn runs one Turn inside an existing Web / CLI Session:
// persist Turn, stamp user message, run ChatUC.Execute, close Turn.
// `onEvent` is an optional hook invoked for envelope-level lifecycle
// events (turn_started / session_message_added / turn_ended) so the
// WebSocket handler can publish them to the client; pass nil for CLI
// where there is nothing to fan out.
func (x *UseCases) executeChatTurn(
	ctx context.Context,
	t *ticket.Ticket,
	sess *sessModel.Session,
	message string,
	onEvent func(event string, payload any),
) error {
	// When no Session has been resolved, fall through to the legacy
	// path so tests and unreachable branches keep working. Production
	// callers always provide sess != nil.
	if sess == nil {
		chatCtx, err := x.buildChatContext(ctx, t, nil, message, nil)
		if err != nil {
			return err
		}
		return x.ChatUC.Execute(ctx, message, chatCtx)
	}

	// Persist the Turn first. If the write fails we fall back to a
	// nil-TurnID flow so we never leave dangling TurnID references on
	// session.Messages that point at a Turn that was never saved.
	turn := sessModel.NewTurn(ctx, sess.ID)
	turnPersisted := true
	if err := x.repository.PutTurn(ctx, turn); err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to persist turn",
			goerr.V("session_id", sess.ID)))
		turnPersisted = false
	} else if onEvent != nil {
		onEvent("turn_started", turn)
	}

	// Persist the user input as a type=user Message.
	var turnIDPtr *types.TurnID
	if turnPersisted {
		tid := turn.ID
		turnIDPtr = &tid
	}
	tidCopy := t.ID
	var tidPtr *types.TicketID
	if t.ID != "" {
		tidPtr = &tidCopy
	}
	userID := types.UserID(user.FromContext(ctx))
	author := &sessModel.Author{
		UserID:      userID,
		DisplayName: string(userID),
	}
	userMsg := sessModel.NewMessageV2(ctx, sess.ID, tidPtr, turnIDPtr, sessModel.MessageTypeUser, message, author)
	if err := x.repository.PutSessionMessage(ctx, userMsg); err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to persist user input message",
			goerr.V("session_id", sess.ID),
		))
	} else if onEvent != nil {
		onEvent("session_message_added", userMsg)
	}

	chatCtx, err := x.buildChatContext(ctx, t, nil, message, sess)
	if err != nil {
		return err
	}
	// Exclude the user message we just persisted from the prompt
	// timeline so prompt templates don't see the current input twice
	// (once as `{{ .message }}` and once inside `session_messages`).
	chatCtx.SessionMessages = filterOutMessage(chatCtx.SessionMessages, userMsg.ID)
	chatCtx.CurrentTurnID = turnIDPtr
	if onEvent != nil {
		chatCtx.OnSessionEvent = func(kind string, m *sessModel.Message) {
			onEvent(kind, m)
		}
		// HITL events carry a hitl.Request (not a session.Message) so
		// they ride a dedicated envelope — the WebSocket handler maps
		// them to hitl_request_pending / hitl_request_resolved.
		chatCtx.OnHITLEvent = func(kind string, req *hitlModel.Request, messageID string) {
			onEvent("hitl_request_"+kind, hitlEventPayload{Request: req, MessageID: messageID})
		}
	}

	runErr := x.ChatUC.Execute(ctx, message, chatCtx)

	// Close the Turn only when we actually persisted it. Otherwise
	// UpdateTurnStatus would hit a missing row and turn_ended would
	// reference a nonexistent Turn.
	if turnPersisted {
		finalStatus := sessModel.TurnStatusCompleted
		if runErr != nil {
			finalStatus = sessModel.TurnStatusAborted
		}
		turn.Close(ctx, finalStatus)
		if updErr := x.repository.UpdateTurnStatus(ctx, turn.ID, turn.Status, turn.EndedAt); updErr != nil {
			errutil.Handle(ctx, goerr.Wrap(updErr, "failed to update turn status",
				goerr.V("turn_id", turn.ID)))
		}
		if onEvent != nil {
			onEvent("turn_ended", turn)
		}
	}

	return runErr
}

// hitlEventPayload is carried on "hitl_request_pending" /
// "hitl_request_resolved" onEvent invocations so the WebSocket handler
// can marshal both the Request and the optional progress-message
// binding without committing to a concrete envelope shape at this
// layer.
type hitlEventPayload struct {
	Request   *hitlModel.Request
	MessageID string
}

// filterOutMessage returns msgs with the row whose ID matches `id`
// removed. Used by executeChatTurn to keep the just-persisted user
// input from appearing twice in the prompt timeline.
func filterOutMessage(msgs []*sessModel.Message, id types.MessageID) []*sessModel.Message {
	out := msgs[:0]
	for _, m := range msgs {
		if m == nil || m.ID == id {
			continue
		}
		out = append(out, m)
	}
	return out
}

// ChatFromCLI processes a chat message from the CLI inside an already-
// established CLI Session. The CLI main is responsible for the Session
// lifetime: non-interactive mode calls EnsureCLISession+ChatFromCLI
// once, interactive mode reuses the Session across inputs so every
// Turn shares working memory. Pass sess==nil only in tests or the
// degenerate ticketless path.
func (x *UseCases) ChatFromCLI(ctx context.Context, t *ticket.Ticket, message string, sess *sessModel.Session) error {
	return x.executeChatTurn(ctx, t, sess, message, nil)
}

// buildChatContext fetches all data needed for chat execution and assembles a ChatContext.
//
// sess is the pre-resolved Session for this invocation (Slack: long-lived thread,
// Web/CLI: fresh per-call). When non-nil, gollem history is loaded from the
// Session-scoped storage slot and the Session's prior Message timeline is
// attached to chatCtx.SessionMessages so prompts can reference it.
func (x *UseCases) buildChatContext(ctx context.Context, t *ticket.Ticket, slackHistory []slack.HistoryMessage, _ string, sess *sessModel.Session) (chatModel.ChatContext, error) {
	ticketless := t.ID == ""

	chatCtx := chatModel.ChatContext{
		Ticket:       t,
		SlackHistory: slackHistory,
		Session:      sess,
	}

	// Fetch alerts (skip for ticketless)
	if !ticketless {
		alerts, err := x.repository.BatchGetAlerts(ctx, t.AlertIDs)
		if err != nil {
			return chatCtx, goerr.Wrap(err, "failed to get alerts")
		}
		chatCtx.Alerts = alerts
	}

	// Agent Memory search is removed — knowledge is now accessed via tool search (knowledge v2).

	// Build tools (convert interfaces.ToolSet to gollem.ToolSet for ChatContext)
	allTools := make([]gollem.ToolSet, 0, len(x.tools)+1)
	for _, ts := range x.tools {
		allTools = append(allTools, ts)
	}
	if !ticketless {
		slackUpdateFunc := func(ctx context.Context, updatedTicket *ticket.Ticket) error {
			if x.slackService == nil || !updatedTicket.HasSlackThread() || updatedTicket.Finding == nil {
				return nil
			}
			threadSvc := x.slackService.NewThread(*updatedTicket.SlackThread)
			return threadSvc.PostFinding(ctx, updatedTicket.Finding)
		}
		baseAction := base.New(x.repository, t.ID, base.WithSlackUpdate(slackUpdateFunc), base.WithLLMClient(x.llmClient))
		allTools = append(allTools, baseAction)
	}
	// Add knowledge search tool if knowledge service is configured
	if x.knowledgeSvc != nil {
		factTool := knowledgeTool.New(x.knowledgeSvc, types.KnowledgeCategoryFact, knowledgeTool.ModeReadOnly)
		allTools = append(allTools, factTool)
	}

	chatCtx.Tools = allTools

	// Load history (skip for ticketless and when no Session was
	// resolved — the latter leaves working memory empty so the next
	// turn starts fresh rather than resurrecting pre-redesign data).
	if !ticketless && sess != nil {
		storageSvc := storage.New(x.storageClient, storage.WithPrefix(x.storagePrefix))
		history, err := chatpkg.LoadSessionHistory(ctx, sess.ID, storageSvc)
		if err != nil {
			return chatCtx, err
		}
		chatCtx.History = history
	}

	// Populate the Session Message timeline (excluding the just-persisted
	// user input, which lives at the tail). Errors are logged and treated
	// as empty so a read hiccup never blocks the chat response.
	if sess != nil {
		msgs, err := x.repository.GetSessionMessages(ctx, sess.ID)
		if err != nil {
			logging.From(ctx).Warn("failed to load session messages", "error", err, "session_id", sess.ID)
		} else {
			chatCtx.SessionMessages = msgs
		}
	}

	return chatCtx, nil
}

//go:embed prompt/ticket_comment.md
var ticketCommentPromptTemplate string

// generateInitialTicketComment generates an LLM-based initial comment for a ticket
func (x *UseCases) generateInitialTicketComment(ctx context.Context, ticketData *ticket.Ticket, alerts alert.Alerts) (string, error) {
	commentPrompt, err := prompt.GenerateWithStruct(ctx, ticketCommentPromptTemplate, map[string]any{
		"ticket": ticketData,
		"alerts": alerts,
		"lang":   lang.From(ctx),
	})
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate comment prompt")
	}

	session, err := x.llmClient.NewSession(ctx)
	if err != nil {
		return "", goerr.Wrap(err, "failed to create LLM session")
	}

	response, err := session.Generate(ctx, []gollem.Input{gollem.Text(commentPrompt)})
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate comment")
	}

	if len(response.Texts) == 0 {
		return "", goerr.New("no comment generated by LLM")
	}

	return response.Texts[0], nil
}
