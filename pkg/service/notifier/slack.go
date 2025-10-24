package notifier

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/secmon-lab/warren/pkg/domain/event"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

// ThreadPoster is an interface for posting messages to a Slack thread
type ThreadPoster interface {
	PostMessage(ctx context.Context, text string) error
	AttachFile(ctx context.Context, title, fileName string, data []byte) error
}

const slackMessageLimit = 3000

// SlackNotifier is a Slack-based event notifier that posts
// alert pipeline events to a Slack thread with formatted messages.
// Provides real-time visibility into alert processing for team collaboration.
type SlackNotifier struct {
	thread ThreadPoster
}

// NewSlackNotifier creates a new Slack notifier that posts events to the specified thread
func NewSlackNotifier(thread ThreadPoster) interfaces.Notifier {
	return &SlackNotifier{
		thread: thread,
	}
}

func (n *SlackNotifier) NotifyAlertPolicyResult(ctx context.Context, ev *event.AlertPolicyResultEvent) {
	handleAlertPolicyResult(ctx, n.thread, ev)
}

func (n *SlackNotifier) NotifyEnrichPolicyResult(ctx context.Context, ev *event.EnrichPolicyResultEvent) {
	handleEnrichPolicyResult(ctx, n.thread, ev)
}

func (n *SlackNotifier) NotifyCommitPolicyResult(ctx context.Context, ev *event.CommitPolicyResultEvent) {
	handleCommitPolicyResult(ctx, n.thread, ev)
}

func (n *SlackNotifier) NotifyEnrichTaskPrompt(ctx context.Context, ev *event.EnrichTaskPromptEvent) {
	handleEnrichTaskPrompt(ctx, n.thread, ev)
}

func (n *SlackNotifier) NotifyEnrichTaskResponse(ctx context.Context, ev *event.EnrichTaskResponseEvent) {
	handleEnrichTaskResponse(ctx, n.thread, ev)
}

func (n *SlackNotifier) NotifyError(ctx context.Context, ev *event.ErrorEvent) {
	handleError(ctx, n.thread, ev)
}

func handleAlertPolicyResult(ctx context.Context, thread ThreadPoster, e *event.AlertPolicyResultEvent) {
	// Post summary message
	summary := fmt.Sprintf("*Alert Policy Result*\nSchema: `%s` | Alerts: %d", e.Schema, len(e.Alerts))
	_ = thread.PostMessage(ctx, summary)

	// Upload full alert details as JSON file
	if len(e.Alerts) > 0 {
		alertsJSON, err := json.MarshalIndent(e.Alerts, "", "  ")
		if err == nil {
			_ = thread.AttachFile(ctx, "Alert Details", "alerts.json", alertsJSON)
		}
	}
}

func handleEnrichPolicyResult(ctx context.Context, thread ThreadPoster, e *event.EnrichPolicyResultEvent) {
	message := formatEnrichPolicyResult(e)
	_ = thread.PostMessage(ctx, message)
}

func handleCommitPolicyResult(ctx context.Context, thread ThreadPoster, e *event.CommitPolicyResultEvent) {
	message := formatCommitPolicyResult(e)
	_ = thread.PostMessage(ctx, message)
}

func handleEnrichTaskPrompt(ctx context.Context, thread ThreadPoster, e *event.EnrichTaskPromptEvent) {
	summary := fmt.Sprintf("*Task Prompt* `%s` (%s)\nLength: %d chars", e.TaskID, e.TaskType, len(e.PromptText))

	if len(e.PromptText) > slackMessageLimit {
		_ = thread.PostMessage(ctx, summary)
		_ = thread.AttachFile(ctx, fmt.Sprintf("Prompt [%s]", e.TaskID), fmt.Sprintf("prompt_%s.txt", e.TaskID), []byte(e.PromptText))
	} else {
		message := summary + fmt.Sprintf("\n```\n%s\n```", e.PromptText)
		_ = thread.PostMessage(ctx, message)
	}
}

func handleEnrichTaskResponse(ctx context.Context, thread ThreadPoster, e *event.EnrichTaskResponseEvent) {
	summary := fmt.Sprintf("*Task Response* `%s` (%s)", e.TaskID, e.TaskType)

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
		_ = thread.PostMessage(ctx, summary)
		_ = thread.AttachFile(ctx, fmt.Sprintf("Response [%s]", e.TaskID), fileName, []byte(content))
	} else {
		message := summary + fmt.Sprintf("\n```\n%s\n```", content)
		_ = thread.PostMessage(ctx, message)
	}
}

func handleError(ctx context.Context, thread ThreadPoster, e *event.ErrorEvent) {
	message := formatError(e)
	_ = thread.PostMessage(ctx, message)
}

func formatEnrichPolicyResult(e *event.EnrichPolicyResultEvent) string {
	msg := "*Enrich Policy Result*\n"
	msg += fmt.Sprintf("Tasks: %d\n", e.TaskCount)

	if e.Policy != nil {
		if len(e.Policy.Query) > 0 {
			queryIDs := make([]string, 0, len(e.Policy.Query))
			for _, task := range e.Policy.Query {
				queryIDs = append(queryIDs, task.ID)
			}
			msg += "  • Query tasks: "
			for i, id := range queryIDs {
				if i > 0 {
					msg += ", "
				}
				msg += fmt.Sprintf("`%s`", id)
			}
			msg += "\n"
		}
		if len(e.Policy.Agent) > 0 {
			agentIDs := make([]string, 0, len(e.Policy.Agent))
			for _, task := range e.Policy.Agent {
				agentIDs = append(agentIDs, task.ID)
			}
			msg += "  • Agent tasks: "
			for i, id := range agentIDs {
				if i > 0 {
					msg += ", "
				}
				msg += fmt.Sprintf("`%s`", id)
			}
			msg += "\n"
		}
	}

	return msg
}

func formatCommitPolicyResult(e *event.CommitPolicyResultEvent) string {
	msg := "*Commit Policy Result*\n"
	msg += fmt.Sprintf("Publish: `%s`\n", e.Result.Publish)

	if e.Result.Title != "" {
		msg += fmt.Sprintf("  • Title: %s\n", e.Result.Title)
	}
	if e.Result.Description != "" {
		desc := e.Result.Description
		if len(desc) > 100 {
			desc = desc[:100] + "..."
		}
		msg += fmt.Sprintf("  • Description: %s\n", desc)
	}
	if e.Result.Channel != "" {
		msg += fmt.Sprintf("  • Channel: `%s`\n", e.Result.Channel)
	}
	if len(e.Result.Attr) > 0 {
		msg += fmt.Sprintf("  • Attributes: %d\n", len(e.Result.Attr))
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
