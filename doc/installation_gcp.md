# Google Cloud Setup for Warren

This guide provides detailed instructions for setting up Warren on Google Cloud, including all required services, permissions, and deployment configurations.

## Prerequisites

- Google Cloud account with billing enabled
- `gcloud` CLI installed and authenticated
- Project Owner or Editor role (for initial setup)
- Basic familiarity with Google Cloud services

## 1. Project Setup

### 1.1. Create and Configure Project

```bash
# Set your project ID - use lowercase letters, numbers, and hyphens
export PROJECT_ID="your-warren-project"
export REGION="us-central1"  # Choose region close to your users

# Create a new project (skip if using existing project)
gcloud projects create $PROJECT_ID --name="Warren Security Platform"

# Set the project as default
gcloud config set project $PROJECT_ID

# Link billing account (required for Cloud Run and other services)
# First, list your billing accounts
gcloud billing accounts list

# Then link the billing account to your project
gcloud billing projects link $PROJECT_ID \
  --billing-account=BILLING_ACCOUNT_ID
```

### 1.2. Enable Required APIs

```bash
# Enable all required Google Cloud APIs
gcloud services enable \
    run.googleapis.com \
    secretmanager.googleapis.com \
    firestore.googleapis.com \
    storage-component.googleapis.com \
    aiplatform.googleapis.com \
    artifactregistry.googleapis.com \
    cloudbuild.googleapis.com \
    cloudresourcemanager.googleapis.com \
    iam.googleapis.com

# Verify APIs are enabled
gcloud services list --enabled --filter="name:run.googleapis.com OR name:firestore.googleapis.com"
```

## 2. Firestore Configuration

Warren uses Firestore for storing alerts, tickets, and metadata.

### 2.1. Create Firestore Database

```bash
# Create Firestore database in Native mode
# Note: You can only have one Firestore database per project
gcloud firestore databases create \
    --region=$REGION \
    --type=firestore-native

# The database will be created with the ID "(default)"
```

### 2.2. Configure Firestore Indexes

Warren requires specific indexes for vector search and query performance. These indexes are essential for:
- Alert clustering using embedding vectors (256-dimensional)
- Efficient ticket and alert queries
- Status-based ticket filtering

#### Option 1: Using Warren Migrate Command (Recommended)

```bash
# Clone or download the Warren repository
git clone https://github.com/secmon-lab/warren.git
cd warren

# Dry run to see what indexes will be created
go run . migrate --dry-run \
  --firestore-project-id=$PROJECT_ID \
  --firestore-database-id="(default)"

# Create all required indexes
go run . migrate \
  --firestore-project-id=$PROJECT_ID \
  --firestore-database-id="(default)"

# Or build and use the binary
go build -o warren
./warren migrate \
  --firestore-project-id=$PROJECT_ID \
  --firestore-database-id="(default)"
```

#### Option 2: Manual Index Creation

If you prefer to create indexes manually using gcloud CLI:

```bash
# Set your project and database
export PROJECT_ID="your-project-id"
export DATABASE_ID="(default)"

# Alerts collection indexes
gcloud firestore indexes composite create \
  --project=$PROJECT_ID \
  --database=$DATABASE_ID \
  --collection-group=alerts \
  --query-scope=COLLECTION \
  --field-config=field-path=Embedding,vector-config='{"dimension":256,"flat":{}}'

gcloud firestore indexes composite create \
  --project=$PROJECT_ID \
  --database=$DATABASE_ID \
  --collection-group=alerts \
  --query-scope=COLLECTION \
  --field-config=order=DESCENDING,field-path=CreatedAt \
  --field-config=field-path=Embedding,vector-config='{"dimension":256,"flat":{}}'

# Tickets collection indexes
gcloud firestore indexes composite create \
  --project=$PROJECT_ID \
  --database=$DATABASE_ID \
  --collection-group=tickets \
  --query-scope=COLLECTION \
  --field-config=field-path=Embedding,vector-config='{"dimension":256,"flat":{}}'

gcloud firestore indexes composite create \
  --project=$PROJECT_ID \
  --database=$DATABASE_ID \
  --collection-group=tickets \
  --query-scope=COLLECTION \
  --field-config=order=DESCENDING,field-path=CreatedAt \
  --field-config=field-path=Embedding,vector-config='{"dimension":256,"flat":{}}'

gcloud firestore indexes composite create \
  --project=$PROJECT_ID \
  --database=$DATABASE_ID \
  --collection-group=tickets \
  --query-scope=COLLECTION \
  --field-config=order=ASCENDING,field-path=Status \
  --field-config=order=DESCENDING,field-path=CreatedAt

# Lists collection indexes
gcloud firestore indexes composite create \
  --project=$PROJECT_ID \
  --database=$DATABASE_ID \
  --collection-group=lists \
  --query-scope=COLLECTION \
  --field-config=field-path=Embedding,vector-config='{"dimension":256,"flat":{}}'

gcloud firestore indexes composite create \
  --project=$PROJECT_ID \
  --database=$DATABASE_ID \
  --collection-group=lists \
  --query-scope=COLLECTION \
  --field-config=order=DESCENDING,field-path=CreatedAt \
  --field-config=field-path=Embedding,vector-config='{"dimension":256,"flat":{}}'

# Memories subcollection index (COLLECTION scope)
gcloud firestore indexes composite create \
  --project=$PROJECT_ID \
  --database=$DATABASE_ID \
  --collection-group=memories \
  --query-scope=COLLECTION_GROUP \
  --field-config=field-path=QueryEmbedding,vector-config='{"dimension":256,"flat":{}}'
```

#### Index Details

The following indexes are created:

| Collection | Index Type | Fields | Purpose |
|------------|------------|--------|---------|
| alerts, tickets, lists | Vector | Embedding (256d) | Alert clustering & similarity search |
| alerts, tickets, lists | Composite | CreatedAt DESC + Embedding | Time-based vector searches |
| tickets | Composite | Status ASC + CreatedAt DESC | Dashboard filtering |
| execution_memories, ticket_memories | Vector | QueryEmbedding (256d) | Agent memory semantic search |
| execution_memories, ticket_memories | Composite | CreatedAt DESC + QueryEmbedding | Time-based memory searches |
| memories (subcollection) | Vector | QueryEmbedding (256d) | Agent-specific memory search |

> **Note**: 
> - Index creation may take 5-10 minutes
> - Monitor progress: [Firestore Console](https://console.cloud.google.com/firestore/databases/-default-/indexes)
> - Vector indexes require Firestore in Native mode (not Datastore mode)

#### Troubleshooting

- **Permission errors**: Ensure `roles/datastore.indexAdmin` role
- **Index already exists**: Safe to ignore, the migrate command is idempotent

## 3. Cloud Storage Setup

Warren uses Cloud Storage for file attachments and artifacts.

### 3.1. Create Storage Bucket

```bash
# Create bucket with uniform bucket-level access
export STORAGE_BUCKET="${PROJECT_ID}-warren-storage"

gsutil mb -p $PROJECT_ID \
  -c standard \
  -l $REGION \
  -b on \
  gs://$STORAGE_BUCKET

# Set lifecycle rules to auto-delete old temporary files (optional)
cat > lifecycle.json << EOF
{
  "lifecycle": {
    "rule": [
      {
        "action": {"type": "Delete"},
        "condition": {
          "age": 30,
          "matchesPrefix": ["temp/"]
        }
      }
    ]
  }
}
EOF

gsutil lifecycle set lifecycle.json gs://$STORAGE_BUCKET
```

## 4. Vertex AI Configuration

Warren uses Vertex AI's Gemini model for AI-powered analysis.

### 4.1. Enable Vertex AI

```bash
# Vertex AI API should already be enabled, but verify:
gcloud services enable aiplatform.googleapis.com

# Set default location for Vertex AI
gcloud config set ai_platform/region $REGION
```

### 4.2. Grant Model Access

The service account will need access to use Gemini models. This is configured later with the service account setup.

## 5. Secret Manager Configuration

Warren stores sensitive credentials in Secret Manager.

### 5.1. Create Secrets

```bash
# Create secret for Slack OAuth token
echo -n "xoxb-your-slack-bot-token" | \
  gcloud secrets create slack-oauth-token \
    --data-file=- \
    --replication-policy="automatic"

# Create secret for Slack signing secret
echo -n "your-slack-signing-secret" | \
  gcloud secrets create slack-signing-secret \
    --data-file=- \
    --replication-policy="automatic"

# Create secrets for Slack OAuth (Web UI)
echo -n "your-slack-client-id" | \
  gcloud secrets create slack-client-id \
    --data-file=- \
    --replication-policy="automatic"

echo -n "your-slack-client-secret" | \
  gcloud secrets create slack-client-secret \
    --data-file=- \
    --replication-policy="automatic"

# Optional: Create secrets for external service API keys
# Only create these if you have the API keys
echo -n "your-virustotal-api-key" | \
  gcloud secrets create vt-api-key \
    --data-file=- \
    --replication-policy="automatic"

echo -n "your-otx-api-key" | \
  gcloud secrets create otx-api-key \
    --data-file=- \
    --replication-policy="automatic"

echo -n "your-urlscan-api-key" | \
  gcloud secrets create urlscan-api-key \
    --data-file=- \
    --replication-policy="automatic"
```

### 5.2. List and Verify Secrets

```bash
# List all secrets
gcloud secrets list

# View secret metadata (not the value)
gcloud secrets describe slack-oauth-token
```

## 6. Service Account Setup

Create a dedicated service account for Warren with minimal required permissions.

### 6.1. Create Service Account

```bash
# Create service account
gcloud iam service-accounts create warren-service \
    --description="Warren Security Bot Service Account" \
    --display-name="Warren Service"

# Set service account email variable
export SERVICE_ACCOUNT="warren-service@${PROJECT_ID}.iam.gserviceaccount.com"
```

### 6.2. Grant Required Permissions

```bash
# Firestore permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/datastore.user"

# Cloud Storage permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/storage.objectAdmin" \
    --condition="expression=resource.name.startsWith('projects/_/buckets/${STORAGE_BUCKET}'),title=warren-bucket-only"

# Secret Manager permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/secretmanager.secretAccessor"

# Vertex AI permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/aiplatform.user"

# Cloud Run invoker (if using authenticated endpoints)
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/run.invoker"

# Optional: BigQuery permissions (if using BigQuery integration)
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/bigquery.jobUser"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/bigquery.dataViewer"
```

## 7. Artifact Registry Setup

If you plan to build custom Docker images:

### 7.1. Create Docker Repository

```bash
# Create repository for Warren images
gcloud artifacts repositories create warren \
    --repository-format=docker \
    --location=$REGION \
    --description="Warren Docker images"

# Configure Docker authentication
gcloud auth configure-docker ${REGION}-docker.pkg.dev
```

## 8. Deploy Warren to Cloud Run

> **Configuration Note**: For a complete list of all available environment variables and configuration options, see the [Configuration Reference](./configuration.md).

### 8.1. Using Pre-built Image

For quick deployment using the official Warren image:

```bash
# Deploy Warren service
gcloud run deploy warren \
    --image=ghcr.io/secmon-lab/warren:latest \
    --region=$REGION \
    --platform=managed \
    --service-account=$SERVICE_ACCOUNT \
    --allow-unauthenticated \
    --memory=2Gi \
    --cpu=2 \
    --concurrency=80 \
    --max-instances=10 \
    --timeout=300 \
    --port=8080 \
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

### 8.2. Using Custom Image

If you built a custom image with your policies:

```bash
# Tag for your custom image
export IMAGE_TAG="${REGION}-docker.pkg.dev/${PROJECT_ID}/warren/warren:latest"

# Build and push your custom image (see Advanced Configuration guide)
docker build -t $IMAGE_TAG .
docker push $IMAGE_TAG

# Deploy with custom image
gcloud run deploy warren \
    --image=$IMAGE_TAG \
    --region=$REGION \
    # ... (same parameters as above)
```

### 8.3. Get Service URL

```bash
# Get the service URL
export SERVICE_URL=$(gcloud run services describe warren \
    --region=$REGION \
    --format='value(status.url)')

echo "Warren is deployed at: $SERVICE_URL"

# This URL is needed for:
# - Slack event subscriptions
# - OAuth redirect URLs  
# - Web UI access
```

## 9. Configure Frontend URL

If you want the Slack messages to include links to the Web UI:

```bash
# Update the service with frontend URL
gcloud run services update warren \
    --region=$REGION \
    --set-env-vars="WARREN_FRONTEND_URL=${SERVICE_URL}"
```

## 10. Optional: BigQuery Setup

If you plan to use BigQuery for analytics:

### 10.1. Create Dataset

```bash
# Create BigQuery dataset
bq mk --dataset \
    --location=$REGION \
    --description="Warren security analytics" \
    ${PROJECT_ID}:warren_analytics

# Grant service account access
bq add-iam-policy-binding \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/bigquery.dataViewer" \
    ${PROJECT_ID}:warren_analytics
```

> **Important**: BigQuery integration requires you to populate your own security data. Warren does not automatically create or populate BigQuery tables. You'll need to:
> 1. Create your own tables with security log data
> 2. Define appropriate schemas for your data
> 3. Set up data ingestion pipelines (e.g., from Cloud Logging, external SIEM)
> 4. Write SQL queries that the AI agent can use for investigations
>
> Example: To analyze AWS CloudTrail events, you would need to export CloudTrail logs to BigQuery and create appropriate views for the AI agent to query.

### 10.2. Update Cloud Run Configuration

```bash
# Add BigQuery environment variables
gcloud run services update warren \
    --region=$REGION \
    --set-env-vars="WARREN_BIGQUERY_PROJECT_ID=${PROJECT_ID}" \
    --set-env-vars="WARREN_BIGQUERY_DATASET_ID=warren_analytics"
```

## 11. Monitoring and Logging

### 11.1. View Logs

```bash
# View recent logs
gcloud logs read "resource.type=cloud_run_revision AND resource.labels.service_name=warren" \
    --limit=50 \
    --format="table(timestamp,textPayload)"

# Stream logs in real-time
gcloud alpha run services logs tail warren --region=$REGION
```

### 11.2. Set Up Alerts (Optional)

```bash
# Create alert policy for high error rate
gcloud alpha monitoring policies create \
    --display-name="Warren High Error Rate" \
    --condition-display-name="Error rate > 5%" \
    --condition-filter='resource.type="cloud_run_revision" AND resource.labels.service_name="warren" AND severity="ERROR"' \
    --condition-threshold-value=5 \
    --condition-threshold-duration=60s
```

## 12. Cost Optimization

### Development/Testing Configuration

For minimal costs during development:

```bash
# Scale down to zero when not in use
gcloud run services update warren \
    --region=$REGION \
    --min-instances=0 \
    --max-instances=2 \
    --cpu=1 \
    --memory=512Mi
```

### Cost Tips

- Use `min-instances=0` to avoid charges when idle
- Firestore free tier includes 50,000 reads/20,000 writes per day
- Vertex AI charges per request - monitor usage
- Cloud Storage has minimal costs for small amounts of data

## Verification

1. **Check Service Status**:
   ```bash
   # Use GraphQL endpoint to verify service is running
   curl -X POST $SERVICE_URL/graphql \
     -H "Content-Type: application/json" \
     -d '{"query": "{ __typename }"}'
   ```

2. **Verify Firestore Connection**:
   Check Cloud Run logs for successful Firestore initialization

3. **Test Secret Access**:
   Logs should show successful loading of Slack credentials

## Troubleshooting

### Common Issues

1. **"Permission denied" errors**:
   - Verify service account has all required roles
   - Check that secrets exist and are accessible

2. **"Vertex AI quota exceeded"**:
   - Check quotas: `gcloud compute project-info describe`
   - Request quota increase if needed

3. **"Failed to start container"**:
   - Check logs for specific error
   - Verify all environment variables are set
   - Ensure secrets are properly formatted

4. **Firestore errors**:
   - Confirm database exists in correct region
   - Check that indexes are deployed
   - Verify service account permissions

### Debug Commands

```bash
# Describe service configuration
gcloud run services describe warren --region=$REGION

# Check service account permissions
gcloud projects get-iam-policy $PROJECT_ID \
    --flatten="bindings[].members" \
    --filter="bindings.members:serviceAccount:$SERVICE_ACCOUNT"

# Test secret access
gcloud secrets versions access latest --secret=slack-oauth-token
```

## Next Steps

1. Complete [Slack Configuration](./installation_slack.md)
2. Set up alert policies (see [Policy Guide](./policy.md))
3. Configure external tool integrations
4. Test the complete alert pipeline