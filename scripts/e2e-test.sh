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
# Helper: create_target NAME ENDPOINT → prints target ID
# ---------------------------------------------------------------------------

create_target() {
  local name="$1" endpoint="$2"
  local body
  body=$(printf '{"name":"%s-%s","restWebhook":{"interruptOnError":false},"endpoint":"%s","timeout":"10s"}' "$name" "$SUFFIX" "$endpoint")
  local resp
  resp=$(curl -s -X POST \
    "${ZITADEL_URL}/zitadel.action.v2beta.ActionService/CreateTarget" \
    -H "Authorization: Bearer ${PAT}" \
    -H "Content-Type: application/json" \
    -d "$body")
  local tid
  tid=$(echo "$resp" | jq -r '.id // .details.id // empty' 2>/dev/null || true)
  if [ -z "$tid" ]; then
    log "Target creation response: $resp" >&2
    fail "Failed to create target '${name}'."
  fi
  log "Target '${name}' created: ${tid}" >&2
  echo "$tid"
}

# ---------------------------------------------------------------------------
# Helper: set_execution CONDITION_JSON TARGET_ID
# ---------------------------------------------------------------------------

set_execution() {
  local condition="$1" target_id="$2"
  local body
  body=$(printf '{"condition":%s,"targets":["%s"]}' "$condition" "$target_id")
  local resp
  resp=$(curl -s -X POST \
    "${ZITADEL_URL}/zitadel.action.v2beta.ActionService/SetExecution" \
    -H "Authorization: Bearer ${PAT}" \
    -H "Content-Type: application/json" \
    -d "$body")
  log "Execution response: ${resp}"
  if echo "$resp" | jq -e '.code' >/dev/null 2>&1; then
    fail "Failed to set execution: $resp"
  fi
}

# ---------------------------------------------------------------------------
# Helper: wait_for_log PATTERN LABEL
# ---------------------------------------------------------------------------

wait_for_log() {
  local pattern="$1" label="$2"
  log "Waiting for ${label} (polling every 2s, max 30s)..."
  local elapsed=0
  while true; do
    SERVICE_LOGS=$(compose logs service 2>/dev/null || true)
    if echo "$SERVICE_LOGS" | grep -q "$pattern"; then
      log "PASS: ${label}"
      return 0
    fi
    if [ "$elapsed" -ge 30 ]; then
      log "Service logs:"
      echo "$SERVICE_LOGS"
      fail "${label} — log line not found within 30s."
    fi
    sleep 2
    elapsed=$((elapsed + 2))
  done
}

# ---------------------------------------------------------------------------
# 4. Create targets for all endpoints
# ---------------------------------------------------------------------------

ADDED_TID=$(create_target "e2e-added"           "http://service:8080/user/human/added")
PROFILE_TID=$(create_target "e2e-profile"        "http://service:8080/user/human/profile/changed")
EVENT_TID=$(create_target "e2e-any-event"        "http://service:8080/event")
REQUEST_TID=$(create_target "e2e-any-request"    "http://service:8080/request")
RESPONSE_TID=$(create_target "e2e-any-response"  "http://service:8080/response")

# ---------------------------------------------------------------------------
# 5. Create executions
# ---------------------------------------------------------------------------

# Event executions — specific event types
set_execution '{"event":{"event":"user.human.added"}}'           "$ADDED_TID"
set_execution '{"event":{"event":"user.human.profile.changed"}}' "$PROFILE_TID"

# Catch-all event execution (all events)
set_execution '{"event":{"all":true}}'  "$EVENT_TID"

# Request execution — fires before any API request is processed
set_execution '{"request":{"all":true}}' "$REQUEST_TID"

# Response execution — fires after any API response is sent
set_execution '{"response":{"all":true}}' "$RESPONSE_TID"

# ---------------------------------------------------------------------------
# 6. Create a test human user (triggers event + request/response hooks)
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
# 7. Verify typed event handler
# ---------------------------------------------------------------------------

wait_for_log "received UserHumanAdded event"  "Service received UserHumanAdded event"

# ---------------------------------------------------------------------------
# 8. Verify catch-all event handler
# ---------------------------------------------------------------------------

wait_for_log "received event"  "Service received catch-all event"

# ---------------------------------------------------------------------------
# 9. Verify request handler
# ---------------------------------------------------------------------------

wait_for_log "received request"  "Service received request hook"

# ---------------------------------------------------------------------------
# 10. Verify response handler
# ---------------------------------------------------------------------------

wait_for_log "received response"  "Service received response hook"

# ---------------------------------------------------------------------------
# 11. Update profile (triggers user.human.profile.changed)
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
# 12. Verify profile changed handler
# ---------------------------------------------------------------------------

wait_for_log "received UserHumanProfileChanged event"  "Service received UserHumanProfileChanged event"

# ---------------------------------------------------------------------------
# Final result
# ---------------------------------------------------------------------------

log "================================================"
log "  PASS: All E2E tests passed!"
log "================================================"
exit 0
