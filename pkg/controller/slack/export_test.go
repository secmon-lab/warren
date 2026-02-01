package slack

import "github.com/slack-go/slack/slackevents"

// ResolveMessageEventUserID exports resolveMessageEventUserID for testing.
func ResolveMessageEventUserID(event *slackevents.MessageEvent) string {
	return resolveMessageEventUserID(event)
}
