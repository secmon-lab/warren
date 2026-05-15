package webfetch_test

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/webfetch"
	"github.com/secmon-lab/warren/pkg/utils/safe"
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
<h3>SubSub</h3>
<h6>Deep</h6>
<p>Body paragraph.</p>
<ul>
  <li>first</li>
  <li>second</li>
</ul>
<pre><code>code line</code></pre>
</body></html>`)

	text, _, err := webfetch.Extract("text/html", body)
	gt.NoError(t, err).Required()
	// Validate the exact heading level by anchoring to line boundaries —
	// substring matching alone would let `## Subtitle` pass against
	// `###### Subtitle` and hide a level-calculation bug.
	gt.True(t, regexp.MustCompile(`(?m)^# Title$`).MatchString(text))
	gt.True(t, regexp.MustCompile(`(?m)^## Subtitle$`).MatchString(text))
	gt.True(t, regexp.MustCompile(`(?m)^### SubSub$`).MatchString(text))
	gt.True(t, regexp.MustCompile(`(?m)^###### Deep$`).MatchString(text))
	gt.S(t, text).NotContains("####### ") // never produce 7+ hashes
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

// Live extraction tests against real websites that the warren agent is likely
// to fetch during security investigations. Each case asserts that:
//   - the response can be fetched and parsed,
//   - extract returns isHTML=true,
//   - identifying keywords appear in the extracted text (case-insensitive),
//   - common HTML noise (<script>, <style>) is absent.
//
// Transient upstream failures (network errors, 4xx/5xx anti-bot responses)
// produce a per-case skip rather than a hard failure so that test runs are
// not blocked by third-party site flakiness.

type liveExtractCase struct {
	name        string
	url         string
	mustContain []string // case-insensitive substring match
}

var liveExtractCases = []liveExtractCase{
	{
		name:        "nvd_cve_log4shell",
		url:         "https://nvd.nist.gov/vuln/detail/CVE-2021-44228",
		mustContain: []string{"CVE-2021-44228", "log4j"},
	},
	{
		name:        "mitre_attack_t1059",
		url:         "https://attack.mitre.org/techniques/T1059/",
		mustContain: []string{"T1059", "scripting interpreter"},
	},
	{
		name:        "wikipedia_heartbleed",
		url:         "https://en.wikipedia.org/wiki/Heartbleed",
		mustContain: []string{"Heartbleed", "OpenSSL"},
	},
	{
		name:        "github_advisory_log4j",
		url:         "https://github.com/advisories/GHSA-jfh8-c2jp-5v3q",
		mustContain: []string{"log4j"},
	},
	{
		name:        "owasp_password_storage",
		url:         "https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html",
		mustContain: []string{"password", "hash"},
	},
}

func TestExtract_Live_RealWebsites(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}

	for _, tc := range liveExtractCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, tc.url, nil)
			gt.NoError(t, err).Required()
			req.Header.Set("User-Agent", "warren-webfetch-test/1.0 (+https://github.com/secmon-lab/warren)")

			resp, err := client.Do(req)
			if err != nil {
				t.Skipf("fetch failed (likely transient): %v", err)
			}
			defer safe.Close(t.Context(), resp.Body)

			if resp.StatusCode >= 400 {
				t.Skipf("upstream returned HTTP %d (likely transient or anti-bot)", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			gt.NoError(t, err).Required()

			text, isHTML, err := webfetch.Extract(resp.Header.Get("Content-Type"), body)
			gt.NoError(t, err).Required()
			gt.True(t, isHTML)

			lower := strings.ToLower(text)
			for _, want := range tc.mustContain {
				if !strings.Contains(lower, strings.ToLower(want)) {
					t.Errorf("expected extracted text to contain %q, but it did not. extracted prefix: %.300q",
						want, text)
				}
			}

			// Common HTML noise must not survive the extraction.
			gt.S(t, text).NotContains("<script")
			gt.S(t, text).NotContains("</script>")
			gt.S(t, text).NotContains("<style")
		})
	}
}
