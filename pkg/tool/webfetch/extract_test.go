package webfetch_test

import (
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/webfetch"
)

func TestExtract_HTML_DropsScriptAndStyle(t *testing.T) {
	body := []byte(`
<html><head>
<title>Ignored Title Element</title>
<style>body { color: red; }</style>
<script>alert("xss");</script>
<meta charset="utf-8">
<link rel="stylesheet" href="x">
</head>
<body>
<p>Hello world</p>
<noscript>fallback</noscript>
</body></html>`)

	text, isHTML, err := webfetch.Extract("text/html; charset=utf-8", body)
	gt.NoError(t, err).Required()
	gt.True(t, isHTML)
	gt.S(t, text).NotContains("alert(\"xss\")")
	gt.S(t, text).NotContains("color: red")
	gt.S(t, text).NotContains("fallback")
	gt.S(t, text).Contains("Hello world")
}

func TestExtract_HTML_PreservesHeadingsAndLists(t *testing.T) {
	body := []byte(`
<html><body>
<h1>Title</h1>
<h2>Subtitle</h2>
<p>Body paragraph.</p>
<ul>
  <li>first</li>
  <li>second</li>
</ul>
<pre><code>code line</code></pre>
</body></html>`)

	text, _, err := webfetch.Extract("text/html", body)
	gt.NoError(t, err).Required()
	gt.S(t, text).Contains("# Title")
	gt.S(t, text).Contains("## Subtitle")
	gt.S(t, text).Contains("- first")
	gt.S(t, text).Contains("- second")
	gt.S(t, text).Contains("```")
	gt.S(t, text).Contains("code line")
}

func TestExtract_HTML_RendersTable(t *testing.T) {
	body := []byte(`
<table>
  <tr><th>name</th><th>value</th></tr>
  <tr><td>foo</td><td>1</td></tr>
  <tr><td>bar</td><td>2</td></tr>
</table>`)

	text, _, err := webfetch.Extract("text/html", body)
	gt.NoError(t, err).Required()
	gt.S(t, text).Contains("name | value")
	gt.S(t, text).Contains("foo | 1")
	gt.S(t, text).Contains("bar | 2")
}

func TestExtract_HTML_DropsHTMLComments(t *testing.T) {
	body := []byte(`<html><body><p>visible</p><!-- secret comment --></body></html>`)
	text, _, err := webfetch.Extract("text/html", body)
	gt.NoError(t, err).Required()
	gt.S(t, text).NotContains("secret comment")
	gt.S(t, text).Contains("visible")
}

func TestExtract_HTML_DropsHiddenAndDisplayNone(t *testing.T) {
	body := []byte(`
<html><body>
<p>shown</p>
<p hidden>hidden via attr</p>
<div style="display:none">hidden via style</div>
<div style="visibility: hidden">hidden via visibility</div>
</body></html>`)

	text, _, err := webfetch.Extract("text/html", body)
	gt.NoError(t, err).Required()
	gt.S(t, text).Contains("shown")
	gt.S(t, text).NotContains("hidden via attr")
	gt.S(t, text).NotContains("hidden via style")
	gt.S(t, text).NotContains("hidden via visibility")
}

func TestExtract_HTML_CollapsesWhitespace(t *testing.T) {
	body := []byte(`<html><body>   <p>one     two</p>   <p>three</p>   </body></html>`)
	text, _, err := webfetch.Extract("text/html", body)
	gt.NoError(t, err).Required()
	// "one     two" should collapse to "one two"
	gt.S(t, text).Contains("one two")
	gt.S(t, text).NotContains("one     two")
	// No leading whitespace
	gt.False(t, strings.HasPrefix(text, " "))
	gt.False(t, strings.HasPrefix(text, "\n"))
}

func TestExtract_HTML_MalformedDoesNotPanic(t *testing.T) {
	// Various malformed HTML inputs should not crash.
	cases := []string{
		`<html><body><p>unclosed`,
		`<<<<html>`,
		`<p><div><span>nested without close`,
		``,
		`plain text without tags`,
	}
	for _, c := range cases {
		_, _, err := webfetch.Extract("text/html", []byte(c))
		gt.NoError(t, err)
	}
}

func TestExtract_HTML_DropsAnchorURL(t *testing.T) {
	body := []byte(`<p>See <a href="https://example.com/secret">our docs</a> for details.</p>`)
	text, _, err := webfetch.Extract("text/html", body)
	gt.NoError(t, err).Required()
	gt.S(t, text).Contains("our docs")
	gt.S(t, text).NotContains("https://example.com/secret")
}

func TestExtract_PlainTextReturnedVerbatim(t *testing.T) {
	body := []byte("line 1\nline 2\nline 3")
	text, isHTML, err := webfetch.Extract("text/plain; charset=utf-8", body)
	gt.NoError(t, err).Required()
	gt.False(t, isHTML)
	gt.Value(t, text).Equal("line 1\nline 2\nline 3")
}

func TestExtract_JSONReturnedVerbatim(t *testing.T) {
	body := []byte(`{"foo":"bar"}`)
	text, isHTML, err := webfetch.Extract("application/json", body)
	gt.NoError(t, err).Required()
	gt.False(t, isHTML)
	gt.Value(t, text).Equal(`{"foo":"bar"}`)
}

func TestExtract_BinaryRejected(t *testing.T) {
	cases := []string{
		"application/pdf",
		"image/png",
		"application/octet-stream",
		"video/mp4",
	}
	for _, ct := range cases {
		_, _, err := webfetch.Extract(ct, []byte("binary"))
		gt.Error(t, err)
	}
}
