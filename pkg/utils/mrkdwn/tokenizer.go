package mrkdwn

import (
	"regexp"
)

// tokenize breaks down the input mrkdwn string into tokens
func (c *Converter) tokenize(input string) []token {
	var tokens []token
	pos := 0

	for pos < len(input) {
		// Try to match various patterns
		tok := c.nextToken(input, pos)
		if tok != nil {
			tokens = append(tokens, *tok)
			pos = tok.End
		} else {
			// If no pattern matches, consume a single character as text
			tokens = append(tokens, token{
				Type:  tokenText,
				Value: string(input[pos]),
				Start: pos,
				End:   pos + 1,
			})
			pos++
		}
	}

	return tokens
}

// nextToken tries to match the next token starting at the given position
func (c *Converter) nextToken(input string, pos int) *token {
	remaining := input[pos:]

	// Slack-specific sequences (higher priority)
	patterns := []struct {
		regex     *regexp.Regexp
		tokenType tokenType
	}{
		// User mention: <@U123ABCDE> or <@U123ABCDE|username>
		{regexp.MustCompile(`^<@([A-Z0-9]+)(\|([^>]+))?>`), tokenUserMention},

		// Channel link: <#C123ABCDE> or <#C123ABCDE|channel-name>
		{regexp.MustCompile(`^<#([A-Z0-9]+)(\|([^>]+))?>`), tokenChannelLink},

		// User group mention: <!subteam^S123ABCDE> or <!subteam^S123ABCDE|@handle>
		{regexp.MustCompile(`^<!subteam\^([A-Z0-9]+)(\|([^>]+))?>`), tokenUserGroupMention},

		// Special mentions: <!here>, <!channel>, <!everyone>
		{regexp.MustCompile(`^<!(here|channel|everyone)>`), tokenSpecialMention},

		// Date format: <!date^timestamp^{token}^link|fallback>
		{regexp.MustCompile(`^<!date\^(\d+)\^([^>\^]+)(\^([^>]+))?(\|([^>]+))?>`), tokenDateFormat},

		// Link with display text: <url|text>
		{regexp.MustCompile(`^<(https?://[^>|]+)\|([^>]+)>`), tokenLink},

		// Link with display text (markdown style): [text](url)
		{regexp.MustCompile(`^\[([^\]]+)\]\((https?://[^)]+)\)`), tokenLink},

		// Emoji: :emoji_name:
		{regexp.MustCompile(`^:([a-zA-Z0-9_+-]+):`), tokenEmoji},

		// Code block: ```...```
		{regexp.MustCompile("^```([\\s\\S]*?)```"), tokenCodeBlock},

		// Inline code: `...`
		{regexp.MustCompile("^`([^`]+)`"), tokenInlineCode},

		// Bold: *...*
		{regexp.MustCompile(`^\*([^*]+)\*`), tokenBold},

		// Italic: _..._
		{regexp.MustCompile(`^_([^_]+)_`), tokenItalic},

		// Strikethrough: ~...~
		{regexp.MustCompile(`^~([^~]+)~`), tokenStrikethrough},

		// Blockquote: > at line start
		{regexp.MustCompile(`^>(.*)$`), tokenBlockquote},

		// Ordered list: 1. item
		{regexp.MustCompile(`^(\d+\.\s+.*?)(?:\n|$)`), tokenOrderedList},

		// Unordered list: * item or - item
		{regexp.MustCompile(`^([*-]\s+.*?)(?:\n|$)`), tokenUnorderedList},
	}

	for _, pattern := range patterns {
		if match := pattern.regex.FindStringSubmatch(remaining); match != nil {
			return &token{
				Type:  pattern.tokenType,
				Value: match[0],
				Start: pos,
				End:   pos + len(match[0]),
			}
		}
	}

	return nil
}
