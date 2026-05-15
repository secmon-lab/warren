# WebFetch

Fetch a web page and return its body as Markdown after extracting the main content and screening for indirect prompt injection.

## Overview

`web_fetch` follows a 3-stage pipeline:

1. **Fetch** the URL over HTTP/HTTPS (30 s timeout, fixed User-Agent). No response-size cap is applied; the request timeout and the context deadline bound the operation.
2. **Extract** the textual body. HTML pages are parsed with `golang.org/x/net/html`, stripped of `<script>` / `<style>` / `<svg>` / hidden nodes / comments, and rendered as semi-structured plain text that preserves headings, lists, code blocks, and tables. Text-based responses (`text/plain`, `application/json`, etc.) pass through verbatim. Binary content types are rejected.
3. **Analyze** the body through an LLM call that (a) reformats the content into clean Markdown and (b) checks for indirect prompt injection (role-change attempts, system-prompt overrides, control-token-style strings, etc.). The body is delivered as a `user`-role message; all instructions live in the `system` prompt, which declares the user message untrusted.

If the LLM judges the body malicious, the tool returns an error tagged `validation` with the LLM-supplied reason. Otherwise, it returns the formatted Markdown.

## Configuration

No environment variables or CLI flags. The tool is always available when the LLM client is configured.

## Available Functions

| Function | Description |
|---|---|
| `web_fetch` | Fetch the given URL and return Markdown extracted from its body. Argument: `url` (http/https only). |

## Setup

No external API keys. The tool relies on the warren-wide LLM client (Gemini via `gollem`) which is injected at start-up.

`web_fetch` is registered as a HITL (Human-in-the-Loop) tool: every invocation requires user approval before the request is made. This is configured in `pkg/cli/serve.go` via `aster.WithHITLTools` / `bluebell.WithHITLTools`.

## Limitations

- **No URL allow/deny filtering.** Access control is delegated to HITL approval. Any reachable HTTP/HTTPS URL is fetchable once approved.
- **No SSRF protection** beyond scheme restriction. Internal IPs and metadata endpoints are not blocked at this layer.
- **UTF-8 assumed.** The `Content-Type` charset is not honored; non-UTF-8 pages may produce garbled output.
- **No summarization.** The LLM only reformats — it does not condense the body. Large pages (after extraction) are passed to the upstream agent as-is.
- **No retries.** Transient LLM or HTTP failures bubble up to the agent loop, which owns the retry strategy.

## Tests

The package tests cover both pure logic and external integrations:

- **Unit / extract / analyze mock tests** — always run as part of `go test ./pkg/tool/webfetch/...`.
- **`TestExtract_Live_RealWebsites`** — always runs. Fetches a small set of real security-info sites (NVD, MITRE ATT&CK, Wikipedia, GitHub Security Advisory, OWASP) and asserts that extraction produces non-empty text containing expected keywords and no surviving `<script>` / `<style>` markup. Per-case transient failures (network errors, anti-bot 4xx/5xx) are converted into per-case skips rather than hard failures.
- **`TestAnalyze_Live_*`** — runs when both `TEST_GEMINI_PROJECT_ID` and `TEST_GEMINI_LOCATION` are set. Verifies indirect-prompt-injection detection against a real Gemini endpoint.
