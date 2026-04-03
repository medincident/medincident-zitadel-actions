#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/../deployments/docker-compose.e2e.yaml"

echo "==> Tearing down E2E stack..."
docker compose -f "$COMPOSE_FILE" down -v --remove-orphans

echo "==> E2E stack is down."
