package webfetch

import (
	"bytes"
	"mime"
	"regexp"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// extract converts an HTTP response body into plain text suitable for downstream LLM processing.
//
// For HTML / XHTML, the body is parsed, stripped of non-content elements
// (script/style/etc., hidden nodes, comments), and rendered as semi-structured
// plain text that preserves headings, lists, code blocks, and tables.
//
// For other text-based media types (text/plain, application/json, etc.), the
// body is returned verbatim.
//
// Binary media types are rejected with a TagValidation error.
func extract(contentType string, body []byte) (text string, isHTML bool, err error) {
	mediaType, _, mtErr := mime.ParseMediaType(contentType)
	if mtErr != nil {
		mediaType = strings.TrimSpace(strings.ToLower(contentType))
	}
	mediaType = strings.ToLower(mediaType)

	switch mediaType {
	case "text/html", "application/xhtml+xml":
		s, hErr := renderHTML(body)
		if hErr != nil {
			return "", true, goerr.Wrap(hErr, "failed to parse HTML",
				goerr.T(errutil.TagExternal),
				goerr.V("content_type", contentType))
		}
		return s, true, nil
	case
		"text/plain",
		"text/markdown",
		"text/csv",
		"text/xml",
		"application/xml",
		"application/json",
		"application/x-ndjson",
		"application/yaml",
		"application/x-yaml":
		return string(body), false, nil
	default:
		return "", false, goerr.New("unsupported content type for webfetch",
			goerr.T(errutil.TagValidation),
			goerr.V("content_type", contentType))
	}
}

// dropTags lists HTML elements whose entire subtree is removed before rendering.
var dropTags = map[atom.Atom]bool{
	atom.Script:   true,
	atom.Style:    true,
	atom.Noscript: true,
	atom.Template: true,
	atom.Svg:      true,
	atom.Canvas:   true,
	atom.Iframe:   true,
	atom.Object:   true,
	atom.Embed:    true,
	atom.Link:     true,
	atom.Meta:     true,
}

var displayNoneRe = regexp.MustCompile(`(?i)(?:^|;)\s*(?:display\s*:\s*none|visibility\s*:\s*hidden)\b`)

// renderHTML parses the body and produces a plain-text rendering that
// preserves enough structural cues (headings, list markers, code fences,
// table separators) for an LLM to reconstruct Markdown from it.
func renderHTML(body []byte) (string, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	renderNode(&sb, doc)
	return collapseWhitespace(sb.String()), nil
}

// hasHiddenAttr reports whether a node carries an attribute that should hide it
// from the rendered output (HTML "hidden" attribute or inline display:none / visibility:hidden).
func hasHiddenAttr(n *html.Node) bool {
	for _, a := range n.Attr {
		k := strings.ToLower(a.Key)
		if k == "hidden" {
			return true
		}
		if k == "style" && displayNoneRe.MatchString(a.Val) {
			return true
		}
	}
	return false
}

func renderNode(sb *strings.Builder, n *html.Node) {
	switch n.Type {
	case html.CommentNode:
		return
	case html.TextNode:
		sb.WriteString(n.Data)
		return
	case html.ElementNode:
		if dropTags[n.DataAtom] {
			return
		}
		if hasHiddenAttr(n) {
			return
		}
		renderElement(sb, n)
		return
	default:
		// DocumentNode / DoctypeNode: descend into children.
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		renderNode(sb, c)
	}
}

func renderElement(sb *strings.Builder, n *html.Node) {
	switch n.DataAtom {
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		// atom.H1..H6 are NOT sequential integer constants — use an explicit map.
		var level int
		switch n.DataAtom {
		case atom.H1:
			level = 1
		case atom.H2:
			level = 2
		case atom.H3:
			level = 3
		case atom.H4:
			level = 4
		case atom.H5:
			level = 5
		case atom.H6:
			level = 6
		}
		sb.WriteString("\n\n")
		sb.WriteString(strings.Repeat("#", level))
		sb.WriteString(" ")
		renderChildren(sb, n)
		sb.WriteString("\n\n")
		return
	case atom.P, atom.Div, atom.Section, atom.Article, atom.Header, atom.Footer,
		atom.Main, atom.Aside, atom.Nav, atom.Figure, atom.Blockquote:
		sb.WriteString("\n\n")
		renderChildren(sb, n)
		sb.WriteString("\n\n")
		return
	case atom.Br:
		sb.WriteString("\n")
		return
	case atom.Hr:
		sb.WriteString("\n---\n")
		return
	case atom.Ul, atom.Ol:
		sb.WriteString("\n\n")
		renderChildren(sb, n)
		sb.WriteString("\n\n")
		return
	case atom.Li:
		sb.WriteString("\n- ")
		renderChildren(sb, n)
		return
	case atom.Pre:
		sb.WriteString("\n\n```\n")
		renderChildren(sb, n)
		sb.WriteString("\n```\n\n")
		return
	case atom.Code:
		// Inline code only when not nested in <pre>.
		if isInsidePre(n) {
			renderChildren(sb, n)
			return
		}
		sb.WriteString("`")
		renderChildren(sb, n)
		sb.WriteString("`")
		return
	case atom.A:
		// Drop the href; keep only the visible text.
		renderChildren(sb, n)
		return
	case atom.Table:
		sb.WriteString("\n\n")
		renderChildren(sb, n)
		sb.WriteString("\n\n")
		return
	case atom.Tr:
		sb.WriteString("\n")
		first := true
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode {
				continue
			}
			if c.DataAtom != atom.Td && c.DataAtom != atom.Th {
				continue
			}
			if !first {
				sb.WriteString(" | ")
			}
			renderChildren(sb, c)
			first = false
		}
		return
	}

	renderChildren(sb, n)
}

func renderChildren(sb *strings.Builder, n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		renderNode(sb, c)
	}
}

func isInsidePre(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.DataAtom == atom.Pre {
			return true
		}
	}
	return false
}

// collapseWhitespace normalizes whitespace produced by the renderer:
//   - sequences of spaces/tabs collapse to a single space
//   - 3+ consecutive newlines collapse to two
//   - leading and trailing whitespace are trimmed
func collapseWhitespace(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	var prevSpace, prevNewline bool
	newlineRun := 0
	for _, r := range s {
		switch r {
		case '\n':
			newlineRun++
			if newlineRun <= 2 {
				sb.WriteRune('\n')
			}
			prevSpace = false
			prevNewline = true
		case ' ', '\t':
			newlineRun = 0
			if prevSpace || prevNewline {
				// Skip leading-of-line spaces and runs of spaces.
				continue
			}
			sb.WriteRune(' ')
			prevSpace = true
			prevNewline = false
		default:
			newlineRun = 0
			sb.WriteRune(r)
			prevSpace = false
			prevNewline = false
		}
	}

	return strings.TrimSpace(sb.String())
}
