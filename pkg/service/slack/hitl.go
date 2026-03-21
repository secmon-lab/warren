package slack

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/slack-go/slack"
)

// UpdatableBlockMessage manages a Slack message that can be updated with arbitrary blocks.
// It shares the message ID across text-based and block-based updates.
type UpdatableBlockMessage struct {
	threadSvc *ThreadService
	msgID     string
	mu        sync.Mutex
}

// NewUpdatableBlockMessage creates a new updatable block message that posts an initial text message
// and can later be updated with either text or block content.
func (x *ThreadService) NewUpdatableBlockMessage(ctx context.Context, initialMessage string) *UpdatableBlockMessage {
	blocks := buildStateMessageBlocks([]string{initialMessage})
	var msgID string
	if len(blocks) > 0 {
		msgID = x.postInitialMessage(ctx, blocks)
	}
	return &UpdatableBlockMessage{
		threadSvc: x,
		msgID:     msgID,
	}
}

// UpdateText updates the message with plain text content (same as existing NewUpdatableMessage).
func (m *UpdatableBlockMessage) UpdateText(ctx context.Context, text string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	blocks := buildStateMessageBlocks([]string{text})
	if len(blocks) == 0 {
		return
	}

	if m.msgID == "" {
		m.msgID = m.threadSvc.postInitialMessage(ctx, blocks)
		return
	}

	m.threadSvc.updateMessage(ctx, m.msgID, blocks)
}

// UpdateBlocks updates the message with arbitrary Slack blocks (for buttons, inputs, etc.).
func (m *UpdatableBlockMessage) UpdateBlocks(ctx context.Context, blocks []slack.Block) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(blocks) == 0 {
		return
	}

	if m.msgID == "" {
		m.msgID = m.threadSvc.postInitialMessage(ctx, blocks)
		return
	}

	m.threadSvc.updateMessage(ctx, m.msgID, blocks)
}

// BuildToolApprovalBlocks constructs Slack blocks for a HITL tool approval request.
// Format follows the existing task message pattern: {emoji} *[{task}]*\n\n> {message}
func BuildToolApprovalBlocks(taskTitle, userID string, req *hitl.Request) []slack.Block {
	p := req.ToolApproval()
	toolName := p.ToolName
	toolArgs := p.ToolArgs

	// Follow task message pattern with "Approval Required" inserted between emoji and title
	headerText := fmt.Sprintf("⏳ Approval Required *[%s]*\n\n> 🔐 <@%s> `%s`",
		escapeSlackMrkdwn(taskTitle), userID, toolName)

	// Build argument display
	if len(toolArgs) > 0 {
		var argLines []string
		for k, v := range toolArgs {
			argLines = append(argLines, fmt.Sprintf("*%s:* %v", k, v))
		}
		headerText += "\n" + strings.Join(argLines, "\n")
	}

	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, headerText, false, false),
			nil, nil,
		),
		slack.NewInputBlock(
			model.BlockIDHITLComment.String(),
			slack.NewTextBlockObject(slack.PlainTextType, "Comment (optional)", false, false),
			nil,
			slack.NewPlainTextInputBlockElement(
				slack.NewTextBlockObject(slack.PlainTextType, "Enter comment...", false, false),
				model.BlockActionIDHITLComment.String(),
			),
		),
		slack.NewActionBlock(
			"hitl_actions",
			slack.NewButtonBlockElement(
				model.ActionIDHITLApprove.String(),
				req.ID.String(),
				slack.NewTextBlockObject(slack.PlainTextType, "Allow ✅", false, false),
			).WithStyle(slack.StylePrimary),
			slack.NewButtonBlockElement(
				model.ActionIDHITLDeny.String(),
				req.ID.String(),
				slack.NewTextBlockObject(slack.PlainTextType, "Deny ❌", false, false),
			).WithStyle(slack.StyleDanger),
		),
	}

	// Make the comment input optional
	blocks[1].(*slack.InputBlock).Optional = true

	return blocks
}

func escapeSlackMrkdwn(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// HITLPresenter implements hitl.Presenter for Slack transport.
// It updates the task progress message with approval buttons.
type HITLPresenter struct {
	msg       *UpdatableBlockMessage
	taskTitle string
	userID    string
}

// NewHITLPresenter creates a new Slack HITL presenter.
func NewHITLPresenter(msg *UpdatableBlockMessage, taskTitle, userID string) *HITLPresenter {
	return &HITLPresenter{
		msg:       msg,
		taskTitle: taskTitle,
		userID:    userID,
	}
}

// Present updates the task progress message with approval buttons.
func (p *HITLPresenter) Present(ctx context.Context, req *hitl.Request) error {
	blocks := BuildToolApprovalBlocks(p.taskTitle, p.userID, req)
	p.msg.UpdateBlocks(ctx, blocks)

	logging.From(ctx).Info("HITL approval request presented",
		"request_id", req.ID,
		"task_title", p.taskTitle,
		"user_id", p.userID,
	)

	return nil
}
