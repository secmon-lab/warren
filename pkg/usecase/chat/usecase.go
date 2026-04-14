package chat

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	slackService "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/request_id"
	ssnutil "github.com/secmon-lab/warren/pkg/utils/session"
	"github.com/secmon-lab/warren/pkg/utils/slackctx"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

// Strategy defines the interface for chat execution strategies.
// Implementations receive a RunContext with a pre-initialized Warren session
// (session created, message routing configured, authorization verified).
// The strategy is responsible for its own gollem LLM session management
// and execution workflow (planning, task execution, replanning, etc.).
type Strategy interface {
	Execute(ctx context.Context, rc *RunContext) error
}

// RunContext holds all the pre-initialized resources that UseCase
// passes to a Strategy after completing common setup.
type RunContext struct {
	Session *session.Session
	Message string
	ChatCtx *chatModel.ChatContext
}

// UseCase implements interfaces.ChatUseCase by performing common setup
// (Warren session management, message routing, authorization) and delegating
// strategy-specific execution to a Strategy implementation.
type UseCase struct {
	strategy        Strategy
	repository      interfaces.Repository
	policyClient    interfaces.PolicyClient
	slackService    *slackService.Service
	noAuthorization bool
	frontendURL     string
}

// Option configures a UseCase.
type Option func(*UseCase)

// WithRepository sets the repository.
func WithRepository(repo interfaces.Repository) Option {
	return func(u *UseCase) { u.repository = repo }
}

// WithPolicyClient sets the policy client for authorization.
func WithPolicyClient(pc interfaces.PolicyClient) Option {
	return func(u *UseCase) { u.policyClient = pc }
}

// WithSlackService sets the Slack service for message routing.
func WithSlackService(svc *slackService.Service) Option {
	return func(u *UseCase) { u.slackService = svc }
}

// WithNoAuthorization disables policy-based authorization checks.
func WithNoAuthorization(noAuthz bool) Option {
	return func(u *UseCase) { u.noAuthorization = noAuthz }
}

// WithFrontendURL sets the frontend URL for session links.
func WithFrontendURL(url string) Option {
	return func(u *UseCase) { u.frontendURL = url }
}

// NewUseCase creates a new UseCase with the given strategy and options.
func NewUseCase(strategy Strategy, opts ...Option) *UseCase {
	u := &UseCase{
		strategy: strategy,
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

// Execute processes a chat message: creates a Warren session, sets up message
// routing, checks authorization, and delegates to the Strategy.
func (u *UseCase) Execute(ctx context.Context, message string, chatCtx chatModel.ChatContext) error {
	target := chatCtx.Ticket
	logger := logging.From(ctx)
	logger.Debug("chat usecase execute: start",
		"ticket_id", target.ID,
		"request_id", request_id.FromContext(ctx),
	)

	// Phase 1: Warren session setup
	ssn, ctx := u.createSession(ctx, target, message)
	logger = logging.From(ctx)
	logger.Debug("chat usecase execute: session created", "session_id", ssn.ID)

	// Phase 2: Message routing setup
	ctx = u.setupMessageRouting(ctx, ssn, target)
	logger.Debug("chat usecase execute: message routing set up")

	// Phase 3: Session status tracking
	ctx = u.setupStatusCheck(ctx, ssn)

	finalStatus := types.SessionStatusCompleted
	defer u.finishSession(ctx, ssn, target, &finalStatus)

	// Phase 4: Authorization
	authorized, err := u.authorize(ctx, message)
	if err != nil {
		finalStatus = types.SessionStatusAborted
		return err
	}
	if !authorized {
		finalStatus = types.SessionStatusAborted
		logger.Debug("chat usecase execute: not authorized, returning")
		return nil
	}
	logger.Debug("chat usecase execute: authorized, delegating to strategy")

	// Phase 5: Delegate to strategy
	rc := &RunContext{
		Session: ssn,
		Message: message,
		ChatCtx: &chatCtx,
	}
	if err := u.strategy.Execute(ctx, rc); err != nil {
		if errors.Is(err, ErrSessionAborted) || errors.Is(err, context.Canceled) {
			finalStatus = types.SessionStatusAborted
			return nil
		}
		finalStatus = types.SessionStatusAborted
		return err
	}

	return nil
}

// createSession creates and persists a new Warren chat session.
func (u *UseCase) createSession(ctx context.Context, target *ticket.Ticket, message string) (*session.Session, context.Context) {
	userID := types.UserID(user.FromContext(ctx))
	slackURL := slackctx.SlackURL(ctx)

	ssn := session.NewSession(ctx, target.ID, userID, message, slackURL)
	if err := u.repository.PutSession(ctx, ssn); err != nil {
		logging.From(ctx).Error("failed to save session", "error", err)
	}

	logger := logging.From(ctx).With("session_id", ssn.ID, "request_id", request_id.FromContext(ctx))
	ctx = logging.With(ctx, logger)

	if target.SlackThread != nil {
		ctx = slackctx.WithThread(ctx, *target.SlackThread)
	}

	return ssn, ctx
}

// setupMessageRouting configures Slack/CLI message routing functions in the context.
func (u *UseCase) setupMessageRouting(ctx context.Context, ssn *session.Session, target *ticket.Ticket) context.Context {
	if u.slackService != nil && target.SlackThread != nil {
		notifyFunc, traceFunc, warnFunc := u.setupSlackMessageFuncs(ctx, ssn, target)
		ctx = msg.With(ctx, notifyFunc, traceFunc, warnFunc)

		// Post a brief status indicator as a context block immediately
		verbs := []string{
			"Investigating", "Analyzing", "Processing", "Inspecting",
			"Examining", "Scanning", "Assessing", "Evaluating",
			"Reviewing", "Probing", "Surveying", "Diagnosing",
			"Exploring", "Scrutinizing", "Correlating", "Parsing",
			"Decoding", "Interpreting", "Triaging", "Resolving",
		}
		verb := verbs[rand.IntN(len(verbs))] // #nosec G404 -- not security-sensitive, just picking a random UI verb
		threadSvc := u.slackService.NewThread(*target.SlackThread)
		if err := threadSvc.PostContextBlock(ctx, fmt.Sprintf("%s ...", verb)); err != nil {
			logging.From(ctx).Error("failed to post status", "error", err)
		}
	}

	return ctx
}

// setupSlackMessageFuncs creates Slack message routing functions for notify, trace, and warn.
func (u *UseCase) setupSlackMessageFuncs(_ context.Context, sess *session.Session, target *ticket.Ticket) (msg.NotifyFunc, msg.TraceFunc, msg.WarnFunc) {
	threadSvc := u.slackService.NewThread(*target.SlackThread)

	notifyFunc := func(ctx context.Context, message string) {
		m := session.NewMessage(ctx, sess.ID, session.MessageTypeResponse, message)
		if err := u.repository.PutSessionMessage(ctx, m); err != nil {
			errutil.Handle(ctx, err)
		}
		if err := threadSvc.PostComment(ctx, message); err != nil {
			errutil.Handle(ctx, err)
		}
	}

	var traceUpdateFunc func(context.Context, string)
	traceFunc := func(ctx context.Context, message string) {
		m := session.NewMessage(ctx, sess.ID, session.MessageTypeTrace, message)
		if err := u.repository.PutSessionMessage(ctx, m); err != nil {
			errutil.Handle(ctx, err)
		}

		if traceUpdateFunc == nil {
			traceUpdateFunc = threadSvc.NewUpdatableMessage(ctx, message)
		} else {
			traceUpdateFunc(ctx, message)
		}
	}

	warnFunc := func(ctx context.Context, message string) {
		m := session.NewMessage(ctx, sess.ID, session.MessageTypeWarning, message)
		if err := u.repository.PutSessionMessage(ctx, m); err != nil {
			errutil.Handle(ctx, err)
		}
		if err := threadSvc.PostComment(ctx, message); err != nil {
			errutil.Handle(ctx, err)
		}
	}

	return notifyFunc, traceFunc, warnFunc
}

// setupStatusCheck embeds a session abort check function in the context.
func (u *UseCase) setupStatusCheck(ctx context.Context, ssn *session.Session) context.Context {
	statusCheckFunc := func(ctx context.Context) error {
		s, err := u.repository.GetSession(ctx, ssn.ID)
		if err != nil {
			return goerr.Wrap(err, "failed to get session status")
		}
		if s != nil && s.Status == types.SessionStatusAborted {
			return ErrSessionAborted
		}
		return nil
	}
	return ssnutil.WithStatusCheck(ctx, statusCheckFunc)
}

// authorize checks policy-based authorization for agent execution.
func (u *UseCase) authorize(ctx context.Context, message string) (bool, error) {
	if err := AuthorizeAgentRequest(ctx, u.policyClient, u.noAuthorization, message); err != nil {
		if errors.Is(err, ErrAgentAuthPolicyNotDefined) {
			msg.Notify(ctx, "🚫 *Authorization Failed*\n\nAgent execution policy is not defined. Please configure the `auth.agent` policy or use `--no-authorization` flag for development.\n\nSee: https://docs.warren.secmon-lab.com/policy.md#agent-execution-authorization")
		} else if errors.Is(err, ErrAgentAuthDenied) {
			msg.Notify(ctx, "🚫 *Authorization Failed*\n\nYou are not authorized to execute agent requests. Please contact your administrator if you believe this is an error.")
		} else {
			msg.Notify(ctx, "🚫 *Authorization Failed*\n\nFailed to check authorization. Please contact your administrator.")
			return false, goerr.Wrap(err, "failed to evaluate agent auth")
		}
		return false, nil
	}
	return true, nil
}

// finishSession updates session status and posts session actions on completion.
func (u *UseCase) finishSession(ctx context.Context, ssn *session.Session, target *ticket.Ticket, finalStatus *types.SessionStatus) {
	logger := logging.From(ctx)
	if r := recover(); r != nil {
		*finalStatus = types.SessionStatusAborted
		ssn.UpdateStatus(ctx, *finalStatus)
		if err := u.repository.PutSession(ctx, ssn); err != nil {
			logger.Error("failed to update session status on panic", "error", err, "status", *finalStatus)
		}
		panic(r)
	}

	ssn.UpdateStatus(ctx, *finalStatus)
	if err := u.repository.PutSession(ctx, ssn); err != nil {
		logger.Error("failed to update final session status", "error", err, "status", *finalStatus)
	}

	// Skip session actions for ticketless chat (no ticket to act on)
	if target.ID != "" && *finalStatus == types.SessionStatusCompleted && u.slackService != nil && target.SlackThread != nil {
		threadSvc := u.slackService.NewThread(*target.SlackThread)

		var sessionURL string
		if u.frontendURL != "" {
			sessionURL = fmt.Sprintf("%s/sessions/%s", u.frontendURL, ssn.ID)
		}

		currentTicket, err := u.repository.GetTicket(ctx, target.ID)
		if err != nil {
			logger.Error("failed to get ticket for session actions", "error", err, "ticket_id", target.ID)
		} else if currentTicket != nil {
			if err := threadSvc.PostSessionActions(ctx, target.ID, currentTicket.Status, sessionURL); err != nil {
				logger.Error("failed to post session actions to Slack", "error", err, "session_id", ssn.ID)
			}
		}
	}
}
