package http_test

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type useCaseInterface struct {
	AlertUsecases            interfaces.AlertUsecases
	SlackEventUsecases       interfaces.SlackEventUsecases
	SlackInteractionUsecases interfaces.SlackInteractionUsecases
	ApiUsecases              interfaces.ApiUsecases
}

// AlertUsecases methods
func (u *useCaseInterface) HandleAlert(ctx context.Context, schema types.AlertSchema, alertData any) ([]*alert.Alert, error) {
	if u.AlertUsecases != nil {
		return u.AlertUsecases.HandleAlert(ctx, schema, alertData)
	}
	return nil, nil
}

// SlackEventUsecases methods
func (u *useCaseInterface) HandleSlackAppMention(ctx context.Context, slackMsg slack.Message) error {
	if u.SlackEventUsecases != nil {
		return u.SlackEventUsecases.HandleSlackAppMention(ctx, slackMsg)
	}
	return nil
}

func (u *useCaseInterface) HandleSlackMessage(ctx context.Context, slackMsg slack.Message) error {
	if u.SlackEventUsecases != nil {
		return u.SlackEventUsecases.HandleSlackMessage(ctx, slackMsg)
	}
	return nil
}

// SlackInteractionUsecases methods
func (u *useCaseInterface) HandleSlackInteractionBlockActions(ctx context.Context, user slack.User, thread slack.Thread, actionID slack.ActionID, value string, triggerID string) error {
	if u.SlackInteractionUsecases != nil {
		return u.SlackInteractionUsecases.HandleSlackInteractionBlockActions(ctx, user, thread, actionID, value, triggerID)
	}
	return nil
}

func (u *useCaseInterface) HandleSlackInteractionViewSubmission(ctx context.Context, user slack.User, callbackID slack.CallbackID, metadata string, values slack.StateValue) error {
	if u.SlackInteractionUsecases != nil {
		return u.SlackInteractionUsecases.HandleSlackInteractionViewSubmission(ctx, user, callbackID, metadata, values)
	}
	return nil
}

func (u *useCaseInterface) HandleSalvageRefresh(ctx context.Context, user slack.User, metadata string, values slack.StateValue, viewID string) error {
	if u.SlackInteractionUsecases != nil {
		return u.SlackInteractionUsecases.HandleSalvageRefresh(ctx, user, metadata, values, viewID)
	}
	return nil
}

// ApiUsecases methods
func (u *useCaseInterface) GetUserIcon(ctx context.Context, userID string) ([]byte, string, error) {
	if u.ApiUsecases != nil {
		return u.ApiUsecases.GetUserIcon(ctx, userID)
	}
	// Mock implementation for testing
	return []byte("mock-icon-data"), "image/png", nil
}

func (u *useCaseInterface) GetUserProfile(ctx context.Context, userID string) (string, error) {
	if u.ApiUsecases != nil {
		return u.ApiUsecases.GetUserProfile(ctx, userID)
	}
	return "", nil
}

func (u *useCaseInterface) GenerateTicketAlertsJSONL(ctx context.Context, ticketID types.TicketID) ([]byte, error) {
	if u.ApiUsecases != nil {
		return u.ApiUsecases.GenerateTicketAlertsJSONL(ctx, ticketID)
	}
	// Mock implementation for testing - return only data without metadata
	return []byte(`{"test":"data","value":123}
{"another":"record","count":456}
`), nil
}
