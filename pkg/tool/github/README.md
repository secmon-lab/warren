# GitHub

Search code, issues, and pull requests, list commit history, and get file blame across any GitHub repository reachable by the configured GitHub App installation.

## Configuration

| Environment Variable | CLI Flag | Required | Description |
|---|---|---|---|
| `WARREN_GITHUB_APP_ID` | `--github-app-id` | Yes | GitHub App ID |
| `WARREN_GITHUB_APP_INSTALLATION_ID` | `--github-app-installation-id` | Yes | GitHub App Installation ID |
| `WARREN_GITHUB_APP_PRIVATE_KEY` | `--github-app-private-key` | Yes | GitHub App private key (PEM format) |
| `WARREN_GITHUB_APP_CONFIG` | `--github-app-config` | No | YAML file(s) listing recommended repositories. These are presented to the LLM as starting-point hints only and do **not** act as an access allowlist — any repository the App installation can reach is callable. |

## Available Functions

| Function | Description |
|---|---|
| `github_code_search` | Search for code. Supports filters: `language`, `path`, `filename`, `repo_filter` (comma-separated `owner/name` list, optional) |
| `github_issue_search` | Search issues and pull requests. Supports filters: `state`, `labels`, `author`, `type`, `repo_filter` (comma-separated `owner/name` list, optional) |
| `github_get_content` | Get file content from a specific repository by owner, repo, path, and ref |
| `github_list_commits` | List commits for a repository. Supports filters: `sha`, `path`, `author`, `per_page`, `page` |
| `github_get_blame` | Get git blame for a file, showing which commit last modified each line. Uses GraphQL API |

The access boundary for every function is the GitHub App installation scope: configure the App to grant access to exactly the repositories you want investigatable. The `WARREN_GITHUB_APP_CONFIG` YAML is used only to populate the LLM-facing hint section in the system prompt (and to look up a `default_branch` for `github_get_blame`).

## Setup

### 1. Create a GitHub App

1. Go to **Settings > Developer settings > GitHub Apps > New GitHub App**
2. Configure the app:
   - **App name**: A descriptive name (e.g., `warren-code-search`)
   - **Homepage URL**: Your organization URL
   - **Webhook**: Uncheck "Active" (not needed)
3. Set permissions:
   - **Repository permissions > Contents**: Read-only
   - **Repository permissions > Issues**: Read-only
4. Click **Create GitHub App**
5. Note the **App ID**

### 2. Generate a Private Key

1. In the app settings, scroll to **Private keys**
2. Click **Generate a private key**
3. Save the downloaded `.pem` file securely

### 3. Install the App

1. Click **Install App** in the left sidebar
2. Select your organization
3. Choose repositories to grant access to
4. Note the **Installation ID** from the URL after installation

### 4. Configure Warren

```bash
export WARREN_GITHUB_APP_ID="123456"
export WARREN_GITHUB_APP_INSTALLATION_ID="78901234"
export WARREN_GITHUB_APP_PRIVATE_KEY="$(cat /path/to/private-key.pem)"
```

The tool is automatically enabled when all required credentials are configured.
