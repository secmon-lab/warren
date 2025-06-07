package mrkdwn

import (
	"regexp"
	"unicode/utf8"
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
			// If no pattern matches, consume a single UTF-8 character (rune) as text
			r, width := utf8.DecodeRuneInString(input[pos:])
			if r == utf8.RuneError && width == 1 {
				// Invalid UTF-8, skip this byte
				pos++
				continue
			}
			tokens = append(tokens, token{
				Type:  tokenText,
				Value: string(r),
				Start: pos,
				End:   pos + width,
			})
			pos += width
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
