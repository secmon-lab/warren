package slack

import (
	"testing"

	"github.com/m-mizutani/gt"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
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

func TestBuildResolveTicketModalViewRequest_WithTags(t *testing.T) {
	tk := &ticket.Ticket{
		ID: types.TicketID("ticket-1"),
		Metadata: ticket.Metadata{
			Title: "Test Ticket",
		},
	}
	availableTags := []*tag.Tag{
		{ID: "tag-1", Name: "double check"},
		{ID: "tag-2", Name: "debug"},
		{ID: "tag-3", Name: "refine"},
	}

	result := buildResolveTicketModalViewRequest(model.CallbackSubmitResolveTicket, tk, availableTags)

	// Find the tags input block
	var found bool
	for _, block := range result.Blocks.BlockSet {
		inputBlock, ok := block.(*slack_sdk.InputBlock)
		if !ok {
			continue
		}
		if inputBlock.BlockID != model.BlockIDTicketTags.String() {
			continue
		}
		found = true

		// Verify it uses multi-select, not checkboxes
		multiSelect, ok := inputBlock.Element.(*slack_sdk.MultiSelectBlockElement)
		gt.V(t, ok).Equal(true)
		gt.V(t, multiSelect.Type).Equal(string(slack_sdk.OptTypeStatic))
		gt.V(t, len(multiSelect.Options)).Equal(3)
		gt.V(t, multiSelect.Options[0].Value).Equal("tag-1")
		gt.V(t, multiSelect.Options[0].Text.Text).Equal("double check")
		gt.V(t, multiSelect.Options[1].Value).Equal("tag-2")
		gt.V(t, multiSelect.Options[2].Value).Equal("tag-3")
	}
	gt.V(t, found).Equal(true)
}

func TestBuildResolveTicketModalViewRequest_NoTags(t *testing.T) {
	tk := &ticket.Ticket{
		ID: types.TicketID("ticket-1"),
		Metadata: ticket.Metadata{
			Title: "Test Ticket",
		},
	}

	result := buildResolveTicketModalViewRequest(model.CallbackSubmitResolveTicket, tk, nil)

	// Verify no tags block exists
	for _, block := range result.Blocks.BlockSet {
		inputBlock, ok := block.(*slack_sdk.InputBlock)
		if !ok {
			continue
		}
		gt.V(t, inputBlock.BlockID).NotEqual(model.BlockIDTicketTags.String())
	}
}
