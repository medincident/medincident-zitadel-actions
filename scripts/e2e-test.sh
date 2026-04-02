#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/../deployments/docker-compose.e2e.yaml"

ZITADEL_URL="http://localhost:8085"
MAX_WAIT=120  # seconds to wait for Zitadel readiness
SUFFIX=$(date +%s)

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

log "Retrieving PAT from Zitadel logs..."

# The Zitadel image is distroless (no ls/cat). When PatPath is not set,
# Zitadel writes the machine key JSON and PAT to stdout during first
# instance setup. The PAT appears on the line immediately after the
# machine key JSON line (which contains "serviceaccount").
PAT=""
for attempt in $(seq 1 30); do
  # The PAT is the line right after the JSON machine key blob
  PAT=$(compose logs zitadel 2>/dev/null \
    | sed -n '/serviceaccount/{n;p;}' \
    | head -1 \
    | sed 's/^.*| //' \
    | tr -d '\r\n ')
  if [ -n "$PAT" ]; then
    break
  fi
  sleep 2
done

if [ -z "$PAT" ]; then
  fail "Could not retrieve PAT from Zitadel logs."
fi

log "PAT retrieved successfully (length: ${#PAT})."

# ---------------------------------------------------------------------------
# 4. Create an Actions v2 Target pointing to our service
# ---------------------------------------------------------------------------

log "Creating Actions target..."

TARGET_RESPONSE=$(curl -s -X POST \
  "${ZITADEL_URL}/zitadel.action.v2beta.ActionService/CreateTarget" \
  -H "Authorization: Bearer ${PAT}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "e2e-webhook-'"${SUFFIX}"'",
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

EXECUTION_RESPONSE=$(curl -s -X POST \
  "${ZITADEL_URL}/zitadel.action.v2beta.ActionService/SetExecution" \
  -H "Authorization: Bearer ${PAT}" \
  -H "Content-Type: application/json" \
  -d '{
    "condition": {
      "event": {
        "event": "user.human.added"
      }
    },
    "targets": ["'"${TARGET_ID}"'"]
  }')

log "Execution response: ${EXECUTION_RESPONSE}"

if echo "$EXECUTION_RESPONSE" | jq -e '.code' >/dev/null 2>&1; then
  log "Execution creation failed: $EXECUTION_RESPONSE"
  fail "Failed to set execution."
fi

# ---------------------------------------------------------------------------
# 6. Create a test human user (triggers the webhook)
# ---------------------------------------------------------------------------

log "Creating test human user..."

USER_RESPONSE=$(curl -s -X POST "${ZITADEL_URL}/v2/users/human" \
  -H "Authorization: Bearer ${PAT}" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "e2e-testuser-'"${SUFFIX}"'",
    "profile": {
      "givenName": "Test",
      "familyName": "User"
    },
    "email": {
      "email": "e2e-test-'"${SUFFIX}"'@example.com",
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
# 7. Wait for the user.human.added webhook to fire
# ---------------------------------------------------------------------------

log "Waiting for UserHumanAdded webhook delivery (polling every 2s, max 30s)..."

webhook_wait=0
while true; do
  SERVICE_LOGS=$(compose logs service 2>/dev/null || true)
  if echo "$SERVICE_LOGS" | grep -q "received UserHumanAdded event"; then
    log "PASS: Service received UserHumanAdded event."
    break
  fi
  if [ "$webhook_wait" -ge 30 ]; then
    log "Service logs:"
    echo "$SERVICE_LOGS"
    fail "UserHumanAdded log line not found within 30s."
  fi
  sleep 2
  webhook_wait=$((webhook_wait + 2))
done

# ---------------------------------------------------------------------------
# 8. Create an Actions target + execution for user.human.profile.changed
# ---------------------------------------------------------------------------

log "Creating Actions target for profile changed..."

PROFILE_TARGET_RESPONSE=$(curl -s -X POST \
  "${ZITADEL_URL}/zitadel.action.v2beta.ActionService/CreateTarget" \
  -H "Authorization: Bearer ${PAT}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "e2e-webhook-profile-'"${SUFFIX}"'",
    "restWebhook": {
      "interruptOnError": false
    },
    "endpoint": "http://service:8080/user/human/profile/changed",
    "timeout": "10s"
  }')

PROFILE_TARGET_ID=$(echo "$PROFILE_TARGET_RESPONSE" | jq -r '.id // .details.id // empty' 2>/dev/null || true)

if [ -z "$PROFILE_TARGET_ID" ]; then
  log "Profile target creation response: $PROFILE_TARGET_RESPONSE"
  fail "Failed to create profile Actions target — no target ID in response."
fi

log "Profile target created: ${PROFILE_TARGET_ID}"

log "Setting execution for user.human.profile.changed event..."

PROFILE_EXEC_RESPONSE=$(curl -s -X POST \
  "${ZITADEL_URL}/zitadel.action.v2beta.ActionService/SetExecution" \
  -H "Authorization: Bearer ${PAT}" \
  -H "Content-Type: application/json" \
  -d '{
    "condition": {
      "event": {
        "event": "user.human.profile.changed"
      }
    },
    "targets": ["'"${PROFILE_TARGET_ID}"'"]
  }')

log "Profile execution response: ${PROFILE_EXEC_RESPONSE}"

if echo "$PROFILE_EXEC_RESPONSE" | jq -e '.code' >/dev/null 2>&1; then
  log "Profile execution creation failed: $PROFILE_EXEC_RESPONSE"
  fail "Failed to set profile execution."
fi

# ---------------------------------------------------------------------------
# 9. Update the test user's profile (triggers user.human.profile.changed)
# ---------------------------------------------------------------------------

log "Updating test user profile..."

UPDATE_RESPONSE=$(curl -s -X POST \
  "${ZITADEL_URL}/zitadel.user.v2.UserService/UpdateHumanUser" \
  -H "Authorization: Bearer ${PAT}" \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "'"${USER_ID}"'",
    "profile": {
      "givenName": "Updated",
      "familyName": "Profile"
    }
  }')

log "Profile update response: ${UPDATE_RESPONSE}"

if echo "$UPDATE_RESPONSE" | jq -e '.code' >/dev/null 2>&1; then
  fail "Profile update failed: $UPDATE_RESPONSE"
fi

# ---------------------------------------------------------------------------
# 10. Wait for the user.human.profile.changed webhook to fire
# ---------------------------------------------------------------------------

log "Waiting for UserHumanProfileChanged webhook delivery (polling every 2s, max 30s)..."

webhook_wait=0
while true; do
  SERVICE_LOGS=$(compose logs service 2>/dev/null || true)
  if echo "$SERVICE_LOGS" | grep -q "received UserHumanProfileChanged event"; then
    log "PASS: Service received UserHumanProfileChanged event."
    break
  fi
  if [ "$webhook_wait" -ge 30 ]; then
    log "Service logs:"
    echo "$SERVICE_LOGS"
    fail "UserHumanProfileChanged log line not found within 30s."
  fi
  sleep 2
  webhook_wait=$((webhook_wait + 2))
done

# ---------------------------------------------------------------------------
# Final result
# ---------------------------------------------------------------------------

log "================================================"
log "  PASS: All E2E tests passed!"
log "================================================"
exit 0
