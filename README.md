# warren
AI agent and Slack based security alert management tool

![Image](https://github.com/user-attachments/assets/dc2c9c25-9869-4500-97c2-77ac70ae7b08)

## Concept

## How it works

## Usage

## Options

The following options can be configured via command line flags or environment variables:

### `serve` mode

- `--addr` (WARREN_ADDR): Address to listen on [default: "127.0.0.1:8080"]

### `run` mode

- `--alert, -a` (WARREN_ALERT_PATH): Alert file path [required]
- `--schema, -s` (WARREN_ALERT_SCHEMA): Alert schema definition [required]

### Common

#### Gemini (Required)
- `--gemini-location`: GCP Location for Vertex AI (default: "us-central1") [$WARREN_GEMINI_LOCATION]
- `--gemini-model`: Gemini model (default: "gemini-2.0-flash-exp")
- `--gemini-project-id`: GCP Project ID for Vertex AI [$WARREN_GEMINI_PROJECT_ID]

#### Policy (Required)
- `--policy, -p`: Policy file/dir path [$WARREN_POLICY]

#### Firestore (Required for `serve` mode)
- `--firestore-database-id`: Firestore database ID (default: "(default)") [$WARREN_FIRESTORE_DATABASE_ID]
- `--firestore-project-id`: Firestore project ID [$WARREN_FIRESTORE_PROJECT_ID]

#### Slack (Required for `serve` mode)
- `--slack-channel-name`: Slack channel name, # is not required [$WARREN_SLACK_CHANNEL_NAME]
- `--slack-oauth-token`: Slack OAuth token [$WARREN_SLACK_OAUTH_TOKEN]
- `--slack-signing-secret`: Slack signing secret [$WARREN_SLACK_SIGNING_SECRET]

#### Sentry (Optional for `serve` mode)
- `--sentry-dsn`: Sentry DSN [$WARREN_SENTRY_DSN]
- `--sentry-env`: Sentry environment [$WARREN_SENTRY_ENV]

#### Action
- `--bigquery-config`: BigQuery config file [$WARREN_BIGQUERY_CONFIG]
- `--bigquery-project-id`: BigQuery project ID [$WARREN_BIGQUERY_PROJECT_ID]
- `--otx-api-key`: OTX API key [$WARREN_OTX_API_KEY]
- `--otx-base-url`: OTX API base URL (default: "https://otx.alienvault.com/api/v1") [$WARREN_OTX_BASE_URL]
- `--urlscan-api-key`: URLScan API key [$WARREN_URLSCAN_API_KEY]
- `--urlscan-backoff`: URLScan API backoff duration (default: 3s) [$WARREN_URLSCAN_BACKOFF]
- `--urlscan-base-url`: URLScan API base URL (default: "https://urlscan.io/api/v1") [$WARREN_URLSCAN_BASE_URL]

## License

Apache 2.0 License

