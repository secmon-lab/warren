# LLM configuration

Warren reads its LLM provider/model configuration from a TOML file pointed to by
`--llm-config` (or `WARREN_LLM_CONFIG`). LLM is mandatory — there is no
"disable" mode and no command-line fallback for provider/model selection.

The file declares a pool of LLMs in `[[llm]]` blocks and assigns them roles via
the `[agent]` section. The bluebell chat strategy uses the `main` LLM for
planning / replanning / final synthesis, and the planner picks one of the
`task` LLMs per task at plan time. This lets you route cheap one-shot lookups
to a small model while keeping reasoning on a larger one.

A complete example lives at [`llm.toml.example`](./llm.toml.example).

## Loading and template rendering

The file is read, then run through `text/template` with the strict option
`missingkey=error`. The only context exposed is `.Env`, a map of the process
environment. Reference variables as `{{ .Env.MY_API_KEY }}`.

```toml
claude = { api_key = "{{ .Env.ANTHROPIC_API_KEY }}" }
```

Missing variables fail loud at startup — Warren refuses to boot rather than
silently substitute an empty string.

After rendering, validation runs across all sections and emits **all** errors
at once via `errors.Join` so a single startup attempt surfaces every problem.
On success, only the LLMs referenced by `[agent].main` or `[agent].task` are
instantiated; entries defined but never referenced are warned about and left
unloaded.

A startup health check then issues a one-token "ping" prompt to every loaded
client in parallel (30s timeout). If any fail, startup fails.

## Sections

### `[agent]`

| field  | type       | required | description                                                                  |
| ------ | ---------- | -------- | ---------------------------------------------------------------------------- |
| `main` | string     | yes      | id of the `[[llm]]` entry used by the planner / replanner / final synthesis. |
| `task` | `[]string` | yes      | Allow-list of `[[llm]]` ids the planner may pick for individual tasks.       |

```toml
[agent]
main = "claude-sonnet"
task = ["claude-sonnet", "gemini-flash", "gemini-pro"]
```

The planner is shown each task LLM's `id`, `description`, `provider`, and
`model`, and writes a `llm_id` field per task in its plan output. At task
execution time, Warren resolves the id against the registry; ids outside the
`task` allow-list are rejected even if defined in `[[llm]]`.

### `[[llm]]`

| field         | type   | required | description                                                       |
| ------------- | ------ | -------- | ----------------------------------------------------------------- |
| `id`          | string | yes      | Unique identifier referenced from `[agent]` and the planner.      |
| `description` | string | yes      | Free-form text shown to the planner — explain when to choose it.  |
| `provider`    | string | yes      | `"claude"` or `"gemini"`.                                         |
| `model`       | string | yes      | Provider-specific model name (e.g. `claude-sonnet-4-6`).          |
| `claude`      | table  | provider | Required when `provider = "claude"`; forbidden otherwise.         |
| `gemini`      | table  | provider | Required when `provider = "gemini"`; forbidden otherwise.         |

#### `claude = { ... }`

Vertex AI mode and Anthropic API key mode are mutually exclusive.

| field        | description                                                                  |
| ------------ | ---------------------------------------------------------------------------- |
| `project_id` | GCP project id (Vertex mode). Pair with `location`.                          |
| `location`   | Vertex region (e.g. `us-east5`). Pair with `project_id`.                     |
| `api_key`    | Anthropic API key. Mutually exclusive with `project_id` / `location`.        |

#### `gemini = { ... }`

Vertex AI only — gollem does not currently expose Gemini's API-key direct mode,
and configs that set `api_key` here are rejected at startup.

| field             | description                                                                                |
| ----------------- | ------------------------------------------------------------------------------------------ |
| `project_id`      | GCP project id (required).                                                                 |
| `location`        | Vertex region (required).                                                                  |
| `thinking_budget` | Optional integer. Defaults to `0` (thinking disabled, matching prior warren behavior).     |

### `[embedding]`

Embeddings are Gemini-only via Vertex.

| field        | description                                                              |
| ------------ | ------------------------------------------------------------------------ |
| `provider`   | Must be `"gemini"`.                                                      |
| `model`      | Embedding model name (e.g. `text-embedding-004`).                        |
| `project_id` | GCP project id.                                                          |
| `location`   | Vertex region.                                                           |
| `api_key`    | Reserved; rejected at validation. Use Vertex.                            |

## Picking models in practice

The planner sees descriptions and decides per task. Make `description`
opinionated:

- **Cheap fast lane** — for "is this IP in our intel feed", "summarize this
  log", "format this response". Mention the cost tier explicitly.
- **Reasoning lane** — for multi-step triage, correlation, exfiltration
  hypotheses. Note when the heavier model is *justified*.
- **Distribute quota** — define two `[[llm]]` entries with the same
  provider/model but different ids/descriptions when you want the planner to
  spread load across separate quota pools.

The planner's prompt explicitly asks it to pick the cheapest entry whose
description matches the task, and to distribute load when entries are
functionally equivalent.

## Common validation errors

| symptom                                                          | cause                                                                |
| ---------------------------------------------------------------- | -------------------------------------------------------------------- |
| `[agent].main does not match any [[llm]] entry`                  | typo in `main` or the corresponding `[[llm]].id`.                    |
| `[agent].task entry does not match any [[llm]]`                  | id in `task` array isn't defined in any `[[llm]]` block.             |
| `[[llm]] id is duplicated`                                       | two `[[llm]]` blocks share the same `id`.                            |
| `mode is ambiguous: both vertex … and api_key are set`           | a `claude = { … }` table sets both modes at once.                    |
| `gemini api_key mode is not supported`                           | Gemini configured with `api_key`. Use Vertex.                        |
| `template … map has no entry for key "MY_VAR"`                   | a `{{ .Env.MY_VAR }}` reference but the env var is unset.            |

## Migration from `--gemini-*` / `--claude-*` flags

Previous releases configured a single Gemini (and optionally Claude) client
through CLI flags. Replace those with a single TOML file and set
`--llm-config`. The `--disable-llm` flag has been removed; LLM is required.

```diff
- warren serve \
-   --gemini-project-id "$GCP_PROJECT" \
-   --gemini-location us-central1 \
-   --gemini-model gemini-2.5-pro
+ warren serve --llm-config /etc/warren/llm.toml
```
