# Jira Tool Configuration

## Overview

The Jira tool enables Warren to query Jira Cloud: listing accessible projects, searching issues with JQL, and fetching the content of one or more issues (with the description rendered to Markdown). Security analysts can correlate alerts with tracked tickets during an investigation.

This is a **read-only** tool — it does not create or modify any Jira data. It is backed by [`github.com/gollem-dev/tools/jira`](https://github.com/gollem-dev/tools).

## Configuration

| Environment Variable | CLI Flag | Default | Required |
|---|---|---|---|
| `WARREN_JIRA_BASE_URL` | `--jira-base-url` | - | Yes |
| `WARREN_JIRA_USER_EMAIL` | `--jira-user-email` | - | Yes |
| `WARREN_JIRA_API_TOKEN` | `--jira-api-token` | - | Yes |

The tool is enabled only when all three values are set; otherwise it is silently skipped. The base URL is your Jira site, e.g. `https://your-domain.atlassian.net`.

## Setup

1. Sign in to Jira Cloud with the account Warren should authenticate as.
2. Go to [**Atlassian account > Security > API tokens**](https://id.atlassian.com/manage-profile/security/api-tokens).
3. Click **Create API token**, give it a label, and copy the generated token.
4. Configure Warren with:
   - `WARREN_JIRA_BASE_URL` — your site URL (`https://<your-domain>.atlassian.net`)
   - `WARREN_JIRA_USER_EMAIL` — the account email
   - `WARREN_JIRA_API_TOKEN` — the token created above

The account's existing permissions bound what the tool can read; no extra scopes are configured on the token itself.

## Available Functions

| Function | Description |
|---|---|
| `jira_list_projects` | List Jira projects accessible to the authenticated account |
| `jira_search_issues` | Search issues using a JQL query |
| `jira_get_issues` | Fetch the content of one or more issues by key/ID (description rendered to Markdown) |

## Authentication

The tool talks to the Jira Cloud REST API v3 directly over HTTP using Basic authentication (`base64(email:apiToken)`). Because each Jira site lives on its own tenant domain, the base URL is a required setting rather than a fixed constant.
