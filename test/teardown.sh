#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== Tearing Down Test Environment ==="
echo ""

echo "Stopping containers..."
docker compose -f "$SCRIPT_DIR/docker-compose.yml" down -v

echo ""
echo "Test environment stopped and cleaned up."
