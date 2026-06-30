# WebFetch

Fetch a web page and return its body. When an LLM is configured, the body is reformatted as Markdown and screened for indirect prompt injection; otherwise the extracted text is returned verbatim and every call is gated by a HITL approval dialog.

## Overview

`web_fetch` follows a 3-stage pipeline:

1. **Fetch** the URL over HTTP/HTTPS (30 s timeout, fixed User-Agent). No response-size cap is applied; the request timeout and the context deadline bound the operation.
2. **Extract** the textual body. HTML pages are parsed with `golang.org/x/net/html`, stripped of `<script>` / `<style>` / `<svg>` / hidden nodes / comments, and rendered as semi-structured plain text that preserves headings, lists, code blocks, and tables. Text-based responses (`text/plain`, `application/json`, etc.) pass through verbatim. Binary content types are rejected.
3. **Analyze** (optional). When an LLM is configured via `--webfetch-llm-*` flags, the body goes through an LLM call that (a) reformats the content into clean Markdown and (b) checks for indirect prompt injection (role-change attempts, system-prompt overrides, control-token-style strings, etc.). The body is delivered as a `user`-role message; all instructions live in the `system` prompt, which declares the user message untrusted. When no LLM is configured, this stage is skipped — the extracted text is returned verbatim with `"llm_analysis": "disabled"` in the response.

If the LLM judges the body malicious, the tool returns an error tagged `validation` with the LLM-supplied reason. Otherwise, it returns the formatted Markdown (or the raw extracted text in LLM-disabled mode).

## Configuration

The webfetch tool owns its LLM configuration independently from warren's main `--gemini-*` / `--claude-*` flags, so the screening provider can differ from (or be absent compared to) the main warren LLM.

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_WEBFETCH_LLM_PROVIDER` | `--webfetch-llm-provider` | _(empty)_ | LLM provider for analyze step: `gemini`, `claude`, or `openai`. Empty disables LLM analysis. |
| `WARREN_WEBFETCH_LLM_MODEL` | `--webfetch-llm-model` | _(empty)_ | LLM model name. Required when provider is set. |
| `WARREN_WEBFETCH_LLM_ARGS` | `--webfetch-llm-args` | _(empty)_ | Provider-specific options as `key=value,key=value`. Recognized: `project_id`, `location`, `temperature`. |
| `WARREN_WEBFETCH_LLM_API_KEY` | `--webfetch-llm-api-key` | _(empty)_ | API key. Required for `openai` and `claude` Anthropic-direct. Ignored for `gemini` and `claude` Vertex. |

### Claude routing

The `claude` provider auto-selects between Vertex AI and the direct Anthropic API based on the supplied inputs:

| Inputs | Route |
|---|---|
| `project_id` + `location` in `--webfetch-llm-args`, no api-key | Vertex AI |
| `--webfetch-llm-api-key` set, no `project_id` / `location` | Anthropic direct |
| Both Vertex args and api-key | _Start-up error (ambiguous)_ |
| Neither | _Start-up error (route unspecified)_ |

### Examples

Gemini on Vertex AI:

```bash
--webfetch-llm-provider gemini
--webfetch-llm-model    gemini-2.5-flash
--webfetch-llm-args     "project_id=my-proj,location=us-central1"
```

Claude on Vertex AI:

```bash
--webfetch-llm-provider claude
--webfetch-llm-model    "claude-sonnet-4@20250514"
--webfetch-llm-args     "project_id=my-proj,location=us-east5"
```

Claude via Anthropic direct API:

```bash
--webfetch-llm-provider claude
--webfetch-llm-model    claude-sonnet-4-5-20250929
--webfetch-llm-api-key  "$ANTHROPIC_API_KEY"
```

OpenAI:

```bash
--webfetch-llm-provider openai
--webfetch-llm-model    gpt-4o
--webfetch-llm-args     "temperature=0.2"
--webfetch-llm-api-key  "$OPENAI_API_KEY"
```

LLM disabled (no flags): every call is HITL-gated, response includes `"llm_analysis": "disabled"`.

### Start-up ping

Whenever `--webfetch-llm-provider` is set, warren issues a minimal `max_tokens=1` generation request at start-up so misconfigurations surface immediately instead of at first use. The ping cost is negligible (single-digit tokens) but it catches:

- API key typos or expired keys
- Model name typos
- Vertex AI `project_id` / `location` mistakes or permission gaps
- Network reachability issues

The ping covers the analyze provider only; the warren-wide LLM (Gemini/Claude) is independent.

## Available Functions

| Function | Description |
|---|---|
| `web_fetch` | Fetch the given URL and return Markdown (when LLM is configured) or raw extracted text (when not). Argument: `url` (http/https only). |

## HITL (Human-in-the-Loop)

`web_fetch` is conditionally registered as a HITL-gated tool. The decision is driven by `--webfetch-llm-provider`:

- **`--webfetch-llm-provider` empty (LLM disabled)** → HITL approval dialog is shown for every `web_fetch` call. The human is the only safety net since indirect-prompt-injection screening does not run in this mode.
- **`--webfetch-llm-provider` set (LLM enabled)** → HITL approval is suppressed. The LLM screening handles IPI detection.

The wiring lives in `pkg/cli/serve.go`: it consults each tool's `RequiresHITL()` method (implemented by webfetch) and passes the resulting names to `aster.WithHITLTools` / `bluebell.WithHITLTools`.

## Setup

No external API keys are required when running in LLM-disabled mode (every call goes through HITL approval). When LLM analysis is enabled, the API key requirement depends on the provider — see the routing table above.

## Limitations

- **No URL allow/deny filtering.** Access control is delegated to the HITL approval dialog (LLM-disabled mode) and / or the LLM IPI screen (LLM-enabled mode).
- **SSRF guard blocks private/internal IPs by default.** The underlying `github.com/gollem-dev/tools/webfetch` module refuses to fetch URLs that resolve to private, loopback, or link-local IP ranges (including cloud metadata endpoints), so internal-network targets are rejected before the request is sent. warren runs with this guard enabled and does not expose an opt-out, so `web_fetch` cannot reach internal hosts.
- **UTF-8 assumed.** The `Content-Type` charset is not honored; non-UTF-8 pages may produce garbled output.
- **No summarization.** The LLM only reformats — it does not condense the body. Large pages (after extraction) are passed to the upstream agent as-is.
- **No retries.** Transient LLM or HTTP failures bubble up to the agent loop, which owns the retry strategy.

## Tests

The package tests cover both pure logic and external integrations:

- **Unit tests for the LLM-flag parser / builder / ping helper** (`llmflag_test.go`) — always run; uses gollem mocks for the ping path.
- **Unit / extract / analyze mock tests** — always run as part of `go test ./pkg/tool/webfetch/...`.
- **`TestExtract_Live_RealWebsites`** — always runs. Fetches a small set of real security-info sites (NVD, MITRE ATT&CK, Wikipedia, GitHub Security Advisory, OWASP) and asserts that extraction produces non-empty text containing expected keywords and no surviving `<script>` / `<style>` markup. Per-case transient failures (network errors, anti-bot 4xx/5xx) are converted into per-case skips rather than hard failures.
- **`TestAnalyze_Live_*`** — runs when both `TEST_GEMINI_PROJECT_ID` and `TEST_GEMINI_LOCATION` are set. Verifies indirect-prompt-injection detection against a real Gemini endpoint.
