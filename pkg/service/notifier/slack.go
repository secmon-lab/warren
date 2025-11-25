package notifier

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/secmon-lab/warren/pkg/domain/event"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/notice"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

const slackMessageLimit = 3000

// SlackNotifier is a Slack-based event notifier that buffers
// alert pipeline events and posts them to a Slack thread when the alert is published.
// Provides real-time visibility into alert processing for team collaboration.
type SlackNotifier struct {
	// Buffer for storing event handlers until alert is published
	buffer []func(ctx context.Context, thread interfaces.SlackThreadService) error
}

// NewSlackNotifier creates a new Slack notifier that buffers events and flushes to thread on publish
func NewSlackNotifier() interfaces.Notifier {
	return &SlackNotifier{
		buffer: make([]func(ctx context.Context, thread interfaces.SlackThreadService) error, 0),
	}
}

func (n *SlackNotifier) NotifyIngestPolicyResult(ctx context.Context, ev *event.IngestPolicyResultEvent) {
	n.buffer = append(n.buffer, func(ctx context.Context, thread interfaces.SlackThreadService) error {
		return handleIngestPolicyResult(ctx, thread, ev)
	})
}

func (n *SlackNotifier) NotifyEnrichPolicyResult(ctx context.Context, ev *event.EnrichPolicyResultEvent) {
	n.buffer = append(n.buffer, func(ctx context.Context, thread interfaces.SlackThreadService) error {
		return handleEnrichPolicyResult(ctx, thread, ev)
	})
}

func (n *SlackNotifier) NotifyTriagePolicyResult(ctx context.Context, ev *event.TriagePolicyResultEvent) {
	n.buffer = append(n.buffer, func(ctx context.Context, thread interfaces.SlackThreadService) error {
		return handleTriagePolicyResult(ctx, thread, ev)
	})
}

func (n *SlackNotifier) NotifyEnrichTaskPrompt(ctx context.Context, ev *event.EnrichTaskPromptEvent) {
	n.buffer = append(n.buffer, func(ctx context.Context, thread interfaces.SlackThreadService) error {
		return handleEnrichTaskPrompt(ctx, thread, ev)
	})
}

func (n *SlackNotifier) NotifyEnrichTaskResponse(ctx context.Context, ev *event.EnrichTaskResponseEvent) {
	n.buffer = append(n.buffer, func(ctx context.Context, thread interfaces.SlackThreadService) error {
		return handleEnrichTaskResponse(ctx, thread, ev)
	})
}

func (n *SlackNotifier) NotifyError(ctx context.Context, ev *event.ErrorEvent) {
	n.buffer = append(n.buffer, func(ctx context.Context, thread interfaces.SlackThreadService) error {
		return handleError(ctx, thread, ev)
	})
}

func handleIngestPolicyResult(ctx context.Context, thread interfaces.SlackThreadService, e *event.IngestPolicyResultEvent) error {
	logger := logging.From(ctx)

	// Post summary message with better formatting as context block
	var summary string
	if len(e.Alerts) == 0 {
		summary = fmt.Sprintf(":inbox_tray: *Ingest Policy Result*\nSchema: `%s`\n_No alerts generated_", e.Schema)
	} else if len(e.Alerts) == 1 {
		summary = fmt.Sprintf(":inbox_tray: *Ingest Policy Result*\nSchema: `%s`\nGenerated: *1 alert*", e.Schema)
	} else {
		summary = fmt.Sprintf(":inbox_tray: *Ingest Policy Result*\nSchema: `%s`\nGenerated: *%d alerts*", e.Schema, len(e.Alerts))
	}

	if err := thread.PostContextBlock(ctx, summary); err != nil {
		logger.Warn("failed to post ingest policy result to Slack", "error", err, "schema", e.Schema)
		return err
	}

	return nil
}

func handleEnrichPolicyResult(ctx context.Context, thread interfaces.SlackThreadService, e *event.EnrichPolicyResultEvent) error {
	logger := logging.From(ctx)

	message := formatEnrichPolicyResult(e)
	if err := thread.PostContextBlock(ctx, message); err != nil {
		logger.Warn("failed to post enrich policy result to Slack", "error", err, "task_count", e.TaskCount)
		return err
	}
	return nil
}

func handleTriagePolicyResult(ctx context.Context, thread interfaces.SlackThreadService, e *event.TriagePolicyResultEvent) error {
	logger := logging.From(ctx)

	message := formatTriagePolicyResult(e)
	if err := thread.PostContextBlock(ctx, message); err != nil {
		logger.Warn("failed to post triage policy result to Slack", "error", err, "publish", e.Result.Publish)
		return err
	}
	return nil
}

func handleEnrichTaskPrompt(ctx context.Context, thread interfaces.SlackThreadService, e *event.EnrichTaskPromptEvent) error {
	logger := logging.From(ctx)
	summary := fmt.Sprintf("*Task Prompt* `%s`\nLength: %d chars", e.TaskID, len(e.PromptText))

	if len(e.PromptText) > slackMessageLimit {
		if err := thread.PostContextBlock(ctx, summary); err != nil {
			logger.Warn("failed to post enrich task prompt summary to Slack", "error", err, "task_id", e.TaskID)
			return err
		}
		if err := thread.AttachFile(ctx, fmt.Sprintf("Prompt [%s]", e.TaskID), fmt.Sprintf("prompt_%s.txt", e.TaskID), []byte(e.PromptText)); err != nil {
			logger.Warn("failed to attach enrich task prompt file to Slack", "error", err, "task_id", e.TaskID)
			return err
		}
	} else {
		message := summary + fmt.Sprintf("\n```\n%s\n```", e.PromptText)
		if err := thread.PostContextBlock(ctx, message); err != nil {
			logger.Warn("failed to post enrich task prompt to Slack", "error", err, "task_id", e.TaskID)
			return err
		}
	}
	return nil
}

func handleEnrichTaskResponse(ctx context.Context, thread interfaces.SlackThreadService, e *event.EnrichTaskResponseEvent) error {
	logger := logging.From(ctx)
	summary := fmt.Sprintf("*Task Response* `%s`", e.TaskID)

	var content string
	var fileName string

	switch v := e.Response.(type) {
	case string:
		content = v
		fileName = fmt.Sprintf("response_%s.txt", e.TaskID)

	case map[string]any, []any:
		jsonBytes, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			content = fmt.Sprintf("%v", v)
			fileName = fmt.Sprintf("response_%s.txt", e.TaskID)
		} else {
			content = string(jsonBytes)
			fileName = fmt.Sprintf("response_%s.json", e.TaskID)
		}

	default:
		content = fmt.Sprintf("%v", v)
		fileName = fmt.Sprintf("response_%s.txt", e.TaskID)
	}

	if len(content) > slackMessageLimit {
		if err := thread.PostContextBlock(ctx, summary); err != nil {
			logger.Warn("failed to post enrich task response summary to Slack", "error", err, "task_id", e.TaskID)
			return err
		}
		if err := thread.AttachFile(ctx, fmt.Sprintf("Response [%s]", e.TaskID), fileName, []byte(content)); err != nil {
			logger.Warn("failed to attach enrich task response file to Slack", "error", err, "task_id", e.TaskID)
			return err
		}
	} else {
		message := summary + fmt.Sprintf("\n```\n%s\n```", content)
		if err := thread.PostContextBlock(ctx, message); err != nil {
			logger.Warn("failed to post enrich task response to Slack", "error", err, "task_id", e.TaskID)
			return err
		}
	}
	return nil
}

func handleError(ctx context.Context, thread interfaces.SlackThreadService, e *event.ErrorEvent) error {
	logger := logging.From(ctx)

	message := formatError(e)
	if err := thread.PostContextBlock(ctx, message); err != nil {
		logger.Warn("failed to post error event to Slack", "error", err, "task_id", e.TaskID, "original_error", e.Error)
		return err
	}
	return nil
}

func formatEnrichPolicyResult(e *event.EnrichPolicyResultEvent) string {
	msg := ":mag: *Enrich Policy Result*\n"
	msg += fmt.Sprintf("Total Tasks: *%d*\n", e.TaskCount)

	if e.Policy == nil || len(e.Policy.Prompts) == 0 {
		msg += "_No enrichment tasks defined_\n"
		return msg
	}

	msg += fmt.Sprintf("\n*Prompt Tasks* (%d):\n", len(e.Policy.Prompts))
	for i, task := range e.Policy.Prompts {
		msg += fmt.Sprintf("  %d. `%s` [%s]", i+1, task.ID, task.Format)
		if task.Template != "" {
			msg += fmt.Sprintf(" • Template: `%s`", task.Template)
			if len(task.Params) > 0 {
				msg += fmt.Sprintf(" (with %d params)", len(task.Params))
			}
		} else if task.Inline != "" {
			inlinePreview := task.Inline
			if len(inlinePreview) > 50 {
				inlinePreview = inlinePreview[:50] + "..."
			}
			msg += fmt.Sprintf(" • Inline: %s", inlinePreview)
		}
		msg += "\n"
	}

	return msg
}

func formatTriagePolicyResult(e *event.TriagePolicyResultEvent) string {
	msg := ":white_check_mark: *Triage Policy Result*\n"

	// Show publish decision with appropriate emoji
	var publishIcon string
	switch e.Result.Publish {
	case "alert":
		publishIcon = ":rotating_light:"
	case "notice":
		publishIcon = ":bell:"
	case "discard":
		publishIcon = ":wastebasket:"
	default:
		publishIcon = ":question:"
	}
	msg += fmt.Sprintf("%s Publish: *%s*\n", publishIcon, e.Result.Publish)

	// Show metadata if set
	hasMetadata := false
	if e.Result.Title != "" {
		msg += fmt.Sprintf("\n*Title:*\n%s\n", e.Result.Title)
		hasMetadata = true
	}
	if e.Result.Description != "" {
		desc := e.Result.Description
		if len(desc) > 200 {
			desc = desc[:200] + "..."
		}
		msg += fmt.Sprintf("\n*Description:*\n%s\n", desc)
		hasMetadata = true
	}
	if e.Result.Channel != "" {
		msg += fmt.Sprintf("\n*Channel:* `%s`\n", e.Result.Channel)
		hasMetadata = true
	}
	if len(e.Result.Attr) > 0 {
		msg += fmt.Sprintf("\n*Attributes* (%d):\n", len(e.Result.Attr))
		for i, attr := range e.Result.Attr {
			msg += fmt.Sprintf("  %d. *%s* = `%s`", i+1, attr.Key, attr.Value)
			if attr.Link != "" {
				msg += fmt.Sprintf(" [<%s|link>]", attr.Link)
			}
			msg += "\n"
		}
		hasMetadata = true
	}

	if !hasMetadata {
		msg += "_No metadata modifications_\n"
	}

	return msg
}

func formatError(e *event.ErrorEvent) string {
	var msg string
	if e.TaskID != "" {
		msg = fmt.Sprintf("*Error* `%s`\n", e.TaskID)
	} else {
		msg = "*Error*\n"
	}

	msg += fmt.Sprintf(":x: %s\n", e.Message)

	if e.Error != nil {
		msg += fmt.Sprintf("```\n%v\n```", e.Error)
	}

	return msg
}

// SlackServicePoster is an interface for posting alerts to Slack
type SlackServicePoster interface {
	PostAlert(ctx context.Context, alert *alert.Alert) (interfaces.SlackThreadService, error)
}

// PublishAlert posts the alert to Slack and flushes all buffered pipeline events to the thread
// This method combines alert posting with pipeline event flushing to avoid duplication
func (n *SlackNotifier) PublishAlert(ctx context.Context, slackService SlackServicePoster, alertToPublish *alert.Alert) (interfaces.SlackThreadService, error) {
	logger := logging.From(ctx)

	// Post alert to Slack (creates new thread)
	thread, err := slackService.PostAlert(ctx, alertToPublish)
	if err != nil {
		return nil, err
	}

	logger.Debug("flushing buffered pipeline events to Slack thread",
		"event_count", len(n.buffer),
		"alert_id", alertToPublish.ID)

	// Flush all buffered pipeline events to the thread
	for i, handler := range n.buffer {
		if err := handler(ctx, thread); err != nil {
			logger.Warn("failed to flush event to Slack thread",
				"error", err,
				"event_index", i,
				"total_events", len(n.buffer),
				"alert_id", alertToPublish.ID)
			// Continue flushing other events even if one fails
		}
	}

	// Clear buffer after flushing
	n.buffer = make([]func(ctx context.Context, thread interfaces.SlackThreadService) error, 0)

	return thread, nil
}

// SlackServiceNotice is an interface for posting notices to Slack
type SlackServiceNotice interface {
	PostNotice(ctx context.Context, channelID, message string, noticeID fmt.Stringer) (string, error)
	PostNoticeThreadDetails(ctx context.Context, channelID, threadTS string, alert *alert.Alert, llmResponse *alert.GenAIResponse) error
	NewThread(thread slack.Thread) interfaces.SlackThreadService
}

// PublishNotice posts the notice to Slack and flushes all buffered pipeline events to the thread
// This method combines notice posting with pipeline event flushing to avoid duplication
func (n *SlackNotifier) PublishNotice(ctx context.Context, slackService SlackServiceNotice, notice *notice.Notice, channel string, llmResponse *alert.GenAIResponse) (string, error) {
	logger := logging.From(ctx)

	alertData := &notice.Alert

	// Create simple message with only title for main channel
	var mainMessage string
	if alertData.Title != "" {
		mainMessage = "🔔 " + alertData.Title
	} else {
		mainMessage = "🔔 Security Notice"
	}

	// Post main notice message
	timestamp, err := slackService.PostNotice(ctx, channel, mainMessage, notice.ID)
	if err != nil {
		return "", err
	}

	// Post detailed information in thread
	if err := slackService.PostNoticeThreadDetails(ctx, channel, timestamp, alertData, llmResponse); err != nil {
		logger.Warn("failed to post notice thread details", "error", err, "channel", channel)
	}

	// Create thread service
	thread := slackService.NewThread(slack.Thread{
		ChannelID: channel,
		ThreadID:  timestamp,
	})

	logger.Debug("flushing buffered pipeline events to notice thread",
		"event_count", len(n.buffer),
		"notice_id", notice.ID)

	// Flush all buffered pipeline events to the thread
	for i, handler := range n.buffer {
		if err := handler(ctx, thread); err != nil {
			logger.Warn("failed to flush event to Slack thread",
				"error", err,
				"event_index", i,
				"total_events", len(n.buffer),
				"notice_id", notice.ID)
			// Continue flushing other events even if one fails
		}
	}

	// Clear buffer after flushing
	n.buffer = make([]func(ctx context.Context, thread interfaces.SlackThreadService) error, 0)

	return timestamp, nil
}
