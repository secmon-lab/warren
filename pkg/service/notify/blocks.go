package notify

import (
	"strings"

	"github.com/slack-go/slack"
)

func newContextBlocks(base string, messages []string) []slack.Block {
	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, base, false, false),
			nil,
			nil,
		),
	}

	if len(messages) > 0 {
		blocks = append(blocks, slack.NewContextBlock(
			"context_messages",
			slack.NewTextBlockObject(slack.MarkdownType, strings.Join(messages, "\n"), false, false),
		))
	}

	return blocks
}
