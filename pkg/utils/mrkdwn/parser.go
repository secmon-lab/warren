package mrkdwn

import (
	"regexp"
	"strconv"
	"strings"
)

// parse converts tokens into an AST
func (c *Converter) parse(tokens []token) *documentNode {
	doc := &documentNode{Children: []node{}}

	i := 0
	for i < len(tokens) {
		node, consumed := c.parseNode(tokens, i)
		if node != nil {
			doc.Children = append(doc.Children, node)
		}
		i += consumed
	}

	return doc
}

// parseNode parses a single node from tokens starting at index i
func (c *Converter) parseNode(tokens []token, i int) (node, int) {
	if i >= len(tokens) {
		return nil, 0
	}

	token := tokens[i]

	switch token.Type {
	case tokenText:
		return &textNode{Content: token.Value}, 1

	case tokenBold:
		content := extractContent(token.Value, `^\*(.+)\*$`)
		return &boldNode{Children: []node{&textNode{Content: content}}}, 1

	case tokenItalic:
		content := extractContent(token.Value, `^_(.+)_$`)
		return &italicNode{Children: []node{&textNode{Content: content}}}, 1

	case tokenStrikethrough:
		content := extractContent(token.Value, `^~(.+)~$`)
		return &strikethroughNode{Children: []node{&textNode{Content: content}}}, 1

	case tokenInlineCode:
		content := extractContent(token.Value, "^`(.+)`$")
		return &inlineCodeNode{Content: content}, 1

	case tokenCodeBlock:
		content := extractContent(token.Value, "^```([\\s\\S]*)```$")
		return &codeBlockNode{Content: content}, 1

	case tokenBlockquote:
		content := extractContent(token.Value, `^>(.*)$`)
		return &blockquoteNode{Children: []node{&textNode{Content: strings.TrimSpace(content)}}}, 1

	case tokenOrderedList:
		content := extractContent(token.Value, `^\d+\.\s+(.*)$`)
		return &orderedListNode{Children: []node{&listItemNode{Children: []node{&textNode{Content: content}}}}}, 1

	case tokenUnorderedList:
		content := extractContent(token.Value, `^[*-]\s+(.*)$`)
		return &unorderedListNode{Children: []node{&listItemNode{Children: []node{&textNode{Content: content}}}}}, 1

	case tokenUserMention:
		return c.parseUserMention(token.Value), 1

	case tokenChannelLink:
		return c.parseChannelLink(token.Value), 1

	case tokenUserGroupMention:
		return c.parseUserGroupMention(token.Value), 1

	case tokenSpecialMention:
		return c.parseSpecialMention(token.Value), 1

	case tokenDateFormat:
		return c.parseDateFormat(token.Value), 1

	case tokenLink:
		return c.parseLink(token.Value), 1

	case tokenEmoji:
		return c.parseEmoji(token.Value), 1

	default:
		return &textNode{Content: token.Value}, 1
	}
}

// parseUserMention parses user mention tokens
func (c *Converter) parseUserMention(value string) node {
	re := regexp.MustCompile(`^<@([A-Z0-9]+)(\|([^>]+))?>$`)
	matches := re.FindStringSubmatch(value)
	if len(matches) >= 2 {
		userID := matches[1]
		fallbackText := ""
		if len(matches) >= 4 && matches[3] != "" {
			fallbackText = matches[3]
		}
		return &userMentionNode{UserID: userID, Text: fallbackText}
	}
	return &textNode{Content: value}
}

// parseChannelLink parses channel link tokens
func (c *Converter) parseChannelLink(value string) node {
	re := regexp.MustCompile(`^<#([A-Z0-9]+)(\|([^>]+))?>$`)
	matches := re.FindStringSubmatch(value)
	if len(matches) >= 2 {
		channelID := matches[1]
		fallbackText := ""
		if len(matches) >= 4 && matches[3] != "" {
			fallbackText = matches[3]
		}
		return &channelLinkNode{ChannelID: channelID, Text: fallbackText}
	}
	return &textNode{Content: value}
}

// parseUserGroupMention parses user group mention tokens
func (c *Converter) parseUserGroupMention(value string) node {
	re := regexp.MustCompile(`^<!subteam\^([A-Z0-9]+)(\|([^>]+))?>$`)
	matches := re.FindStringSubmatch(value)
	if len(matches) >= 2 {
		groupID := matches[1]
		fallbackText := ""
		if len(matches) >= 4 && matches[3] != "" {
			fallbackText = matches[3]
		}
		return &userGroupMentionNode{GroupID: groupID, Text: fallbackText}
	}
	return &textNode{Content: value}
}

// parseSpecialMention parses special mention tokens
func (c *Converter) parseSpecialMention(value string) node {
	re := regexp.MustCompile(`^<!(here|channel|everyone)>$`)
	matches := re.FindStringSubmatch(value)
	if len(matches) >= 2 {
		mentionType := matches[1]
		return &specialMentionNode{MentionType: mentionType}
	}
	return &textNode{Content: value}
}

// parseDateFormat parses date format tokens
func (c *Converter) parseDateFormat(value string) node {
	re := regexp.MustCompile(`^<!date\^(\d+)\^([^>\^]+)(\^([^>]+))?(\|([^>]+))?>$`)
	matches := re.FindStringSubmatch(value)
	if len(matches) >= 3 {
		timestamp, _ := strconv.ParseInt(matches[1], 10, 64)
		format := matches[2]
		link := ""
		if len(matches) >= 5 && matches[4] != "" {
			link = matches[4]
		}
		fallback := ""
		if len(matches) >= 7 && matches[6] != "" {
			fallback = matches[6]
		}
		return &dateFormatNode{
			Timestamp: timestamp,
			Format:    format,
			Link:      link,
			Fallback:  fallback,
		}
	}
	return &textNode{Content: value}
}

// parseLink parses link tokens
func (c *Converter) parseLink(value string) node {
	// Try Slack-style link first: <url|text>
	re1 := regexp.MustCompile(`^<(https?://[^>|]+)\|([^>]+)>$`)
	if matches := re1.FindStringSubmatch(value); len(matches) >= 3 {
		return &linkNode{URL: matches[1], Text: matches[2]}
	}

	// Try markdown-style link: [text](url)
	re2 := regexp.MustCompile(`^\[([^\]]+)\]\((https?://[^)]+)\)$`)
	if matches := re2.FindStringSubmatch(value); len(matches) >= 3 {
		return &linkNode{URL: matches[2], Text: matches[1]}
	}

	return &textNode{Content: value}
}

// parseEmoji parses emoji tokens
func (c *Converter) parseEmoji(value string) node {
	re := regexp.MustCompile(`^:([a-zA-Z0-9_+-]+):$`)
	matches := re.FindStringSubmatch(value)
	if len(matches) >= 2 {
		emojiName := matches[1]
		return &emojiNode{Name: emojiName}
	}
	return &textNode{Content: value}
}

// extractContent extracts content using regex pattern
func extractContent(value, pattern string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(value)
	if len(matches) >= 2 {
		return matches[1]
	}
	return value
}
