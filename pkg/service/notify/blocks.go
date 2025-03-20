package notify

import (
	"strings"

	"github.com/slack-go/slack"
)

func newContextBlock(messages []string) slack.Block {
	return slack.NewContextBlock(
		"",
		slack.NewTextBlockObject(slack.MarkdownType, strings.Join(messages, "\n"), false, false),
	)
}
