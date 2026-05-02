#!/usr/bin/env bash
#
# Local / CI runner for Playwright E2E tests against a freshly-built warren
# server.
#
# Environment variables (all optional; CI=true ignores the skip flags below):
#   E2E_SKIP_FRONTEND_INSTALL=1   skip `pnpm install`
#   E2E_SKIP_FRONTEND_BUILD=1     skip `pnpm run build` (requires existing dist)
#   E2E_SKIP_BACKEND_BUILD=1      skip `go build`; uses $E2E_WARREN_BIN instead
#   E2E_WARREN_BIN=/path/to/bin   pre-built warren binary (with above)
#   E2E_PORT_MIN / E2E_PORT_MAX   override port selection range
#                                 (default 49152..60999, IANA dynamic range,
#                                  picked to avoid common dev ports like
#                                  3000/3306/5173/5432/8080/8443/9000)
#
# Examples:
#   ./frontend/scripts/e2e.sh
#   ./frontend/scripts/e2e.sh frontend/e2e/tests/alert.spec.ts
#   E2E_SKIP_FRONTEND_INSTALL=1 E2E_SKIP_FRONTEND_BUILD=1 \
#     E2E_SKIP_BACKEND_BUILD=1 E2E_WARREN_BIN=$(pwd)/warren \
#     ./frontend/scripts/e2e.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
FRONTEND_DIR="$PROJECT_ROOT/frontend"

E2E_PORT_MIN="${E2E_PORT_MIN:-49152}"
E2E_PORT_MAX="${E2E_PORT_MAX:-60999}"

# In CI, always perform a full build. Skip flags are a local-only convenience.
if [ "${CI:-}" = "true" ]; then
  E2E_SKIP_FRONTEND_INSTALL=""
  E2E_SKIP_FRONTEND_BUILD=""
  E2E_SKIP_BACKEND_BUILD=""
fi

# Cleanup on exit
BACKEND_PID=""
WARREN_BIN_TMP=""
cleanup() {
  if [ -n "$BACKEND_PID" ]; then
    if kill -0 "$BACKEND_PID" 2>/dev/null; then
      echo "Stopping backend server (PID: $BACKEND_PID)..."
      kill "$BACKEND_PID" 2>/dev/null || true
      # Give it up to 5s to exit cleanly, then SIGKILL.
      for _ in 1 2 3 4 5; do
        if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
          break
        fi
        sleep 1
      done
      if kill -0 "$BACKEND_PID" 2>/dev/null; then
        echo "Backend did not exit on SIGTERM; sending SIGKILL."
        kill -9 "$BACKEND_PID" 2>/dev/null || true
      fi
      wait "$BACKEND_PID" 2>/dev/null || true
    fi
  fi
  if [ -n "$WARREN_BIN_TMP" ] && [ -f "$WARREN_BIN_TMP" ]; then
    rm -f "$WARREN_BIN_TMP"
  fi
}
trap cleanup EXIT

# Frontend install
#
# Note: frontend/pnpm-workspace.yaml declares `allowBuilds: { esbuild: true }`
# so that pnpm 11 explicitly approves esbuild's postinstall script. Without
# this approval, pnpm 11 exits non-zero with ERR_PNPM_IGNORED_BUILDS.
# (`onlyBuiltDependencies` was removed in pnpm 11; use `allowBuilds` instead.)
if [ "${E2E_SKIP_FRONTEND_INSTALL:-}" = "1" ]; then
  echo "==> Skipping frontend install (E2E_SKIP_FRONTEND_INSTALL=1)"
else
  echo "==> Installing frontend dependencies..."
  cd "$FRONTEND_DIR"
  pnpm install
fi

# Frontend build
if [ "${E2E_SKIP_FRONTEND_BUILD:-}" = "1" ]; then
  echo "==> Skipping frontend build (E2E_SKIP_FRONTEND_BUILD=1)"
  if [ ! -d "$FRONTEND_DIR/dist" ]; then
    echo "ERROR: E2E_SKIP_FRONTEND_BUILD=1 but $FRONTEND_DIR/dist does not exist."
    exit 1
  fi
else
  echo "==> Building frontend..."
  cd "$FRONTEND_DIR"
  pnpm run build
fi

# Backend binary
if [ "${E2E_SKIP_BACKEND_BUILD:-}" = "1" ]; then
  if [ -z "${E2E_WARREN_BIN:-}" ]; then
    echo "ERROR: E2E_SKIP_BACKEND_BUILD=1 requires E2E_WARREN_BIN to be set."
    exit 1
  fi
  if [ ! -x "$E2E_WARREN_BIN" ]; then
    echo "ERROR: E2E_WARREN_BIN ($E2E_WARREN_BIN) is not executable."
    exit 1
  fi
  WARREN_BIN="$E2E_WARREN_BIN"
  echo "==> Using prebuilt warren binary: $WARREN_BIN"
else
  echo "==> Building backend..."
  cd "$PROJECT_ROOT"
  WARREN_BIN_TMP=$(mktemp "${TMPDIR:-/tmp}/warren-e2e.XXXXXX")
  go build -o "$WARREN_BIN_TMP" .
  WARREN_BIN="$WARREN_BIN_TMP"
fi

# Pick a random port in the configured range. The IANA "dynamic" range
# (49152-65535) is reserved for ephemeral allocation and avoids well-known
# dev ports (3000, 3306, 5173, 5432, 8080, 8443, 9000, etc.) that frequently
# collide with other locally running products.
pick_port() {
  local port
  while true; do
    port=$((RANDOM % (E2E_PORT_MAX - E2E_PORT_MIN + 1) + E2E_PORT_MIN))
    if ! lsof -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
      echo "$port"
      return
    fi
  done
}

# Start backend; retry on a fresh port if it fails to come up.
MAX_RETRIES=3
E2E_PORT=""
for i in $(seq 1 $MAX_RETRIES); do
  CANDIDATE_PORT=$(pick_port)
  echo "==> Starting backend server on 127.0.0.1:$CANDIDATE_PORT (attempt $i/$MAX_RETRIES)..."

  "$WARREN_BIN" serve \
    --addr="127.0.0.1:$CANDIDATE_PORT" \
    --no-authn \
    --no-authz \
    --enable-graphql \
    --disable-llm \
    --log-level=error &
  BACKEND_PID=$!

  READY=false
  for _ in $(seq 1 30); do
    # If the backend already died (port collision or other error), bail out
    # of the readiness loop early so we can retry on a different port.
    if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
      break
    fi
    if curl -sf "http://127.0.0.1:$CANDIDATE_PORT/api/auth/me" >/dev/null 2>&1; then
      READY=true
      break
    fi
    sleep 1
  done

  if [ "$READY" = true ]; then
    E2E_PORT="$CANDIDATE_PORT"
    echo "==> Server is ready on port $E2E_PORT"
    break
  fi

  echo "==> Server failed to become ready on port $CANDIDATE_PORT; retrying with a fresh port..."
  if kill -0 "$BACKEND_PID" 2>/dev/null; then
    kill "$BACKEND_PID" 2>/dev/null || true
  fi
  wait "$BACKEND_PID" 2>/dev/null || true
  BACKEND_PID=""

  if [ "$i" -eq "$MAX_RETRIES" ]; then
    echo "ERROR: Server failed to start after $MAX_RETRIES attempts."
    echo "Last attempted port: $CANDIDATE_PORT"
    echo "lsof for that port:"
    lsof -iTCP:"$CANDIDATE_PORT" -sTCP:LISTEN || true
    exit 1
  fi
done

# Run e2e tests.
#
# Invoke playwright directly via node_modules/.bin instead of `pnpm exec`.
# Recent pnpm versions run a "deps status check" before exec that triggers
# a fresh `pnpm install`. That re-install is slow on every iteration and,
# on some sandboxed filesystems, fails with reflink EPERM. Calling the
# binary directly skips that check entirely.
echo "==> Running e2e tests against http://127.0.0.1:$E2E_PORT ..."
cd "$FRONTEND_DIR"
PLAYWRIGHT_BIN="$FRONTEND_DIR/node_modules/.bin/playwright"
if [ ! -x "$PLAYWRIGHT_BIN" ]; then
  echo "ERROR: $PLAYWRIGHT_BIN not found or not executable."
  echo "Run pnpm install first, or omit E2E_SKIP_FRONTEND_INSTALL."
  exit 1
fi
BASE_URL="http://127.0.0.1:$E2E_PORT" "$PLAYWRIGHT_BIN" test "$@"
