#!/usr/bin/env bash
# Shared helpers for VirtFoundry API E2E tests against homelab or any deployment.
set -euo pipefail

BASE_URL="${BASE_URL:-http://virtfoundry.homelab/api/v1}"
E2E_USER="${E2E_USER:-root}"
E2E_PASS="${E2E_PASS:-virtfoundry}"

e2e_login() {
  TOKEN=$(curl -sS "$BASE_URL/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"$E2E_USER\",\"password\":\"$E2E_PASS\"}" \
    | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
  export TOKEN
  export AUTH="Authorization: Bearer $TOKEN"
}

e2e_pass() { echo "✓ $*"; }
e2e_fail() { echo "✗ $*" >&2; exit 1; }
e2e_skip() { echo "⚠ $*"; }
