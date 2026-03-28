#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
FRONTEND_DIR="$PROJECT_ROOT/frontend"

# Cleanup on exit
BACKEND_PID=""
WARREN_BIN=""
cleanup() {
  if [ -n "$BACKEND_PID" ]; then
    echo "Stopping backend server (PID: $BACKEND_PID)..."
    kill "$BACKEND_PID" 2>/dev/null || true
    wait "$BACKEND_PID" 2>/dev/null || true
  fi
  if [ -n "$WARREN_BIN" ] && [ -f "$WARREN_BIN" ]; then
    rm -f "$WARREN_BIN"
  fi
}
trap cleanup EXIT

# Build frontend
echo "==> Building frontend..."
cd "$FRONTEND_DIR"
pnpm install
pnpm run build

# Find available port
find_port() {
  local port
  while true; do
    port=$((RANDOM % 10000 + 20000))
    if ! lsof -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
      echo "$port"
      return
    fi
  done
}

E2E_PORT=$(find_port)
echo "==> Using port $E2E_PORT"

# Build and start backend server
echo "==> Building backend..."
cd "$PROJECT_ROOT"
WARREN_BIN=$(mktemp "${TMPDIR:-/tmp}/warren-e2e.XXXXXX")
go build -o "$WARREN_BIN" .

echo "==> Starting backend server..."
MAX_RETRIES=3
for i in $(seq 1 $MAX_RETRIES); do
  "$WARREN_BIN" serve \
    --addr="127.0.0.1:$E2E_PORT" \
    --no-authn \
    --no-authz \
    --enable-graphql \
    --disable-llm \
    --log-level=error &
  BACKEND_PID=$!

  # Wait for server to be ready
  echo "==> Waiting for server to be ready (attempt $i/$MAX_RETRIES)..."
  READY=false
  for _ in $(seq 1 30); do
    if curl -sf "http://127.0.0.1:$E2E_PORT/api/auth/me" >/dev/null 2>&1; then
      READY=true
      break
    fi
    sleep 1
  done

  if [ "$READY" = true ]; then
    echo "==> Server is ready on port $E2E_PORT"
    break
  fi

  echo "==> Server failed to start, retrying..."
  kill "$BACKEND_PID" 2>/dev/null || true
  wait "$BACKEND_PID" 2>/dev/null || true
  BACKEND_PID=""

  if [ "$i" -eq "$MAX_RETRIES" ]; then
    echo "ERROR: Server failed to start after $MAX_RETRIES attempts"
    exit 1
  fi
done

# Run e2e tests
echo "==> Running e2e tests..."
cd "$FRONTEND_DIR"
BASE_URL="http://127.0.0.1:$E2E_PORT" pnpm exec playwright test "$@"
