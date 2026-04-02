#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/../deployments/docker-compose.e2e.yaml"

ZITADEL_URL="http://localhost:8085"
MAX_WAIT=120  # seconds to wait for Zitadel readiness

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

log()  { echo "[e2e] $*"; }
fail() { echo "[e2e] FAIL: $*" >&2; exit 1; }

compose() {
  docker compose -f "$COMPOSE_FILE" "$@"
}

# ---------------------------------------------------------------------------
# 1. Bring up the stack (if not already running)
# ---------------------------------------------------------------------------

if ! compose ps 2>/dev/null | grep -q zitadel; then
  log "Stack not running — starting..."
  "$SCRIPT_DIR/e2e-up.sh"
fi

# ---------------------------------------------------------------------------
# 2. Wait for Zitadel to be healthy
# ---------------------------------------------------------------------------

log "Waiting for Zitadel to become healthy (max ${MAX_WAIT}s)..."
elapsed=0
while true; do
  status=$(curl -s -o /dev/null -w '%{http_code}' "${ZITADEL_URL}/debug/healthz" 2>/dev/null || true)
  if [ "$status" = "200" ]; then
    log "Zitadel is healthy."
    break
  fi
  if [ "$elapsed" -ge "$MAX_WAIT" ]; then
    fail "Zitadel did not become healthy within ${MAX_WAIT}s"
  fi
  sleep 2
  elapsed=$((elapsed + 2))
done

# ---------------------------------------------------------------------------
# 3. Retrieve PAT from the machinekey volume
# ---------------------------------------------------------------------------

log "Retrieving PAT from Zitadel machinekey volume..."

# The PAT file is written inside the zitadel container at /machinekey/.
# Find the .pat file.
PAT=""
for attempt in $(seq 1 30); do
  PAT_FILE=$(compose exec -T zitadel ls /machinekey/ 2>/dev/null | grep '\.pat$' | head -1 | tr -d '\r' || true)
  if [ -n "$PAT_FILE" ]; then
    PAT=$(compose exec -T zitadel cat "/machinekey/${PAT_FILE}" 2>/dev/null | tr -d '\r\n')
    if [ -n "$PAT" ]; then
      break
    fi
  fi
  sleep 2
done

if [ -z "$PAT" ]; then
  log "No .pat file found. Listing machinekey directory:"
  compose exec -T zitadel ls -la /machinekey/ 2>/dev/null || true
  fail "Could not retrieve PAT from Zitadel machinekey volume."
fi

log "PAT retrieved successfully (length: ${#PAT})."

# ---------------------------------------------------------------------------
# 4. Create an Actions v2 Target pointing to our service
# ---------------------------------------------------------------------------

log "Creating Actions target..."

TARGET_RESPONSE=$(curl -s -X POST "${ZITADEL_URL}/v2beta/targets" \
  -H "Authorization: Bearer ${PAT}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "e2e-webhook",
    "restWebhook": {
      "interruptOnError": false
    },
    "endpoint": "http://service:8080/user/human/added",
    "timeout": "10s"
  }')

TARGET_ID=$(echo "$TARGET_RESPONSE" | jq -r '.id // .details.id // empty' 2>/dev/null || true)

if [ -z "$TARGET_ID" ]; then
  log "Target creation response: $TARGET_RESPONSE"
  fail "Failed to create Actions target — no target ID in response."
fi

log "Target created: ${TARGET_ID}"

# ---------------------------------------------------------------------------
# 5. Create an Execution that maps user.human.added to the target
# ---------------------------------------------------------------------------

log "Setting execution for user.human.added event..."

EXECUTION_RESPONSE=$(curl -s -X PUT "${ZITADEL_URL}/v2beta/executions" \
  -H "Authorization: Bearer ${PAT}" \
  -H "Content-Type: application/json" \
  -d '{
    "condition": {
      "event": {
        "event": "user.human.added"
      }
    },
    "targets": [
      {
        "target": "'"${TARGET_ID}"'"
      }
    ]
  }')

log "Execution response: ${EXECUTION_RESPONSE}"

# ---------------------------------------------------------------------------
# 6. Create a test human user (triggers the webhook)
# ---------------------------------------------------------------------------

log "Creating test human user..."

USER_RESPONSE=$(curl -s -X POST "${ZITADEL_URL}/v2/users/human" \
  -H "Authorization: Bearer ${PAT}" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "e2e-testuser",
    "profile": {
      "givenName": "Test",
      "familyName": "User"
    },
    "email": {
      "email": "e2e-test@example.com",
      "isVerified": true
    },
    "password": {
      "password": "E2eTestPassword1!",
      "changeRequired": false
    }
  }')

USER_ID=$(echo "$USER_RESPONSE" | jq -r '.userId // .user_id // empty' 2>/dev/null || true)

if [ -z "$USER_ID" ]; then
  log "User creation response: $USER_RESPONSE"
  fail "Failed to create test user — no user ID in response."
fi

log "Test user created: ${USER_ID}"

# ---------------------------------------------------------------------------
# 7. Wait for the webhook to fire, then check service logs
# ---------------------------------------------------------------------------

log "Waiting for webhook delivery (10s)..."
sleep 10

log "Checking service logs for received event..."

SERVICE_LOGS=$(compose logs service 2>/dev/null || true)

if echo "$SERVICE_LOGS" | grep -q "received UserHumanAdded event"; then
  log "================================================"
  log "  PASS: Service received UserHumanAdded event!"
  log "================================================"
  exit 0
fi

# Show service logs for debugging
log "Service logs:"
echo "$SERVICE_LOGS"
log "================================================"
log "  FAIL: Expected log line not found."
log "================================================"
exit 1
