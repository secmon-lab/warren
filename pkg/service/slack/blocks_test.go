package slack

import (
	"testing"

	"github.com/m-mizutani/gt"
	slack_sdk "github.com/slack-go/slack"
)

func TestBuildTraceMessageBlocks(t *testing.T) {
	// Test building trace message blocks (context blocks)
	message := "Test trace message"

	blocks := buildTraceMessageBlocks(message)

	gt.V(t, len(blocks)).Equal(1)

	// Verify it's a context block
	contextBlock, ok := blocks[0].(*slack_sdk.ContextBlock)
	gt.V(t, ok).Equal(true)
	gt.V(t, contextBlock != nil).Equal(true)

	// Verify the block ID
	gt.V(t, contextBlock.BlockID).Equal("trace_context")

	// Verify the elements
	gt.V(t, len(contextBlock.ContextElements.Elements)).Equal(1)

	// Verify the text content
	textElement, ok := contextBlock.ContextElements.Elements[0].(*slack_sdk.TextBlockObject)
	gt.V(t, ok).Equal(true)
	gt.V(t, textElement.Type).Equal(slack_sdk.MarkdownType)
	gt.V(t, textElement.Text).Equal(message)
}

func TestBuildTraceMessageBlocks_Empty(t *testing.T) {
	// Test with empty message - should return empty blocks
	blocks := buildTraceMessageBlocks("")
	gt.V(t, len(blocks)).Equal(0)
}

func TestContextBlockVsRegularMessage(t *testing.T) {
	// This test demonstrates the difference between context blocks (for traces)
	// and regular section blocks (for normal messages)

	message := "Test message"

	// Context block (what we use for trace messages)
	contextBlock := slack_sdk.NewContextBlock(
		"trace_context",
		slack_sdk.NewTextBlockObject(slack_sdk.MarkdownType, message, false, false),
	)

	// Section block (what we use for regular messages)
	sectionBlock := slack_sdk.NewSectionBlock(
		slack_sdk.NewTextBlockObject(slack_sdk.MarkdownType, message, false, false),
		nil,
		nil,
	)

	// Verify they are different types
	gt.V(t, contextBlock.BlockType()).Equal(slack_sdk.MBTContext)
	gt.V(t, sectionBlock.BlockType()).Equal(slack_sdk.MBTSection)

	// This difference is crucial - context blocks appear differently in Slack
	// (as smaller, grayed-out context information) vs section blocks
	// (as regular message content)
}
