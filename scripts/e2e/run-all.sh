#!/usr/bin/env bash
# Run all phase E2E scripts. Deploy the matching branch/stack before each phase.
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
for script in phase-1-volumes.sh phase-2-offerings.sh phase-3-templates.sh; do
  bash "$DIR/$script"
  echo
done
