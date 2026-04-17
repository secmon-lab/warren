---
name: create-eval-scenario
description: Create an evaluation scenario folder for warren eval. Use when the user wants to create a new test scenario for agent evaluation, or asks to generate scenario data for eval testing.
allowed-tools: Read, Write, Bash(ls:*), Bash(cat:*), Glob, Grep, Agent
---

# Create Evaluation Scenario

Generate a scenario folder for `warren eval` command. A scenario defines an alert, the world context for mock tool responses, and expected evaluation criteria.

## Step 1: Understand the scenario

Ask the user or infer from context:
- What kind of security alert? (e.g., SSH login from Tor, suspicious file download, IAM privilege escalation)
- What schema? (e.g., gcp_scc, aws_guardduty, falco)
- What should the agent find? (expected outcome)
- Which tools should be called? (trajectory expectations)

## Step 2: Explore available tool specs

Read the tool definitions to understand what tools are available and their arguments:

```bash
ls pkg/tool/
```

For each relevant tool, check its `Specs()` to understand the tool's interface:

```
Read pkg/tool/<tool_name>/tool.go
```

## Step 3: Create scenario folder structure

Create the following structure:

```
<scenario_dir>/
  scenario.yaml
  policy/                          (Rego policies for ingest/enrich/triage)
    ingest.rego
    triage.rego
    enrich.rego                    (optional)
  responses/
    <tool_name>__<args_hash>.json  (for deterministic tools)
  traces/                          (auto-created by eval run)
```

### scenario.yaml template

```yaml
name: <descriptive_slug>
description: "<One-line description>"

alert:
  schema: "<schema_name>"
  data:
    # Alert payload matching the schema

world:
  description: |
    <Detailed description of the scenario world.
    Include specific IPs, hostnames, users, timestamps, and behaviors.
    This is fed to the mock LLM to generate consistent tool responses.>

  tool_hints:
    # Only for tools that need specific guidance
    bigquery: |
      Available tables and their schemas.
      Expected data patterns.
    virustotal: |
      Reputation data for relevant IPs/domains/hashes.

initial_message: "Investigate this alert"

config:
  policy_dir: "policy"                    # Rego policies (required)
  # bigquery_configs:                     # BigQuery dataset/table YAML configs
  #   - "bigquery/config.yaml"
  # bigquery_runbooks:                    # BigQuery SQL runbook files/directories
  #   - "bigquery/runbooks"
  # github_configs:                       # GitHub repository config YAMLs
  #   - "github/config.yaml"
  # user_system_prompt: "prompt/system.md" # User system prompt (markdown)

expectations:
  outcome:
    finding_must_contain:
      - "<keyword1>"
    severity: "<expected_severity>"
    criteria:
      - "<LLM judge criterion 1>"
      - "<LLM judge criterion 2>"

  trajectory:
    must_call:
      - <tool1>
      - <tool2>
    must_not_call:
      - <irrelevant_tool>
    ordered_calls:
      - <tool_to_call_first>
      - <tool_to_call_second>

  efficiency:
    max_total_calls: 20
    max_duplicate_calls: 3
```

## Step 3.5: Create Rego policy files

Create Rego policies in `<scenario_dir>/policy/`. These are evaluated by the pipeline during `HandleAlert`.

Look at existing policy examples in the codebase for the correct format:

```bash
ls pkg/usecase/testdata/*.rego
```

Read a few examples to understand the policy structure:

```
Read pkg/usecase/testdata/ingest_test.rego
Read pkg/usecase/testdata/triage_basic.rego
```

At minimum, create:

1. **Ingest policy** (`policy/ingest.rego`) — transforms raw alert data into Warren Alert objects
   - Package: `data.ingest.<schema_name>` (must match `alert.schema` in scenario.yaml)
   - Input: the alert data from `alert.data`
   - Output: array of alert objects with title, description
   - **CRITICAL: Always pass through the full raw data via `"data": input`** so the agent has access to all IoCs (IPs, hashes, domains, etc.). Without this, the agent will lack concrete indicators and ask questions instead of investigating.
   
   Example:
   ```rego
   alerts contains {
       "title": title,
       "description": description,
       "data": input,
   } if {
       # ... conditions and formatting ...
   }
   ```

2. **Triage policy** (`policy/triage.rego`) — determines publish type and metadata
   - Package: `data.triage`
   - Input: alert + enrich results
   - Output: publish type (alert/notice/discard), title, description, severity

3. **Enrich policy** (`policy/enrich.rego`, optional) — defines enrichment tasks
   - Package: `data.enrich`
   - Only needed if you want the pipeline to run enrichment before triage

## Step 4: Create pre-defined response files

For deterministic tools (VT, OTX, Shodan, etc.), create response files in `responses/`:

Each file is a JSON object:
```json
{
  "tool_name": "<tool_name>",
  "args": { <the exact args the tool would be called with> },
  "response": { <realistic response data> }
}
```

File naming: `<tool_name>__<first_8_chars_of_sha256_of_normalized_args>.json`

To compute the hash, normalize args by:
1. Sort map keys recursively
2. Exclude nil values
3. JSON marshal → SHA256 → first 8 hex chars

For non-deterministic tools (BigQuery, Slack search), do NOT create response files. The MockAgent will generate them at runtime using the world description.

## Step 5: Validate the scenario (MANDATORY)

After creating all files, **ALWAYS** run the warren validate command to check the scenario:

```bash
go run . eval validate -s <scenario_dir>
```

This validates:
- Required fields (name, alert.schema, alert.data, initial_message, world.description)
- Trajectory consistency (must_call vs must_not_call overlap, ordered_calls subset)
- Efficiency thresholds are non-negative
- Response files: valid JSON, tool_name present, response present, filename matches tool_name prefix

**Do NOT report the scenario as complete until validation passes.** If validation fails, fix all reported errors and re-run.

## Step 6: Look up the latest Gemini model (MANDATORY)

Before generating the run command, **search the web** for the latest available Gemini model on Vertex AI:

```
WebSearch "Vertex AI Gemini latest model ID site:cloud.google.com"
```

Check the official docs to find the current stable/preview model name (e.g., `gemini-2.5-flash`, `gemini-3-flash-preview`). Also determine the correct `--gemini-location` — newer preview models may only be available in `global`, not regional endpoints like `us-central1`.

## Step 7: Show the run command (MANDATORY)

After validation passes, **ALWAYS** present the exact command to run the scenario. Read the scenario's `config` section and generate CLI flags pointing to the scenario's config files.

Build the command by mapping `config` fields to CLI flags:

| config field | CLI flag |
|---|---|
| `policy_dir` | `--policy <scenario_dir>/<value>` |
| `bigquery_configs` | `--bigquery-config <scenario_dir>/<value>` (repeat per entry) |
| `bigquery_runbooks` | `--bigquery-runbook-path <scenario_dir>/<value>` (repeat per entry) |
| `github_configs` | `--github-app-config <scenario_dir>/<value>` (repeat per entry) |
| `user_system_prompt` | `--user-system-prompt <scenario_dir>/<value>` |

Use the model name and location found in Step 6 for `--gemini-model` and `--gemini-location`.

Example output:

```bash
go run . eval run -s <scenario_dir> \
  --policy <scenario_dir>/policy \
  --bigquery-config <scenario_dir>/bigquery/config.yaml \
  --gemini-project-id <PROJECT_ID> \
  --gemini-model <LATEST_MODEL> \
  --gemini-location <LOCATION> \
  -o report.md -f markdown
```

Only include flags for config fields that are actually set in the scenario. Replace `<PROJECT_ID>` with the user's GCP project ID.

## Tips

- The `world.description` is the most important field. It defines the "ground truth" that all mock responses must be consistent with. Be specific about IPs, timestamps, usernames, and behaviors.
- `tool_hints` are optional but help the mock LLM generate better responses for complex tools like BigQuery (include table schemas).
- Start with 2-3 pre-defined response files for the most predictable tools, and let the MockAgent handle the rest.
- `expectations.criteria` are evaluated by LLM-as-Judge. Write them as specific, verifiable statements about what the agent should conclude.

## Critical: Avoid agent questions

In eval CLI mode, the agent cannot ask questions to the user (no presenter available). If the agent asks a question, the eval run will fail with `"question requires a presenter but none is available"`.

**To prevent this, the `alert.data` MUST contain specific, concrete IoCs** (Indicators of Compromise). The agent asks questions when it lacks enough information to investigate. Include:

- Specific IP addresses, domain names, file hashes in the alert payload
- Process names, container IDs, user accounts involved
- Timestamps and event categories (e.g., MITRE ATT&CK technique IDs)
- Any detection-specific fields the schema would normally contain

**Bad** (too vague — agent will ask for details):
```yaml
alert:
  data:
    finding:
      category: "SUSPICIOUS_PROCESS"
      description: "Suspicious process detected in GKE pod"
```

**Good** (specific — agent can investigate immediately):
```yaml
alert:
  data:
    finding:
      category: "Malware: Cryptomining Bad IP"
      sourceProperties:
        sourceIp: "185.122.204.197"
        destIp: "10.128.0.45"
        processName: "xmrig"
        sha256: "440784338980838b939e663675c2e17e3f81df68853b8214b62f84090940428d"
        podName: "web-frontend-7d8f9b6c4-x2j9k"
        containerName: "nginx"
        projectId: "my-project"
```
