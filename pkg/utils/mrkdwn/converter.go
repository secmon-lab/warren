package mrkdwn

import (
	"context"
	"sync"
	"time"
)

// SlackService defines the interface for querying Slack data
// This interface matches the methods available in pkg/service/slack.Service
type SlackService interface {
	GetUserProfile(ctx context.Context, userID string) (string, error)
	GetChannelName(ctx context.Context, channelID string) (string, error)
	GetUserGroupName(ctx context.Context, groupID string) (string, error)
}

// Cache holds cached data for Slack entities
type cache struct {
	users      map[string]cacheEntry
	channels   map[string]cacheEntry
	userGroups map[string]cacheEntry
	mutex      sync.RWMutex
}

type cacheEntry struct {
	Value     string
	ExpiresAt time.Time
}

// NewCache creates a new cache instance
func newCache() *cache {
	return &cache{
		users:      make(map[string]cacheEntry),
		channels:   make(map[string]cacheEntry),
		userGroups: make(map[string]cacheEntry),
	}
}

// Converter converts Slack mrkdwn to standard markdown
type Converter struct {
	service SlackService
	cache   *cache
	ttl     time.Duration
}

// NewConverter creates a new Converter instance
func NewConverter(service SlackService) *Converter {
	return &Converter{
		service: service,
		cache:   newCache(),
		ttl:     10 * time.Minute,
	}
}

// ConvertToMarkdown converts Slack mrkdwn to standard markdown
func (c *Converter) ConvertToMarkdown(ctx context.Context, mrkdwn string) string {
	// Parse the mrkdwn text into AST
	tokens := c.tokenize(mrkdwn)
	ast := c.parse(tokens)

	// Convert AST to markdown
	return c.renderToMarkdown(ctx, ast)
}

// token represents a lexical token
type token struct {
	Type  tokenType
	Value string
	Start int
	End   int
}

type tokenType int

const (
	tokenText tokenType = iota
	tokenBold
	tokenItalic
	tokenStrikethrough
	tokenInlineCode
	tokenCodeBlock
	tokenBlockquote
	tokenOrderedList
	tokenUnorderedList
	tokenUserMention
	tokenChannelLink
	tokenUserGroupMention
	tokenSpecialMention
	tokenDateFormat
	tokenLink
	tokenEmoji
	tokenEscape
)

// node represents an AST node
type node interface {
	Type() nodeType
}

type nodeType int

const (
	nodeDocument nodeType = iota
	nodeParagraph
	nodeText
	nodeBold
	nodeItalic
	nodeStrikethrough
	nodeInlineCode
	nodeCodeBlock
	nodeBlockquote
	nodeOrderedList
	nodeUnorderedList
	nodeListItem
	nodeUserMention
	nodeChannelLink
	nodeUserGroupMention
	nodeSpecialMention
	nodeDateFormat
	nodeLink
	nodeEmoji
)

// Concrete node implementations
type documentNode struct {
	Children []node
}

func (n *documentNode) Type() nodeType { return nodeDocument }

type textNode struct {
	Content string
}

func (n *textNode) Type() nodeType { return nodeText }

type boldNode struct {
	Children []node
}

func (n *boldNode) Type() nodeType { return nodeBold }

type italicNode struct {
	Children []node
}

func (n *italicNode) Type() nodeType { return nodeItalic }

type strikethroughNode struct {
	Children []node
}

func (n *strikethroughNode) Type() nodeType { return nodeStrikethrough }

type inlineCodeNode struct {
	Content string
}

func (n *inlineCodeNode) Type() nodeType { return nodeInlineCode }

type codeBlockNode struct {
	Content string
}

func (n *codeBlockNode) Type() nodeType { return nodeCodeBlock }

type blockquoteNode struct {
	Children []node
}

func (n *blockquoteNode) Type() nodeType { return nodeBlockquote }

type orderedListNode struct {
	Children []node
}

func (n *orderedListNode) Type() nodeType { return nodeOrderedList }

type unorderedListNode struct {
	Children []node
}

func (n *unorderedListNode) Type() nodeType { return nodeUnorderedList }

type listItemNode struct {
	Children []node
}

func (n *listItemNode) Type() nodeType { return nodeListItem }

type userMentionNode struct {
	UserID string
	Text   string // fallback text
}

func (n *userMentionNode) Type() nodeType { return nodeUserMention }

type channelLinkNode struct {
	ChannelID string
	Text      string // fallback text
}

func (n *channelLinkNode) Type() nodeType { return nodeChannelLink }

type userGroupMentionNode struct {
	GroupID string
	Text    string // fallback text
}

func (n *userGroupMentionNode) Type() nodeType { return nodeUserGroupMention }

type specialMentionNode struct {
	MentionType string // "here", "channel", "everyone"
}

func (n *specialMentionNode) Type() nodeType { return nodeSpecialMention }

type dateFormatNode struct {
	Timestamp int64
	Format    string
	Link      string
	Fallback  string
}

func (n *dateFormatNode) Type() nodeType { return nodeDateFormat }

type linkNode struct {
	URL  string
	Text string
}

func (n *linkNode) Type() nodeType { return nodeLink }

type emojiNode struct {
	Name string
}

func (n *emojiNode) Type() nodeType { return nodeEmoji }

type paragraphNode struct {
	Children []node
}

func (n *paragraphNode) Type() nodeType { return nodeParagraph }

// getCached retrieves cached value or returns empty string if not found/expired
func (c *cache) getCached(category, key string) string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var entry cacheEntry
	var found bool

	switch category {
	case "users":
		entry, found = c.users[key]
	case "channels":
		entry, found = c.channels[key]
	case "userGroups":
		entry, found = c.userGroups[key]
	}

	if !found || time.Now().After(entry.ExpiresAt) {
		return ""
	}

	return entry.Value
}

// setCached stores value in cache with TTL
func (c *cache) setCached(category, key, value string, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry := cacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}

	switch category {
	case "users":
		c.users[key] = entry
	case "channels":
		c.channels[key] = entry
	case "userGroups":
		c.userGroups[key] = entry
	}
}
