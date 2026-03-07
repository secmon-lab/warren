---
paths:
  - "pkg/tool/**"
  - "pkg/agents/**"
  - "doc/**"
  - "README.md"
---

# Documentation Update Rules

When adding, removing, or modifying tools or sub-agents, the following documentation MUST be updated:

## Adding a New Tool (`pkg/tool/<name>/`)

1. **Create `pkg/tool/<name>/README.md`** with unified format:
   - Overview (1-2 sentences)
   - Configuration table (env vars, CLI flags, defaults)
   - Available Functions table (function name, description)
   - Setup instructions (where to get API key, required permissions)

2. **Update `README.md`** (project root):
   - Add to the appropriate category under "## Integrations" (Threat Intelligence Tools / Code & Device Tools)
   - Include link to the tool's README

3. **Update `doc/operation/alert-investigation.md`**:
   - Add to the appropriate tools table under "## Available Tools"

4. **Update `doc/reference/configuration.md`**:
   - Add env vars and CLI flags to the appropriate section

## Adding a New Sub-Agent (`pkg/agents/<name>/`)

1. **Create `pkg/agents/<name>/README.md`** following the sub-agent development rules

2. **Update `README.md`** (project root):
   - Add to "### Sub-Agents" under "## Integrations"

3. **Update `doc/operation/alert-investigation.md`**:
   - Add to "## Sub-Agents" table

4. **Update `doc/reference/configuration.md`**:
   - Add env vars under "## Sub-Agent Configuration"

5. **Register in `pkg/agents/agents.go`**

## Removing a Tool or Sub-Agent

Reverse all of the above: remove README, remove entries from root README, alert-investigation.md, and configuration.md.

## General Rules

- All documentation claims MUST have corresponding code. Never write descriptions of features that don't exist.
- Env var names and CLI flag names must match the actual code in `pkg/cli/config/`.
- Function names in docs must match the `Name` field in the tool's `Specs()` method.
- Default values in docs must match the `Value` field in CLI flag definitions.
