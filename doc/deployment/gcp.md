# Google Cloud Deployment

This guide covers deploying Warren on Google Cloud Platform using Cloud Run, Firestore, and Vertex AI.

## Prerequisites

- Google Cloud account with billing enabled
- `gcloud` CLI installed and authenticated
- Project Owner or Editor role (for initial setup)

## 1. Project Setup

```bash
export PROJECT_ID="your-warren-project"
export REGION="us-central1"

# Create project (skip if using existing)
gcloud projects create $PROJECT_ID --name="Warren Security Platform"
gcloud config set project $PROJECT_ID

# Enable required APIs
gcloud services enable \
    run.googleapis.com \
    secretmanager.googleapis.com \
    firestore.googleapis.com \
    storage-component.googleapis.com \
    aiplatform.googleapis.com \
    artifactregistry.googleapis.com \
    cloudbuild.googleapis.com \
    iam.googleapis.com
```

## 2. Firestore Setup

```bash
# Create Firestore database in Native mode
gcloud firestore databases create \
    --region=$REGION \
    --type=firestore-native
```

### Create Indexes

Warren requires vector indexes for similarity search. Use the migrate command (recommended):

```bash
# Dry run to preview
go run . migrate --dry-run \
  --firestore-project-id=$PROJECT_ID

# Create all required indexes
go run . migrate \
  --firestore-project-id=$PROJECT_ID
```

Alternatively, create indexes manually using `gcloud firestore indexes composite create` commands. Run `warren migrate --dry-run` to see which indexes are needed.

> Index creation takes 5-10 minutes. Monitor at [Firestore Console](https://console.cloud.google.com/firestore/databases/-default-/indexes).

## 3. Cloud Storage

```bash
export STORAGE_BUCKET="${PROJECT_ID}-warren-storage"

gsutil mb -p $PROJECT_ID \
  -c standard -l $REGION -b on \
  gs://$STORAGE_BUCKET
```

## 4. Secret Manager

Store sensitive credentials:

```bash
echo -n "xoxb-your-slack-bot-token" | \
  gcloud secrets create slack-oauth-token --data-file=- --replication-policy="automatic"

echo -n "your-slack-signing-secret" | \
  gcloud secrets create slack-signing-secret --data-file=- --replication-policy="automatic"

echo -n "your-slack-client-id" | \
  gcloud secrets create slack-client-id --data-file=- --replication-policy="automatic"

echo -n "your-slack-client-secret" | \
  gcloud secrets create slack-client-secret --data-file=- --replication-policy="automatic"

# Optional: API keys for threat intelligence tools
echo -n "your-vt-key" | gcloud secrets create vt-api-key --data-file=- --replication-policy="automatic"
```

## 5. Service Account

```bash
gcloud iam service-accounts create warren-service \
    --description="Warren Security Bot" \
    --display-name="Warren Service"

export SERVICE_ACCOUNT="warren-service@${PROJECT_ID}.iam.gserviceaccount.com"

# Grant required permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/datastore.user"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/secretmanager.secretAccessor"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/aiplatform.user"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/storage.objectAdmin"

# Optional: BigQuery permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/bigquery.jobUser"
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/bigquery.dataViewer"
```

## 6. Deploy to Cloud Run

```bash
gcloud run deploy warren \
    --image=ghcr.io/secmon-lab/warren:latest \
    --region=$REGION \
    --platform=managed \
    --service-account=$SERVICE_ACCOUNT \
    --allow-unauthenticated \
    --memory=2Gi --cpu=2 \
    --concurrency=80 --max-instances=10 \
    --timeout=300 --port=8080 \
    --set-env-vars="WARREN_GEMINI_PROJECT_ID=${PROJECT_ID}" \
    --set-env-vars="WARREN_GEMINI_LOCATION=${REGION}" \
    --set-env-vars="WARREN_FIRESTORE_PROJECT_ID=${PROJECT_ID}" \
    --set-env-vars="WARREN_STORAGE_BUCKET=${STORAGE_BUCKET}" \
    --set-env-vars="WARREN_STORAGE_PROJECT_ID=${PROJECT_ID}" \
    --set-env-vars="WARREN_SLACK_CHANNEL_NAME=security-alerts" \
    --set-secrets="WARREN_SLACK_OAUTH_TOKEN=slack-oauth-token:latest" \
    --set-secrets="WARREN_SLACK_SIGNING_SECRET=slack-signing-secret:latest" \
    --set-secrets="WARREN_SLACK_CLIENT_ID=slack-client-id:latest" \
    --set-secrets="WARREN_SLACK_CLIENT_SECRET=slack-client-secret:latest"
```

Get the service URL:

```bash
export SERVICE_URL=$(gcloud run services describe warren \
    --region=$REGION --format='value(status.url)')
echo "Warren is deployed at: $SERVICE_URL"

# Set frontend URL for Slack links
gcloud run services update warren \
    --region=$REGION \
    --set-env-vars="WARREN_FRONTEND_URL=${SERVICE_URL}"
```

## 7. Verification

```bash
# Check service is running
curl -X POST $SERVICE_URL/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ __typename }"}'
```

## Cost Optimization

For development/testing:

```bash
gcloud run services update warren \
    --region=$REGION \
    --min-instances=0 --max-instances=2 \
    --cpu=1 --memory=512Mi
```

## Troubleshooting

- **Permission denied**: Verify service account roles
- **Vertex AI quota exceeded**: Check and request quota increase
- **Firestore errors**: Confirm database exists and indexes are deployed

For all configuration options, see [Configuration Reference](../reference/configuration.md).

## Next Steps

1. [Configure Slack integration](./slack.md)
2. [Set up alert policies](../operation/policy.md)
3. [Configure threat intelligence tools](../reference/configuration.md#threat-intelligence-tools)
