package mrkdwn

import "context"

// Export types for testing
type TestToken = token
type TestTokenType = tokenType
type TestNode = node
type TestNodeType = nodeType

// Export token types for testing
const (
	TestTokenText             = tokenText
	TestTokenBold             = tokenBold
	TestTokenItalic           = tokenItalic
	TestTokenStrikethrough    = tokenStrikethrough
	TestTokenInlineCode       = tokenInlineCode
	TestTokenCodeBlock        = tokenCodeBlock
	TestTokenBlockquote       = tokenBlockquote
	TestTokenOrderedList      = tokenOrderedList
	TestTokenUnorderedList    = tokenUnorderedList
	TestTokenUserMention      = tokenUserMention
	TestTokenChannelLink      = tokenChannelLink
	TestTokenUserGroupMention = tokenUserGroupMention
	TestTokenSpecialMention   = tokenSpecialMention
	TestTokenDateFormat       = tokenDateFormat
	TestTokenLink             = tokenLink
	TestTokenEmoji            = tokenEmoji
	TestTokenEscape           = tokenEscape
)

// Export node types for testing
const (
	TestNodeDocument         = nodeDocument
	TestNodeParagraph        = nodeParagraph
	TestNodeText             = nodeText
	TestNodeBold             = nodeBold
	TestNodeItalic           = nodeItalic
	TestNodeStrikethrough    = nodeStrikethrough
	TestNodeInlineCode       = nodeInlineCode
	TestNodeCodeBlock        = nodeCodeBlock
	TestNodeBlockquote       = nodeBlockquote
	TestNodeOrderedList      = nodeOrderedList
	TestNodeUnorderedList    = nodeUnorderedList
	TestNodeListItem         = nodeListItem
	TestNodeUserMention      = nodeUserMention
	TestNodeChannelLink      = nodeChannelLink
	TestNodeUserGroupMention = nodeUserGroupMention
	TestNodeSpecialMention   = nodeSpecialMention
	TestNodeDateFormat       = nodeDateFormat
	TestNodeLink             = nodeLink
	TestNodeEmoji            = nodeEmoji
)

// Export node types for testing
type TestDocumentNode = documentNode
type TestTextNode = textNode
type TestBoldNode = boldNode
type TestItalicNode = italicNode
type TestStrikethroughNode = strikethroughNode
type TestInlineCodeNode = inlineCodeNode
type TestCodeBlockNode = codeBlockNode
type TestBlockquoteNode = blockquoteNode
type TestOrderedListNode = orderedListNode
type TestUnorderedListNode = unorderedListNode
type TestListItemNode = listItemNode
type TestUserMentionNode = userMentionNode
type TestChannelLinkNode = channelLinkNode
type TestUserGroupMentionNode = userGroupMentionNode
type TestSpecialMentionNode = specialMentionNode
type TestDateFormatNode = dateFormatNode
type TestLinkNode = linkNode
type TestEmojiNode = emojiNode
type TestParagraphNode = paragraphNode

// Export cache types for testing
type TestCache = cache
type TestCacheEntry = cacheEntry

// Export methods for testing
func (c *Converter) TestTokenize(text string) []token {
	return c.tokenize(text)
}

func (c *Converter) TestParse(tokens []token) *documentNode {
	return c.parse(tokens)
}

func (c *Converter) TestRenderToMarkdown(ctx context.Context, doc *documentNode) string {
	return c.renderToMarkdown(ctx, doc)
}

func NewTestCache() *cache {
	return newCache()
}
