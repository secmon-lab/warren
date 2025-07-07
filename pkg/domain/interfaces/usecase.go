package interfaces

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type SlackEventUsecases interface {
	// Slack event handlers
	HandleSlackMessage(ctx context.Context, slackMsg slack.Message) error
	HandleSlackAppMention(ctx context.Context, slackMsg slack.Message) error
}

type SlackInteractionUsecases interface {
	// Slack interaction handlers
	HandleSlackInteractionViewSubmission(ctx context.Context, user slack.User, callbackID slack.CallbackID, metadata string, values slack.StateValue) error
	HandleSlackInteractionBlockActions(ctx context.Context, user slack.User, slackThread slack.Thread, actionID slack.ActionID, value, triggerID string) error
	HandleSalvageRefresh(ctx context.Context, user slack.User, metadata string, values slack.StateValue, viewID string) error
}

type AlertUsecases interface {
	// Alert related handlers
	HandleAlert(ctx context.Context, schema types.AlertSchema, alertData any) ([]*alert.Alert, error)
}

type ApiUsecases interface {
	// User related handlers
	GetUserIcon(ctx context.Context, userID string) ([]byte, string, error)
	GetUserProfile(ctx context.Context, userID string) (string, error)

	// Ticket related handlers
	GenerateTicketAlertsJSONL(ctx context.Context, ticketID types.TicketID) ([]byte, error)
}
