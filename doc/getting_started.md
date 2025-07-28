# Getting Started with Warren in 5 Minutes

Welcome to Warren! This guide will help you get Warren up and running in just 5 minutes using Docker. You'll be able to process security alerts, analyze them with AI, and explore the web interface without any complex setup.

## Prerequisites

Before you begin, ensure you have:

- **Docker**: [Docker Desktop](https://www.docker.com/products/docker-desktop/) (Windows/Mac) or Docker Engine (Linux)
- **Google Cloud Account**: [Create a free account](https://cloud.google.com/free) if you do not have one
- **gcloud CLI**: [Installation guide](https://cloud.google.com/sdk/docs/install)

## Quick Start

### Step 1: Google Cloud Setup (2 minutes)

First, set up Google Cloud authentication and enable the Vertex AI API for Warren's AI capabilities:

```bash
# Set your Google Cloud project ID
export PROJECT_ID="your-project-id"

# Authenticate with Google Cloud
gcloud auth application-default login

# Enable Vertex AI API
gcloud services enable aiplatform.googleapis.com --project=$PROJECT_ID
```

> **Note**: The `gcloud auth application-default login` command will open a browser window for authentication. This creates credentials that Warren will use to access Vertex AI.

### Step 2: Run Warren with Docker (1 minute)

Create an environment file with your configuration:

```bash
# Create your configuration file
cat > warren.env << EOF
# Warren Quick Start Environment Variables

# Required: Google Cloud Project ID for Vertex AI (Gemini)
# You can find this in your Google Cloud Console
WARREN_GEMINI_PROJECT_ID=your-project-id

# Google Cloud location for Vertex AI
# Available regions: https://cloud.google.com/vertex-ai/docs/general/locations
WARREN_GEMINI_LOCATION=us-central1

# Disable authentication for local development
# WARNING: Only use these in development environments
WARREN_NO_AUTHENTICATION=true
WARREN_NO_AUTHORIZATION=true

# Optional: External Tool API Keys
# Get a free API key from https://otx.alienvault.com
# WARREN_OTX_API_KEY=your-otx-api-key

# Optional: Additional threat intelligence services
# WARREN_VIRUSTOTAL_API_KEY=your-virustotal-key
# WARREN_URLSCAN_API_KEY=your-urlscan-key
# WARREN_SHODAN_API_KEY=your-shodan-key
# WARREN_ABUSEIPDB_API_KEY=your-abuseipdb-key

# Optional: Set log level for debugging
# WARREN_LOG_LEVEL=debug
EOF

# Update the PROJECT_ID in the file
sed -i.bak "s/your-project-id/$PROJECT_ID/g" warren.env && rm warren.env.bak

# Run Warren with Docker
docker run -d \
  --name warren \
  -p 8080:8080 \
  -v ~/.config/gcloud:/home/nonroot/.config/gcloud:ro \
  --env-file warren.env \
  ghcr.io/secmon-lab/warren:latest serve

# Check the logs
docker logs warren
```

> **Note**: If the Docker image is not available, you can build it locally:
> ```bash
> git clone https://github.com/secmon-lab/warren.git
> cd warren
> docker build -t warren:local .
> # Then use warren:local instead of ghcr.io/secmon-lab/warren:latest
> ```

Warren should now be running at http://localhost:8080!

### Step 3: Send Your First Alert (1 minute)

Let's send a test security alert to Warren:

```bash
# Send a single alert about a known malicious IP
curl -X POST http://localhost:8080/hooks/alert/raw/test \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Malicious IP Communication Detected",
    "description": "Outbound connection to known C2 server detected",
    "severity": "critical",
    "source_ip": "45.227.255.182",
    "destination_ip": "10.0.1.100",
    "note": "This IP is associated with Cobalt Strike C2 infrastructure"
  }'

# Optional: Send multiple alerts to try the clustering feature
for i in {1..5}; do
  curl -X POST http://localhost:8080/hooks/alert/raw/test \
    -H "Content-Type: application/json" \
    -d "{
      \"title\": \"Suspicious C2 Communication #$i\",
      \"description\": \"Connection to potential C2 infrastructure detected\",
      \"severity\": \"high\",
      \"source_ip\": \"45.227.255.$((180 + i))\",
      \"destination_ip\": \"10.0.1.$((100 + i))\",
      \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
    }"
  sleep 1
done
```

> **Note**: The IP addresses used (45.227.255.x) are real examples of IPs reported as malicious, used here for educational purposes.

### Step 4: Explore the Web UI (1 minute)

Open your browser and navigate to http://localhost:8080

You'll see the Warren dashboard with:
- **Alerts**: View the security alerts you just sent
- **Chat**: Interact with the AI agent for analysis
- **Clusters**: See how similar alerts are grouped (if you sent multiple alerts)

Try these actions:

1. **View Alert Details**: Click on any alert to see its full information

2. **AI Analysis**: Click the "Chat" button and try:
   ```
   Analyze IP 45.227.255.182 using available tools
   ```
   If you added an OTX API key, you'll see real threat intelligence data!

3. **Create a Ticket**: Select one or more alerts and click "Create Ticket" to group them for investigation

`TODO: Screenshot of Warren dashboard showing the alerts list with "Malicious IP Communication Detected" alert visible. The left navigation and main alert grid should be clearly visible.`

`TODO: Screenshot of the Chat interface showing the AI agent analyzing IP 45.227.255.182, with tool execution results visible (especially if OTX integration is configured).`

`TODO: (Optional) If multiple alerts were sent: Screenshot of the Clusters page showing how similar C2 communication alerts are grouped together.`

## What's Next?

Congratulations! You've successfully:
- ✅ Set up Warren with minimal configuration
- ✅ Sent and viewed security alerts
- ✅ Used AI to analyze threats
- ✅ Explored the web interface

### Next Steps

1. **Add Real Security Data**: Connect your security tools to send alerts to Warren
2. **Enable Persistence**: Add Firestore to save your alerts and investigations
3. **Set Up Team Collaboration**: Add Slack integration for team notifications
4. **Configure Security**: Enable authentication and authorization for production use

Ready to move beyond the quick start? Check out the [Installation Guide](./installation.md) to upgrade your setup!

## Troubleshooting

### Common Issues

**Docker container exits immediately**
```bash
# Check the logs for errors
docker logs warren

# Common fix: Ensure gcloud is authenticated
gcloud auth application-default login
```

**"Project not found" error**
```bash
# Verify your project ID is correct
gcloud config get-value project

# Set the correct project
gcloud config set project YOUR_PROJECT_ID
```

**API not enabled error**
```bash
# Enable the required APIs
gcloud services enable aiplatform.googleapis.com
```

### Getting Help

- **Documentation**: [Warren Docs](https://github.com/secmon-lab/warren/tree/main/doc)
- **Issues**: [GitHub Issues](https://github.com/secmon-lab/warren/issues)
- **Discussions**: [GitHub Discussions](https://github.com/secmon-lab/warren/discussions)

## Clean Up

When you're done experimenting:

```bash
# Stop and remove the container
docker stop warren
docker rm warren

# Remove the environment file (optional)
rm warren.env
```

---

*Warren uses Go 1.24+ and integrates with Google Cloud Vertex AI for intelligent security analysis.*