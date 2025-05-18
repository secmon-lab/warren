package slack

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/slack-go/slack"
)

func buildAlertBlocks(alert alert.Alert) []slack.Block {
	ack := "⏳"
	if alert.TicketID != types.EmptyTicketID {
		ack = "✅"
	}

	lines := []string{
		"*ID:* `" + alert.ID.String() + "`",
		"*Schema:* `" + alert.Schema.String() + "`",
		"*Ack:* " + ack,
	}

	title := "❗ " + alert.Title
	titleBytes := []byte(title)
	if len(titleBytes) > 140 {
		// Find the position to cut that doesn't break UTF-8 characters
		pos := 0
		count := 0
		for pos < len(titleBytes) && count < 137 { // 137 to leave room for "..."
			_, size := utf8.DecodeRune(titleBytes[pos:])
			pos += size
			count += size
		}
		title = string(titleBytes[:pos]) + "..."
	}

	description := "_no description_"
	if alert.Description != "" {
		description = alert.Description
	}

	blocks := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject(slack.PlainTextType, title, false, false),
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, description, false, false),
			nil,
			nil,
		),
	}

	blocks = append(blocks, []slack.Block{
		slack.NewDividerBlock(),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", strings.Join(lines, "\n"), false, false),
			nil,
			nil,
		),
		slack.NewDividerBlock(),
	}...)

	if len(alert.Metadata.Attributes) > 0 {
		fields := make([]*slack.TextBlockObject, 0, len(alert.Metadata.Attributes)*2)
		for _, attr := range alert.Metadata.Attributes {
			var value string
			if attr.Link != "" {
				value = "<" + attr.Link + "|" + attr.Value + ">"
			} else {
				value = "`" + attr.Value + "`"
			}
			fields = append(fields,
				slack.NewTextBlockObject("mrkdwn", "*"+attr.Key+":*\n"+value, false, false),
			)
		}
		blocks = append(blocks, slack.NewSectionBlock(nil, fields, nil))
	}
	// Add action buttons
	buttons := []slack.BlockElement{}

	if alert.TicketID == types.EmptyTicketID {
		buttons = append(buttons,
			slack.NewButtonBlockElement(
				model.ActionIDAckAlert.String(),
				alert.ID.String(),
				slack.NewTextBlockObject("plain_text", "Acknowledge", false, false),
			).WithStyle(slack.StylePrimary),
		)

		buttons = append(buttons,
			slack.NewButtonBlockElement(
				model.ActionIDBindAlert.String(),
				alert.ID.String(),
				slack.NewTextBlockObject("plain_text", "Bind to ticket", false, false),
			).WithStyle(slack.StyleDanger),
		)
	}

	if len(buttons) > 0 {
		blocks = append(blocks, slack.NewActionBlock("alert_actions", buttons...))
	}

	return blocks
}

func buildTicketBlocks(ticket ticket.Ticket, alerts alert.Alerts, metadata slackMetadata) []slack.Block {
	var blocks []slack.Block

	// Header with Title and emoji
	blocks = append(blocks, slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, fmt.Sprintf("🎫 %s", ticket.Metadata.Title), false, false),
	))

	// ID and Description
	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*ID:* `%s`\n%s", ticket.ID.String(), ticket.Metadata.Description), false, false),
		nil,
		nil,
	))

	// Status, Assignee, Conclusion fields with emojis
	fields := []*slack.TextBlockObject{
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Status:*\n%s", ticket.Status.Label()), false, false),
	}
	if ticket.Assignee != nil {
		fields = append(fields, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Assignee:*\n👤 <@%s>", ticket.Assignee.ID), false, false))
	}
	if ticket.Conclusion != "" {
		fields = append(fields, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Conclusion:*\n%s", ticket.Conclusion.Label()), false, false))
	}
	if ticket.Reason != "" {
		fields = append(fields, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Reason:*\n💭 %s", ticket.Reason), false, false))
	}
	if len(fields) > 0 {
		blocks = append(blocks, slack.NewSectionBlock(nil, fields, nil))
	}

	// Alert list section
	if len(alerts) > 0 {
		blocks = append(blocks, slack.NewDividerBlock())
		blocks = append(blocks, slack.NewHeaderBlock(
			slack.NewTextBlockObject(slack.PlainTextType, "🔔 Related Alerts", false, false),
		))

		var alertList string
		displayCount := min(len(alerts), 5)

		for i := range displayCount {
			alert := alerts[i]
			alertList += fmt.Sprintf("• <%s|%s>\n",
				metadata.ToMsgURL(alert.SlackThread.ChannelID, alert.SlackThread.ThreadID),
				alert.Metadata.Title)
		}
		if displayCount < len(alerts) {
			alertList += fmt.Sprintf("\n_Showing %d of %d alerts_", displayCount, len(alerts))
		}

		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, alertList, false, false),
			nil,
			nil,
		))
	}

	// Finding section if exists
	if ticket.Finding != nil {
		blocks = append(blocks,
			slack.NewDividerBlock(),
			slack.NewHeaderBlock(
				slack.NewTextBlockObject(slack.PlainTextType, "🔍 Finding", false, false),
			),
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Severity:* %s\n*Summary:* %s",
					ticket.Finding.Severity.Label(),
					ticket.Finding.Summary), false, false),
				nil,
				nil,
			),
		)
		if ticket.Finding.Reason != "" {
			blocks = append(blocks, slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Reason:*\n💭 %s", ticket.Finding.Reason), false, false),
				nil,
				nil,
			))
		}
		if ticket.Finding.Recommendation != "" {
			blocks = append(blocks, slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Recommendation:*\n💡 %s", ticket.Finding.Recommendation), false, false),
				nil,
				nil,
			))
		}
	}

	return blocks
}

func buildBindToTicketModalViewRequest(ctx context.Context, callbackID model.CallbackID, tickets []*ticket.Ticket, metadata string) slack.ModalViewRequest {
	// Create ticket options for dropdown
	ticketOptions := make([]*slack.OptionBlockObject, 0, len(tickets))
	now := clock.Now(ctx)
	for _, t := range tickets {
		elapsed := now.Sub(t.CreatedAt)
		var timeStr string
		switch {
		case elapsed < time.Minute:
			timeStr = "just now"
		case elapsed < time.Hour:
			minutes := int(elapsed.Minutes())
			timeStr = fmt.Sprintf("%dm ago", minutes)
		case elapsed < 24*time.Hour:
			hours := int(elapsed.Hours())
			timeStr = fmt.Sprintf("%dh ago", hours)
		case elapsed < 30*24*time.Hour:
			days := int(elapsed.Hours() / 24)
			timeStr = fmt.Sprintf("%dd ago", days)
		default:
			timeStr = t.CreatedAt.Format("2006-01-02")
		}

		label := fmt.Sprintf("%s (%s)", t.Metadata.Title, timeStr)
		ticketOptions = append(ticketOptions,
			slack.NewOptionBlockObject(
				t.ID.String(),
				slack.NewTextBlockObject(slack.PlainTextType, label, false, false),
				slack.NewTextBlockObject(slack.PlainTextType, t.Metadata.Description, false, false),
			),
		)
	}

	msg := "Please enter a ticket ID."
	if len(ticketOptions) > 0 {
		msg = "Please select an existing ticket or enter a ticket ID."
	}

	blockSet := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.PlainTextType, msg, false, false),
			nil,
			nil,
		),
	}

	if len(ticketOptions) > 0 {
		blockSet = append(blockSet, slack.NewInputBlock(
			model.BlockIDTicketSelect.String(),
			slack.NewTextBlockObject(slack.PlainTextType, "Select Ticket", false, false),
			slack.NewTextBlockObject(slack.PlainTextType, "Choose a ticket from the list", false, false),
			slack.NewOptionsSelectBlockElement(
				slack.OptTypeStatic,
				slack.NewTextBlockObject(slack.PlainTextType, "Select a ticket", false, false),
				model.BlockActionIDTicketSelect.String(),
				ticketOptions...,
			)).WithOptional(true),
		)
	}

	blockSet = append(blockSet, slack.NewInputBlock(
		model.BlockIDTicketID.String(),
		slack.NewTextBlockObject(slack.PlainTextType, "Or Enter Ticket ID", false, false),
		slack.NewTextBlockObject(slack.PlainTextType, "Enter the ticket ID directly", false, false),
		slack.NewPlainTextInputBlockElement(
			slack.NewTextBlockObject(slack.PlainTextType, "Enter ticket ID", false, false),
			model.BlockActionIDTicketID.String(),
		)).WithOptional(true),
	)

	return slack.ModalViewRequest{
		Type: slack.VTModal,
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Bind Alert to Ticket",
		},
		Blocks: slack.Blocks{
			BlockSet: blockSet,
		},
		CallbackID:      callbackID.String(),
		PrivateMetadata: metadata,
		Submit: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Bind",
		},
		Close: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Cancel",
		},
	}
}

func buildResolveTicketModalViewRequest(callbackID model.CallbackID, ticket *ticket.Ticket) slack.ModalViewRequest {
	conclusionOptions := []struct {
		Conclusion  types.AlertConclusion
		Label       string
		Description string
	}{
		{
			Conclusion:  types.AlertConclusionUnaffected,
			Label:       types.AlertConclusionUnaffected.Label(),
			Description: "The alert indicates actual attack or vulnerability, but it is no impact.",
		},
		{
			Conclusion:  types.AlertConclusionIntended,
			Label:       types.AlertConclusionIntended.Label(),
			Description: "The alert is intended behavior or configuration.",
		},
		{
			Conclusion:  types.AlertConclusionFalsePositive,
			Label:       types.AlertConclusionFalsePositive.Label(),
			Description: "The alert is not attack or impact on the system.",
		},
		{
			Conclusion:  types.AlertConclusionTruePositive,
			Label:       types.AlertConclusionTruePositive.Label(),
			Description: "The alert has actual impact on the system.",
		},
	}

	conclusionOptionBlocks := make([]*slack.OptionBlockObject, 0, len(conclusionOptions))
	for _, option := range conclusionOptions {
		conclusionOptionBlocks = append(conclusionOptionBlocks,
			slack.NewOptionBlockObject(
				option.Conclusion.String(),
				slack.NewTextBlockObject(slack.PlainTextType, option.Label, false, false),
				slack.NewTextBlockObject(slack.PlainTextType, option.Description, false, false),
			),
		)
	}

	return slack.ModalViewRequest{
		Type: slack.VTModal,
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Resolve Ticket",
		},
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				slack.NewSectionBlock(
					slack.NewTextBlockObject(slack.PlainTextType, "Please input the conclusion and comment.", false, false),
					nil,
					nil,
				),
				slack.NewInputBlock(
					model.BlockIDTicketConclusion.String(),
					slack.NewTextBlockObject(slack.PlainTextType, "Conclusion", false, false),
					slack.NewTextBlockObject(slack.PlainTextType, "Select the conclusion", false, false),
					slack.NewOptionsSelectBlockElement(
						slack.OptTypeStatic,
						slack.NewTextBlockObject(slack.PlainTextType, "Select a conclusion", false, false),
						model.BlockActionIDTicketConclusion.String(),
						conclusionOptionBlocks...,
					),
				),
				slack.NewInputBlock(
					model.BlockIDTicketComment.String(),
					slack.NewTextBlockObject(slack.PlainTextType, "Comment", false, false),
					slack.NewTextBlockObject(slack.PlainTextType, "Add any reason, context, or information.", false, false),
					slack.NewPlainTextInputBlockElement(
						slack.NewTextBlockObject(slack.PlainTextType, "comment", false, false),
						model.BlockActionIDTicketComment.String(),
					),
				).WithOptional(true),
			},
		},
		CallbackID:      callbackID.String(),
		PrivateMetadata: ticket.ID.String(),
		Submit: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Resolve",
		},
		Close: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Cancel",
		},
	}
}

func buildAlertListBlocks(list *alert.List, metadata slackMetadata) []slack.Block {
	var blocks []slack.Block

	if list.Title != "" {
		blocks = append(blocks, slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", list.Title, false, false),
		))
	}

	if list.Description != "" {
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", list.Description, false, false),
			nil,
			nil,
		))
	}

	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*ID*: `%s`", list.ID.String()), false, false),
		nil,
		nil,
	))
	blocks = append(blocks, buildAlertsBlocks(list.Alerts, metadata)...)
	blocks = append(blocks, slack.NewActionBlock(
		list.ID.String(),
		slack.NewButtonBlockElement(
			model.ActionIDAckList.String(),
			list.ID.String(),
			slack.NewTextBlockObject("plain_text", "Acknowledge", false, false),
		).WithStyle(slack.StyleDefault),
		slack.NewButtonBlockElement(
			model.ActionIDBindList.String(),
			list.ID.String(),
			slack.NewTextBlockObject("plain_text", "Bind to ticket", false, false),
		).WithStyle(slack.StyleDanger),
	))
	blocks = append(blocks, slack.NewDividerBlock())

	return blocks
}

func buildAlertsBlocks(alerts alert.Alerts, metadata slackMetadata) []slack.Block {
	if len(alerts) == 0 {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", "🈳 No alerts found", false, false),
				nil,
				nil,
			),
		}
	}

	var messageText strings.Builder

	maxCharCount := 3000
	msgCount := 0

	for _, alert := range alerts {
		msgURL := metadata.ToMsgURL(alert.SlackThread.ChannelID, alert.SlackThread.ThreadID)
		newString := fmt.Sprintf("<%s|%s>\n", msgURL, alert.Title)
		if messageText.Len()+len(newString) > maxCharCount {
			break
		}
		messageText.WriteString(newString)
		msgCount++
	}

	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", messageText.String(), false, false),
			nil,
			nil,
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("_Showing %d of %d alerts_", msgCount, len(alerts)), false, false),
			nil,
			nil,
		),
	}

	return blocks
}

func buildAlertClustersBlocks(clusters []alert.List, metadata slackMetadata) []slack.Block {
	blocks := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", "🗂️ Alert Clusters", false, false),
		),
		slack.NewDividerBlock(),
	}

	for _, cluster := range clusters {
		blocks = append(blocks, buildAlertListBlocks(&cluster, metadata)...)
	}

	return blocks
}

// buildStateMessageBlocks builds the blocks for the state message in the thread.
func buildStateMessageBlocks(base string, messages []string) []slack.Block {
	blocks := []slack.Block{}

	if base != "" {
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, base, false, false),
			nil,
			nil,
		))
	}

	if len(messages) > 0 {
		blocks = append(blocks, slack.NewContextBlock(
			"context_messages",
			slack.NewTextBlockObject(slack.MarkdownType, strings.Join(messages, "\n"), false, false),
		))
	}

	if len(blocks) == 0 {
		return nil
	}

	return blocks
}
