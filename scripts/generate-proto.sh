#!/usr/bin/env bash
# Regenerate api/gen from api/proto. Requires protoc + plugins on PATH
# (or ./.tools from a local install). Committed stubs are CI-canonical.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="${ROOT}/.tools:${PATH}"
mkdir -p api/gen
protoc \
  --proto_path=api/proto \
  --go_out=api/gen --go_opt=module=github.com/virtfoundry/core/api/gen \
  --go-grpc_out=api/gen --go-grpc_opt=module=github.com/virtfoundry/core/api/gen \
  api/proto/virtfoundry/iaas/v1alpha1/instance.proto
echo "ok: api/gen/virtfoundry/iaas/v1alpha1"
