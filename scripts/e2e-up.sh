#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/../deployments/docker-compose.e2e.yaml"

echo "==> Starting E2E stack..."
docker compose -f "$COMPOSE_FILE" up -d --build --wait

echo "==> E2E stack is up."
