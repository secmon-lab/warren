package slack

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
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

func buildTicketBlocks(ticket ticket.Ticket, alerts alert.Alerts, metadata slackMetadata, frontendURL string) []slack.Block {
	var blocks []slack.Block

	// Header with Title and emoji - add TEST indicator if it's a test ticket
	var headerTitle string
	if ticket.IsTest {
		headerTitle = fmt.Sprintf("🧪 [TEST] %s", ticket.Metadata.Title)
	} else {
		headerTitle = fmt.Sprintf("🎫 %s", ticket.Metadata.Title)
	}
	blocks = append(blocks, slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, headerTitle, false, false),
	))

	// ID, Description and Frontend Link - add TEST indicator
	var idDescText string
	testPrefix := ""
	if ticket.IsTest {
		testPrefix = "🧪 *[TEST TICKET]* "
	}

	if frontendURL != "" {
		ticketURL := fmt.Sprintf("%s/tickets/%s", frontendURL, ticket.ID.String())
		idDescText = fmt.Sprintf("%s*ID:* `%s` | <%s|🔗 View Details>\n%s", testPrefix, ticket.ID.String(), ticketURL, ticket.Metadata.Description)
	} else {
		idDescText = fmt.Sprintf("%s*ID:* `%s`\n%s", testPrefix, ticket.ID.String(), ticket.Metadata.Description)
	}

	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, idDescText, false, false),
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

	// Add Resolve and Salvage buttons if ticket is not resolved or archived
	if ticket.Status != types.TicketStatusResolved && ticket.Status != types.TicketStatusArchived {
		buttons := []slack.BlockElement{
			slack.NewButtonBlockElement(
				model.ActionIDResolveTicket.String(),
				ticket.ID.String(),
				slack.NewTextBlockObject("plain_text", "Resolve", false, false),
			).WithStyle(slack.StylePrimary),
			slack.NewButtonBlockElement(
				model.ActionIDSalvage.String(),
				ticket.ID.String(),
				slack.NewTextBlockObject("plain_text", "Salvage", false, false),
			).WithStyle(slack.StyleDefault),
		}
		blocks = append(blocks, slack.NewActionBlock("ticket_actions", buttons...))
	}

	return blocks
}

func shortenString(s string, maxLen int) string {
	if len([]rune(s)) <= maxLen {
		return s
	}
	return string([]rune(s)[:maxLen-3]) + "..."
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

		title := shortenString(t.Metadata.Title, 32)
		description := shortenString(t.Metadata.Description, 64)
		status := t.Status.Icon()

		label := fmt.Sprintf("%s %s (%s)", status, title, timeStr)
		ticketOptions = append(ticketOptions,
			slack.NewOptionBlockObject(
				t.ID.String(),
				slack.NewTextBlockObject(slack.PlainTextType, label, false, false),
				slack.NewTextBlockObject(slack.PlainTextType, description, false, false),
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
		slack.NewTextBlockObject(slack.PlainTextType, "Enter Ticket ID", false, false),
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

func buildResolveTicketModalViewRequest(callbackID model.CallbackID, ticket *ticket.Ticket, availableTags []*tag.Tag) slack.ModalViewRequest {
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
		{
			Conclusion:  types.AlertConclusionEscalated,
			Label:       types.AlertConclusionEscalated.Label(),
			Description: "The alert has been escalated to external management.",
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

	// Build tag options if available
	blockSet := []slack.Block{
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
				slack.NewTextBlockObject(slack.PlainTextType, "Enter comment", false, false),
				model.BlockActionIDTicketComment.String(),
			),
		).WithOptional(true),
	}

	// Add tag selection if tags are available
	if len(availableTags) > 0 {
		// Use checkboxes if 10 or fewer tags, otherwise use multi-select dropdown
		if len(availableTags) <= 10 {
			// Create checkbox options for tags
			checkboxOptions := make([]*slack.OptionBlockObject, 0, len(availableTags))
			for _, tag := range availableTags {
				checkboxOptions = append(checkboxOptions,
					slack.NewOptionBlockObject(
						tag.ID, // Use Tag ID as value
						slack.NewTextBlockObject(slack.PlainTextType, tag.Name, false, false), // Use Tag name for display
						nil,
					),
				)
			}

			// Add checkboxes for tags
			blockSet = append(blockSet, slack.NewInputBlock(
				model.BlockIDTicketTags.String(),
				slack.NewTextBlockObject(slack.PlainTextType, "Tags", false, false),
				slack.NewTextBlockObject(slack.PlainTextType, "Select tags for this ticket", false, false),
				slack.NewCheckboxGroupsBlockElement(
					model.BlockActionIDTicketTags.String(),
					checkboxOptions...,
				),
			).WithOptional(true))
		} else {
			// Use multi-select dropdown for more than 10 tags
			tagOptions := make([]*slack.OptionBlockObject, 0, len(availableTags))

			// Create tag options
			for _, tag := range availableTags {
				tagOptions = append(tagOptions,
					slack.NewOptionBlockObject(
						tag.ID, // Use Tag ID as value
						slack.NewTextBlockObject(slack.PlainTextType, tag.Name, false, false), // Use Tag name for display
						nil,
					),
				)
			}

			// Add multi-select for tags
			// Note: initial_options is not supported in modal views for multi_select elements
			blockSet = append(blockSet, slack.NewInputBlock(
				model.BlockIDTicketTags.String(),
				slack.NewTextBlockObject(slack.PlainTextType, "Tags", false, false),
				slack.NewTextBlockObject(slack.PlainTextType, "Select tags for this ticket", false, false),
				slack.NewOptionsMultiSelectBlockElement(
					slack.OptTypeStatic,
					slack.NewTextBlockObject(slack.PlainTextType, "Select tags", false, false),
					model.BlockActionIDTicketTags.String(),
					tagOptions...,
				),
			).WithOptional(true))
		}
	}

	return slack.ModalViewRequest{
		Type: slack.VTModal,
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Resolve Ticket",
		},
		Blocks: slack.Blocks{
			BlockSet: blockSet,
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

func buildSalvageModalViewRequest(callbackID model.CallbackID, ticket *ticket.Ticket, unboundAlerts alert.Alerts, threshold float64, keyword string) slack.ModalViewRequest {
	// Build the alert list section
	var alertListText string
	displayCount := min(len(unboundAlerts), 10)
	totalCount := len(unboundAlerts)

	if displayCount == 0 {
		alertListText = "No unbound alerts found"
	} else {
		for i := range displayCount {
			alert := unboundAlerts[i]
			alertListText += fmt.Sprintf("• %s (ID: %s)\n", alert.Title, alert.ID.String())
		}
		if displayCount < totalCount {
			alertListText += fmt.Sprintf("\nShowing %d of %d alerts", displayCount, totalCount)
		} else {
			alertListText += fmt.Sprintf("\nTotal: %d alerts", totalCount)
		}
	}

	// Set current threshold value
	thresholdValue := "0.9"
	if threshold > 0 {
		thresholdValue = fmt.Sprintf("%.2f", threshold)
	}

	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.PlainTextType, "Search and bind unbound alerts to this ticket", false, false),
			nil,
			nil,
		),
		slack.NewInputBlock(
			model.BlockIDSalvageThreshold.String(),
			slack.NewTextBlockObject(slack.PlainTextType, "Similarity Threshold", false, false),
			slack.NewTextBlockObject(slack.PlainTextType, "Filter alerts by embedding similarity (0-1)", false, false),
			slack.NewPlainTextInputBlockElement(
				slack.NewTextBlockObject(slack.PlainTextType, "0", false, false),
				model.BlockActionIDSalvageThreshold.String(),
			).WithInitialValue(thresholdValue),
		).WithOptional(true),
		slack.NewInputBlock(
			model.BlockIDSalvageKeyword.String(),
			slack.NewTextBlockObject(slack.PlainTextType, "Keyword Filter", false, false),
			slack.NewTextBlockObject(slack.PlainTextType, "Filter alerts by keyword in data", false, false),
			slack.NewPlainTextInputBlockElement(
				slack.NewTextBlockObject(slack.PlainTextType, "Enter keyword", false, false),
				model.BlockActionIDSalvageKeyword.String(),
			).WithInitialValue(keyword),
		).WithOptional(true),
		slack.NewActionBlock(
			"refresh_action",
			slack.NewButtonBlockElement(
				model.BlockActionIDSalvageRefresh.String(),
				"refresh",
				slack.NewTextBlockObject("plain_text", "Refresh", false, false),
			).WithStyle(slack.StyleDefault),
		),
		slack.NewDividerBlock(),
		slack.NewHeaderBlock(
			slack.NewTextBlockObject(slack.PlainTextType, "Matching Alerts", false, false),
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, alertListText, false, false),
			nil,
			nil,
		),
	}

	return slack.ModalViewRequest{
		Type: slack.VTModal,
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Salvage Alerts",
		},
		Blocks: slack.Blocks{
			BlockSet: blocks,
		},
		CallbackID:      callbackID.String(),
		PrivateMetadata: ticket.ID.String(),
		Submit: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Submit",
		},
		Close: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Cancel",
		},
	}
}

func buildAlertListBlocks(list *alert.List, alerts alert.Alerts, metadata slackMetadata) []slack.Block {
	var blocks []slack.Block

	// Header with status icon and title
	headerText := fmt.Sprintf("%s Alert List", list.Status.Icon())
	if list.Title != "" {
		headerText = fmt.Sprintf("%s %s", list.Status.Icon(), list.Title)
	}
	blocks = append(blocks, slack.NewHeaderBlock(
		slack.NewTextBlockObject("plain_text", headerText, false, false),
	))

	if list.Description != "" {
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", list.Description, false, false),
			nil,
			nil,
		))
	}

	// Status and ID information
	statusText := fmt.Sprintf("*Status:* %s %s\n*ID:* `%s`",
		list.Status.Icon(),
		list.Status.DisplayName(),
		list.ID.String())
	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", statusText, false, false),
		nil,
		nil,
	))

	blocks = append(blocks, buildAlertsBlocks(alerts, metadata)...)

	// Add action buttons only if status is unbound
	if list.Status == alert.ListStatusUnbound {
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
	}

	blocks = append(blocks, slack.NewDividerBlock())

	return blocks
}

func buildNewAlertListBlocks(list *alert.List, alerts alert.Alerts, metadata slackMetadata) []slack.Block {
	blocks := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", fmt.Sprintf("📑 New list with %d alerts", len(alerts)), false, false),
		),
		slack.NewDividerBlock(),
	}

	blocks = append(blocks, buildAlertListBlocks(list, alerts, metadata)...)

	return blocks
}

func buildCompletedAlertListBlocks(list *alert.List, alerts alert.Alerts, metadata slackMetadata, status string) []slack.Block {
	var blocks []slack.Block
	var statusIcon string
	var statusText string

	switch status {
	case "acknowledged":
		statusIcon = "✅"
		statusText = "Acknowledged"
	case "bound":
		statusIcon = "🔗"
		statusText = "Bound to ticket"
	default:
		statusIcon = "✨"
		statusText = "Completed"
	}

	blocks = append(blocks, slack.NewHeaderBlock(
		slack.NewTextBlockObject("plain_text", fmt.Sprintf("📑 Alert list with %d alerts", len(alerts)), false, false),
	))

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

	// Status section
	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*ID*: `%s`\n*Status*: %s %s", list.ID.String(), statusIcon, statusText), false, false),
		nil,
		nil,
	))

	blocks = append(blocks, buildAlertsBlocks(alerts, metadata)...)
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

func buildAlertClustersBlocks(clusters []*alert.List, metadata slackMetadata) ([]slack.Block, error) {
	blocks := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", "🗂️ Alert Clusters", false, false),
		),
		slack.NewDividerBlock(),
	}

	for _, cluster := range clusters {
		alerts, err := cluster.Alerts()
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get alerts")
		}
		blocks = append(blocks, buildAlertListBlocks(cluster, alerts, metadata)...)
	}

	return blocks, nil
}

// buildStateMessageBlocks builds the blocks for the state message in the thread.
func buildStateMessageBlocks(messages []string) []slack.Block {
	blocks := []slack.Block{
		slack.NewContextBlock(
			"context_messages",
			slack.NewTextBlockObject(slack.MarkdownType, strings.Join(messages, "\n"), false, false),
		),
	}

	return blocks
}

// buildTraceMessageBlocks builds context blocks for trace messages (status updates)
func buildTraceMessageBlocks(message string) []slack.Block {
	if message == "" {
		return []slack.Block{}
	}

	blocks := []slack.Block{
		slack.NewContextBlock(
			"trace_context",
			slack.NewTextBlockObject(slack.MarkdownType, message, false, false),
		),
	}

	return blocks
}

// buildAccumulatedTraceMessageBlocks builds a single context block for trace messages
// Since NewTraceMessage now handles byte limits by creating new messages,
// this function only needs to create a single context block
func buildAccumulatedTraceMessageBlocks(messages []string) []slack.Block {
	if len(messages) == 0 {
		return []slack.Block{}
	}

	combinedText := strings.Join(messages, "\n")
	return []slack.Block{
		slack.NewContextBlock(
			"trace_context",
			slack.NewTextBlockObject(slack.MarkdownType, combinedText, false, false),
		),
	}
}

func buildTicketListBlocks(ctx context.Context, tickets []*ticket.Ticket, metadata slackMetadata) []slack.Block {
	blocks := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", "🎫 Ticket List", false, false),
		),
		slack.NewDividerBlock(),
	}

	if len(tickets) == 0 {
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", "📭 No tickets found", false, false),
			nil,
			nil,
		))
		return blocks
	}

	var messageText strings.Builder
	now := clock.Now(ctx)
	for _, t := range tickets {
		// Create a link to the ticket with test indicator
		var ticketTitle string
		if t.IsTest {
			ticketTitle = fmt.Sprintf("🧪 [TEST] %s", t.Metadata.Title)
		} else {
			ticketTitle = t.Metadata.Title
		}
		ticketLink := fmt.Sprintf("<%s|%s>", metadata.ToMsgURL(t.SlackThread.ChannelID, t.SlackThread.ThreadID), ticketTitle)

		// Calculate relative time
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

		// Create a line with status, time and assignee information
		statusInfo := fmt.Sprintf("%s %s (%s)", t.Status.Icon(), ticketLink, timeStr)
		if t.Assignee != nil {
			statusInfo += fmt.Sprintf(" 👤 <@%s>", t.Assignee.ID)
		}

		messageText.WriteString(statusInfo + "\n")
	}

	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", messageText.String(), false, false),
		nil,
		nil,
	))

	return blocks
}
