# Firestore Index Tool

This tool creates Firestore indexes for the Warren application.

## Prerequisites

- Go 1.24+
- gcloud CLI installed and properly authenticated

## Installation

### Install from Warren Project Root (Recommended)

```bash
# From Warren project root
go install github.com/secmon-lab/warren/resources/firestore_index@latest
```

This will install the `firestore_index` binary to your `$GOPATH/bin` (or `$GOBIN` if set), making it available from anywhere in your system.

### Alternative: Local Build

```bash
cd resources/firestore_index
go build -o firestore_index
```

## Usage

### Commands

#### Dry Run (Check indexes without creating them)

```bash
# If installed with go install
firestore_index create --dry-run \
  --project=your-project-id \
  --database=your-database-id
```

#### Create Indexes

```bash
# If installed with go install
firestore_index create \
  --project=your-project-id \
  --database=your-database-id

# If built locally
./firestore_index create \
  --project=your-project-id \
  --database=your-database-id
```

### Environment Variables

The tool supports the same environment variables as Warren:

- `WARREN_FIRESTORE_PROJECT_ID`: Firestore project ID
- `WARREN_FIRESTORE_DATABASE_ID`: Firestore database ID (default: "(default)")

Using environment variables:

```bash
export WARREN_FIRESTORE_PROJECT_ID=your-project-id
export WARREN_FIRESTORE_DATABASE_ID=your-database-id
firestore_index create --dry-run
```

### Command Line Options

- `--project, -p`: Firestore project ID (required)
- `--database, -d`: Firestore database ID (default: "(default)")
- `--dry-run`: Check required indexes without creating them

## Created Indexes

This tool creates indexes for the following collections:

- `alerts`
- `tickets`
- `lists`

For each collection, the following indexes are created:

1. **Embedding Vector Index**
   - Vector dimension: 256
   - Configuration: flat

2. **CreatedAt + Embedding Composite Index**
   - CreatedAt: descending
   - Embedding: vector (dimension: 256)

3. **Status + CreatedAt Composite Index** (`tickets` collection only)
   - Status: ascending
   - CreatedAt: descending

## Alternative: Running with go run

You can also run the tool directly without installing:

```bash
# From Warren project root
go run ./resources/firestore_index create --dry-run --project=your-project-id
```

## Troubleshooting

### Command Not Found After Installation

If you get "command not found" after `go install`, ensure your Go bin directory is in your PATH:

```bash
# Add to your shell profile (.bashrc, .zshrc, etc.)
export PATH=$PATH:$(go env GOPATH)/bin

# Or if you have GOBIN set
export PATH=$PATH:$(go env GOBIN)
```

### gcloud CLI Authentication Errors

```bash
gcloud auth login
gcloud config set project your-project-id
```

### Permission Errors

Firestore management permissions are required. The following roles are needed:
- `roles/datastore.indexAdmin`
- `roles/datastore.viewer`

### Build/Install Errors

If you encounter import errors, ensure you're running from the Warren project root:

```bash
# From Warren project root
go mod tidy
``` 