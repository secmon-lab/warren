package mrkdwn

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

// renderToMarkdown converts AST to standard markdown
func (c *Converter) renderToMarkdown(ctx context.Context, doc *documentNode) string {
	var result strings.Builder

	for _, child := range doc.Children {
		rendered := c.renderNode(ctx, child)
		result.WriteString(rendered)
	}

	return result.String()
}

// renderNode renders a single node to markdown
func (c *Converter) renderNode(ctx context.Context, node node) string {
	switch n := node.(type) {
	case *textNode:
		return n.Content

	case *boldNode:
		var content strings.Builder
		for _, child := range n.Children {
			rendered := c.renderNode(ctx, child)
			content.WriteString(rendered)
		}
		return "**" + content.String() + "**"

	case *italicNode:
		var content strings.Builder
		for _, child := range n.Children {
			rendered := c.renderNode(ctx, child)
			content.WriteString(rendered)
		}
		return "*" + content.String() + "*"

	case *strikethroughNode:
		var content strings.Builder
		for _, child := range n.Children {
			rendered := c.renderNode(ctx, child)
			content.WriteString(rendered)
		}
		return "~~" + content.String() + "~~"

	case *inlineCodeNode:
		return "`" + n.Content + "`"

	case *codeBlockNode:
		return "```\n" + n.Content + "\n```"

	case *blockquoteNode:
		var content strings.Builder
		for _, child := range n.Children {
			rendered := c.renderNode(ctx, child)
			content.WriteString(rendered)
		}
		return "> " + content.String()

	case *orderedListNode:
		var result strings.Builder
		for i, child := range n.Children {
			rendered := c.renderNode(ctx, child)
			fmt.Fprintf(&result, "%d. %s\n", i+1, rendered)
		}
		return result.String()

	case *unorderedListNode:
		var result strings.Builder
		for _, child := range n.Children {
			rendered := c.renderNode(ctx, child)
			result.WriteString("- " + rendered + "\n")
		}
		return result.String()

	case *listItemNode:
		var content strings.Builder
		for _, child := range n.Children {
			rendered := c.renderNode(ctx, child)
			content.WriteString(rendered)
		}
		return content.String()

	case *userMentionNode:
		return c.renderUserMention(ctx, n)

	case *channelLinkNode:
		return c.renderChannelLink(ctx, n)

	case *userGroupMentionNode:
		return c.renderUserGroupMention(ctx, n)

	case *specialMentionNode:
		return c.renderSpecialMention(n)

	case *dateFormatNode:
		return c.renderDateFormat(n)

	case *linkNode:
		return fmt.Sprintf("[%s](%s)", n.Text, n.URL)

	case *emojiNode:
		return ":" + n.Name + ":"

	default:
		// This should never happen with our parser, but be defensive
		return fmt.Sprintf("[unknown: %T]", node)
	}
}

// renderUserMention renders user mention node
func (c *Converter) renderUserMention(ctx context.Context, node *userMentionNode) string {
	// Check cache first
	if cached := c.cache.getCached("users", node.UserID); cached != "" {
		return "@" + cached
	}

	// Try to get user profile from service
	if c.service != nil {
		if profile, err := c.service.GetUserProfile(ctx, node.UserID); err != nil {
			errutil.Handle(ctx, goerr.Wrap(err, "failed to get user profile",
				goerr.V("userID", node.UserID)))
		} else if profile != "" {
			c.cache.setCached("users", node.UserID, profile, c.ttl)
			return "@" + profile
		}
	}

	// Fall back to provided text or user ID
	if node.Text != "" {
		return "@" + node.Text
	}
	return "@" + node.UserID
}

// renderChannelLink renders channel link node
func (c *Converter) renderChannelLink(ctx context.Context, node *channelLinkNode) string {
	// Check cache first
	if cached := c.cache.getCached("channels", node.ChannelID); cached != "" {
		return "#" + cached
	}

	// Try to get channel name from service
	if c.service != nil {
		if channelName, err := c.service.GetChannelName(ctx, node.ChannelID); err != nil {
			errutil.Handle(ctx, goerr.Wrap(err, "failed to get channel name",
				goerr.V("channelID", node.ChannelID)))
		} else if channelName != "" {
			c.cache.setCached("channels", node.ChannelID, channelName, c.ttl)
			return "#" + channelName
		}
	}

	// Fall back to provided text or channel ID
	if node.Text != "" {
		return "#" + node.Text
	}
	return "#" + node.ChannelID
}

// renderUserGroupMention renders user group mention node
func (c *Converter) renderUserGroupMention(ctx context.Context, node *userGroupMentionNode) string {
	// Check cache first
	if cached := c.cache.getCached("userGroups", node.GroupID); cached != "" {
		return "@" + cached
	}

	// Try to get user group name from service
	if c.service != nil {
		if groupName, err := c.service.GetUserGroupName(ctx, node.GroupID); err != nil {
			errutil.Handle(ctx, goerr.Wrap(err, "failed to get user group name",
				goerr.V("groupID", node.GroupID)))
		} else if groupName != "" {
			c.cache.setCached("userGroups", node.GroupID, groupName, c.ttl)
			return "@" + groupName
		}
	}

	// Fall back to provided text or group ID
	if node.Text != "" {
		return "@" + node.Text
	}
	return "@" + node.GroupID
}

// renderSpecialMention renders special mention node
func (c *Converter) renderSpecialMention(node *specialMentionNode) string {
	switch node.MentionType {
	case "here":
		return "@here"
	case "channel":
		return "@channel"
	case "everyone":
		return "@everyone"
	default:
		return "@" + node.MentionType
	}
}

// renderDateFormat renders date format node
func (c *Converter) renderDateFormat(node *dateFormatNode) string {
	// If fallback text is available, use it
	if node.Fallback != "" {
		if node.Link != "" {
			return fmt.Sprintf("[%s](%s)", node.Fallback, node.Link)
		}
		return node.Fallback
	}

	// Otherwise, try to format the timestamp
	t := time.Unix(node.Timestamp, 0)
	formattedDate := formatDate(t, node.Format)

	if node.Link != "" {
		return fmt.Sprintf("[%s](%s)", formattedDate, node.Link)
	}
	return formattedDate
}

// formatDate formats a timestamp according to Slack date format tokens
func formatDate(t time.Time, format string) string {
	// Convert to UTC to ensure consistent behavior across different timezones
	utc := t.UTC()

	switch format {
	case "{date_num}":
		return utc.Format("2006-01-02")
	case "{date}":
		return utc.Format("January 2, 2006")
	case "{date_short}":
		return utc.Format("Jan 2, 2006")
	case "{date_long}":
		return utc.Format("Monday, January 2, 2006")
	case "{time}":
		return utc.Format("3:04 PM")
	case "{time_secs}":
		return utc.Format("3:04:05 PM")
	default:
		// For unknown formats or combinations, use ISO format in UTC
		return utc.Format("2006-01-02 15:04:05")
	}
}
