package chat

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// MultiNotifier sends notifications to multiple chat services
type MultiNotifier struct {
	notifiers []interfaces.ChatNotifier
}

// NewMultiNotifier creates a new MultiNotifier with the given notifiers
func NewMultiNotifier(notifiers ...interfaces.ChatNotifier) *MultiNotifier {
	return &MultiNotifier{
		notifiers: notifiers,
	}
}

// NotifyMessage sends a message to all configured notifiers
func (m *MultiNotifier) NotifyMessage(ctx context.Context, ticketID types.TicketID, message string) error {
	var errs []error

	for _, notifier := range m.notifiers {
		if err := notifier.NotifyMessage(ctx, ticketID, message); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return goerr.Wrap(errs[0], "failed to send message via one or more notifiers", goerr.V("total_errors", len(errs)))
	}

	return nil
}

// NotifyTrace sends a trace message to all configured notifiers
func (m *MultiNotifier) NotifyTrace(ctx context.Context, ticketID types.TicketID, message string) error {
	var errs []error

	for _, notifier := range m.notifiers {
		if err := notifier.NotifyTrace(ctx, ticketID, message); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return goerr.Wrap(errs[0], "failed to send trace message via one or more notifiers", goerr.V("total_errors", len(errs)))
	}

	return nil
}
