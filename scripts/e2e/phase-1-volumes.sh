#!/usr/bin/env bash
# Phase 1: volume create, attach, detach, delete (requires PR #15 deployed).
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=scripts/e2e/common.sh
source "$SCRIPT_DIR/common.sh"

echo "=== Phase 1 E2E: volumes ==="
e2e_login

VOL=$(curl -sS "$BASE_URL/volumes" -X POST -H "$AUTH" -H 'Content-Type: application/json' \
  -d "{\"name\":\"e2e-vol-$(date +%s)\",\"size_gi\":1}")
VOL_ID=$(echo "$VOL" | python3 -c "import sys,json; print(json.load(sys.stdin)['volume']['id'])")
e2e_pass "create volume ($VOL_ID)"

VM_ID=$(curl -sS "$BASE_URL/volumes" -H "$AUTH" | python3 -c "
import sys,json
vols=[v for v in json.load(sys.stdin)['volumes'] if v['id']=='$VOL_ID']
print(vols[0].get('vm_id','') if vols else 'missing')
")
[ -z "$VM_ID" ] || e2e_fail "expected unattached volume"

VM_NAME=$(curl -sS "$BASE_URL/vms" -H "$AUTH" | python3 -c "
import sys,json
vms=json.load(sys.stdin).get('vms',[])
print(vms[0]['name'] if vms else '')
")

if [ -z "$VM_NAME" ]; then
  e2e_skip "no VM — skipping attach/detach"
else
  CODE=$(curl -sS -o /dev/null -w "%{http_code}" "$BASE_URL/vms/$VM_NAME/volumes" -X POST \
    -H "$AUTH" -H 'Content-Type: application/json' -d "{\"volume_id\":\"$VOL_ID\"}")
  [ "$CODE" = "200" ] || e2e_fail "attach failed HTTP $CODE"
  e2e_pass "attach to VM $VM_NAME"

  COUNT=$(curl -sS "$BASE_URL/vms/$VM_NAME/volumes" -H "$AUTH" | python3 -c "
import sys,json
print(len([v for v in json.load(sys.stdin)['volumes'] if v['id']=='$VOL_ID']))
")
  [ "$COUNT" = "1" ] || e2e_fail "volume not listed on VM"

  DEL_ATT=$(curl -sS -o /dev/null -w "%{http_code}" "$BASE_URL/volumes/$VOL_ID" -X DELETE -H "$AUTH")
  [ "$DEL_ATT" = "409" ] || e2e_fail "delete should return 409 while attached (got $DEL_ATT)"
  e2e_pass "delete blocked while attached (HTTP $DEL_ATT)"

  DET=$(curl -sS -o /dev/null -w "%{http_code}" "$BASE_URL/vms/$VM_NAME/volumes/$VOL_ID" -X DELETE -H "$AUTH")
  [ "$DET" = "200" ] || e2e_fail "detach failed HTTP $DET"
  e2e_pass "detach volume"
fi

DEL=$(curl -sS -o /dev/null -w "%{http_code}" "$BASE_URL/volumes/$VOL_ID" -X DELETE -H "$AUTH")
[ "$DEL" = "200" ] || e2e_fail "delete failed HTTP $DEL"
e2e_pass "delete volume"

echo "=== Phase 1 PASSED ==="
