package slack

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/prompt"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/slack-go/slack"
)

func buildAlertBlocks(alert alert.Alert) []slack.Block {
	lines := []string{
		"*ID:* `" + alert.ID.String() + "`",
		"*Schema:* `" + alert.Schema + "`",
		"*Status:* " + alert.Status.Label(),
		"*Assignee:* " + func() string {
			if alert.Assignee == nil {
				return ":no_entry: unassigned"
			}
			return ":bust_in_silhouette: <@" + alert.Assignee.ID + ">"
		}(),
		"*Severity:* " + func() string {
			if alert.Finding == nil {
				return types.AlertSeverityUnknown.Label()
			}

			return alert.Finding.Severity.Label()
		}(),
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

	if alert.Conclusion != "" {
		blocks = append(blocks, slack.NewDividerBlock())
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", "*Conclusion:* "+alert.Conclusion.Label(), false, false),
			nil,
			nil,
		))

		if alert.Reason != "" {
			blocks = append(blocks, slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", alert.Reason, false, false),
				nil,
				nil,
			))
		}
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

	if len(alert.Attributes) > 0 {
		fields := make([]*slack.TextBlockObject, 0, len(alert.Attributes)*2)
		for _, attr := range alert.Attributes {
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
	if alert.Finding != nil {
		blocks = append(blocks,
			slack.NewDividerBlock(),
			slack.NewHeaderBlock(
				slack.NewTextBlockObject(slack.PlainTextType, "🤖 AI Analysis Result", false, false),
			),
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", "Severity ➡️ *"+alert.Finding.Severity.String()+"*", false, false),
				nil,
				nil,
			),
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", "📝 *Summary:*\n"+alert.Finding.Summary, false, false),
				nil,
				nil,
			),
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", "🔍 *Reason:*\n"+alert.Finding.Reason, false, false),
				nil,
				nil,
			),
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", "💡 *Recommendation:*\n"+alert.Finding.Recommendation, false, false),
				nil,
				nil,
			),
		)
	}

	// Add action buttons
	buttons := []slack.BlockElement{}
	if alert.Finding == nil {
		buttons = append(buttons,
			slack.NewButtonBlockElement(
				model.SlackActionIDInspect.String(),
				alert.ID.String(),
				slack.NewTextBlockObject("plain_text", "Inspect", false, false),
			).WithStyle(slack.StyleDefault),
		)
	}

	if alert.Status == types.AlertStatusNew {
		buttons = append(buttons,
			slack.NewButtonBlockElement(
				model.SlackActionIDAck.String(),
				alert.ID.String(),
				slack.NewTextBlockObject("plain_text", "Acknowledge", false, false),
			).WithStyle(slack.StylePrimary),
		)
	}

	if alert.Status != types.AlertStatusResolved {
		buttons = append(buttons,
			slack.NewButtonBlockElement(
				model.SlackActionIDResolve.String(),
				alert.ID.String(),
				slack.NewTextBlockObject("plain_text", "Resolve", false, false),
			).WithStyle(slack.StyleDanger),
		)
	}

	if len(buttons) > 0 {
		blocks = append(blocks, slack.NewActionBlock("alert_actions", buttons...))
	}

	return blocks
}

func buildIgnoreModalViewRequest(listID string) slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type: slack.VTModal,
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Generate Ignore Policy",
		},
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				slack.NewSectionBlock(
					slack.NewTextBlockObject(slack.PlainTextType, "Please input prompt for generating ignore policy.", false, false),
					nil,
					nil,
				),
				slack.NewInputBlock(
					model.SlackBlockIDIgnorePrompt.String(),
					slack.NewTextBlockObject(slack.PlainTextType, "Prompt", false, false),
					slack.NewTextBlockObject(slack.PlainTextType, "Add any reason, context, or information.", false, false),
					slack.NewPlainTextInputBlockElement(
						slack.NewTextBlockObject(slack.PlainTextType, "prompt", false, false),
						model.SlackActionIDIgnorePrompt.String(),
					),
				).WithOptional(true),
			},
		},
		CallbackID:      model.CallbackSubmitIgnoreList.String(),
		PrivateMetadata: listID,
		Submit: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Ignore",
		},
		Close: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Cancel",
		},
	}
}

func buildResolveModalViewRequest(callbackID model.CallbackID, metadata string) slack.ModalViewRequest {
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
			Text: "Resolve Alert",
		},
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				slack.NewSectionBlock(
					slack.NewTextBlockObject(slack.PlainTextType, "Please input the conclusion and comment.", false, false),
					nil,
					nil,
				),
				slack.NewInputBlock(
					model.SlackBlockIDConclusion.String(),
					slack.NewTextBlockObject(slack.PlainTextType, "Conclusion", false, false),
					slack.NewTextBlockObject(slack.PlainTextType, "Select the conclusion", false, false),
					slack.NewOptionsSelectBlockElement(
						slack.OptTypeStatic,
						slack.NewTextBlockObject(slack.PlainTextType, "Select a conclusion", false, false),
						model.SlackActionIDConclusion.String(),
						conclusionOptionBlocks...,
					),
				),
				slack.NewInputBlock(
					model.SlackBlockIDComment.String(),
					slack.NewTextBlockObject(slack.PlainTextType, "Comment", false, false),
					slack.NewTextBlockObject(slack.PlainTextType, "Add any reason, context, or information.", false, false),
					slack.NewPlainTextInputBlockElement(
						slack.NewTextBlockObject(slack.PlainTextType, "comment", false, false),
						model.SlackActionIDComment.String(),
					),
				).WithOptional(true),
			},
		},
		CallbackID:      callbackID.String(),
		PrivateMetadata: metadata,
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
			model.SlackActionIDIgnoreList.String(),
			list.ID.String(),
			slack.NewTextBlockObject("plain_text", "Ignore", false, false),
		).WithStyle(slack.StyleDefault),
		slack.NewButtonBlockElement(
			model.SlackActionIDResolveList.String(),
			list.ID.String(),
			slack.NewTextBlockObject("plain_text", "Resolve", false, false),
		).WithStyle(slack.StyleDanger),
	))
	blocks = append(blocks, slack.NewDividerBlock())

	return blocks
}

func buildAlertsBlocks(alerts []alert.Alert, metadata slackMetadata) []slack.Block {
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

	statusCount := make(map[types.AlertStatus]int)
	for _, alert := range alerts {
		statusCount[alert.Status]++
	}

	for _, alert := range alerts {
		assigneeText := ""
		if alert.Assignee != nil {
			assigneeText = fmt.Sprintf(" (👤 <@%s>)", alert.Assignee.ID)
		}

		msgURL := metadata.ToMsgURL(alert.SlackThread.ChannelID, alert.SlackThread.ThreadID)
		newString := fmt.Sprintf("%s <%s|%s>%s\n", alert.Status.Label(), msgURL, alert.Title, assigneeText)
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

	var lines []string
	for status, count := range statusCount {
		if count == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("*%s*: %d", status.Label(), count))
	}

	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", strings.Join(lines, " / "), false, false),
		nil,
		nil,
	))

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

// buildNextActionBlocks builds the blocks for the next action message in the thread.
func buildNextActionBlocks(action prompt.ActionPromptResult) []slack.Block {
	var fields []*slack.TextBlockObject
	for key, arg := range action.Args {
		fields = append(fields, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*%s:* `%s`", key, arg), false, false))
	}

	nextMsg := fmt.Sprintf("⚡ Action: *%s*\n", action.Action)
	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, nextMsg, false, false),
			fields,
			nil,
		),
	}

	return blocks
}
