package usecase

import (
	"context"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/prompt"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/utils/thread"
)

const maxAlertsToGroup = 128

func (x *UseCases) GroupUnclosedAlerts(ctx context.Context, th interfaces.SlackThreadService) error {
	thread.Reply(ctx, "👥 Starting to make groups of alerts...")

	newAlerts, err := x.repository.GetAlertsByStatus(ctx, model.AlertStatusNew)
	if err != nil {
		return err
	}
	ackedAlerts, err := x.repository.GetAlertsByStatus(ctx, model.AlertStatusAcknowledged)
	if err != nil {
		return err
	}

	alerts := append(newAlerts, ackedAlerts...)

	if len(alerts) == 0 {
		thread.Reply(ctx, "⏸️ No alerts to be grouped")
		return nil
	}

	thread.Reply(ctx, fmt.Sprintf("🏃 Found %d alerts to be grouped", len(alerts)))
	if len(alerts) > maxAlertsToGroup {
		thread.Reply(ctx, fmt.Sprintf("⚠️ Too many alerts to be grouped. Let's make groups of first %d alerts.", maxAlertsToGroup))
		alerts = alerts[:maxAlertsToGroup]
	}

	p, err := prompt.BuildMakeGroupPrompt(ctx, alerts, 16)
	if err != nil {
		return err
	}

	chat := x.geminiStartChat()
	var groups []model.AlertGroupMetadata

	const maxRetry = 3
	for i := range maxRetry {
		thread.Reply(ctx, fmt.Sprintf("🏃 Making groups of alerts... (%d/%d)", i+1, maxRetry))
		resp, err := service.AskChat[prompt.MakeGroupPromptResult](ctx, chat, p)
		if err != nil {
			if goerr.HasTag(err, model.ErrTagInvalidLLMResponse) {
				thread.Reply(ctx, "💥 Failed to make groups of alerts. Retry...")
				p = "invalid response, please try again: " + err.Error()
				continue
			}
			return err
		}

		groups = resp.Groups
		break
	}

	if len(groups) == 0 {
		thread.Reply(ctx, "💥 Failed to make groups of alerts")
		return nil
	}

	thread.Reply(ctx, fmt.Sprintf("✅ Successfully made groups of alerts (%d groups)", len(groups)))

	alertGroups := make([]model.AlertGroup, len(groups))
	for i, group := range groups {
		alertGroups[i] = model.NewAlertGroup(ctx, group)

		// pick 3 alerts from alerts
		var alertIDSet []model.AlertID
		if len(group.AlertIDs) > 3 {
			alertIDSet = group.AlertIDs[:3]
		} else {
			alertIDSet = group.AlertIDs
		}

		alerts, err := x.repository.BatchGetAlerts(ctx, alertIDSet)
		if err != nil {
			return err
		}
		alertGroups[i].Alerts = alerts
	}

	if err := x.repository.PutAlertGroups(ctx, alertGroups); err != nil {
		return err
	}

	if err := th.PostAlertGroups(ctx, alertGroups); err != nil {
		return err
	}

	return nil
}
