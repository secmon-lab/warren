// Package chatnotifier provides the Session-scoped message delivery
// abstractions introduced by the chat-session-redesign spec. See
// pkg/domain/interfaces/session_notifier.go for the interface contract.
//
// Implementations are constructed per request via Factory and passed into
// ChatUseCase.Execute through context (see ctx.go). There is no global
// registry; the Factory is the only stateful piece and it holds only
// shared service dependencies (repository, Slack service, etc.), never
// per-Session state.
//
// This subpackage of pkg/service/notifier exists so that the new
// SessionNotifier implementations (SlackChatNotifier / WebNotifier /
// CLINotifier / NoopNotifier) do not collide in identifier space with the
// pre-existing alert pipeline Notifier implementations in the parent
// package (e.g. SlackNotifier, ConsoleNotifier for pipeline events).
package chatnotifier

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/session"
)

// NoopNotifier is a SessionNotifier that does nothing. It satisfies the
// interface for cases where no Session has been wired up yet (ticketless
// bootstrap, tests, etc.).
type NoopNotifier struct{}

func (NoopNotifier) Notify(context.Context, string) error                      { return nil }
func (NoopNotifier) Trace(context.Context, string) error                       { return nil }
func (NoopNotifier) Warn(context.Context, string) error                        { return nil }
func (NoopNotifier) Plan(context.Context, string) error                        { return nil }
func (NoopNotifier) NotifyUser(context.Context, string, *session.Author) error { return nil }
