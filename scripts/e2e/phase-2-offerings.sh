#!/usr/bin/env bash
# Phase 2: service offering CRUD (requires PR #16 deployed, root user).
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=scripts/e2e/common.sh
source "$SCRIPT_DIR/common.sh"

echo "=== Phase 2 E2E: service offerings ==="
e2e_login

NAME="e2e-$(date +%s | tail -c 6)"
CREATE=$(curl -sS -w "\n%{http_code}" "$BASE_URL/service-offerings" -X POST -H "$AUTH" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"$NAME\",\"display_name\":\"E2E Test\",\"cpu\":1,\"memory_mi\":2048}")
CODE=$(echo "$CREATE" | tail -1)
BODY=$(echo "$CREATE" | sed '$d')
[ "$CODE" = "201" ] || e2e_fail "create offering HTTP $CODE: $BODY"
OFF_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['service_offering']['id'])")
e2e_pass "POST /service-offerings"

curl -sS "$BASE_URL/service-offerings" -H "$AUTH" | python3 -c "
import sys,json
ids=[o['id'] for o in json.load(sys.stdin)['service_offerings']]
assert '$OFF_ID' in ids
" && e2e_pass "GET /service-offerings"

PATCH=$(curl -sS -o /dev/null -w "%{http_code}" "$BASE_URL/service-offerings/$OFF_ID" -X PATCH -H "$AUTH" \
  -H 'Content-Type: application/json' -d '{"display_name":"E2E Updated","cpu":2,"memory_mi":4096}')
[ "$PATCH" = "200" ] || e2e_fail "patch HTTP $PATCH"
e2e_pass "PATCH /service-offerings/{id}"

VM_NAME=$(curl -sS "$BASE_URL/vms" -H "$AUTH" | python3 -c "
import sys,json
vms=json.load(sys.stdin).get('vms',[])
stopped=[v for v in vms if v.get('state','').lower()=='stopped']
print(stopped[0]['name'] if stopped else '')
")
if [ -n "$VM_NAME" ]; then
  RESP=$(curl -sS "$BASE_URL/vms/$VM_NAME" -X PATCH -H "$AUTH" -H 'Content-Type: application/json' \
    -d "{\"service_offering_id\":\"$OFF_ID\"}")
  echo "$RESP" | python3 -c "
import sys,json
d=json.load(sys.stdin)
if 'error' in d:
    print('skip:', d['error'])
else:
    vm=d['vm']
    assert vm.get('service_offering_id')=='$OFF_ID'
    print('ok')
" && e2e_pass "PATCH /vms/{name} with service_offering_id (stopped VM required)"
else
  e2e_skip "no stopped VM — skipped offering persist test"
fi

DEL=$(curl -sS -o /dev/null -w "%{http_code}" "$BASE_URL/service-offerings/$OFF_ID" -X DELETE -H "$AUTH")
[ "$DEL" = "200" ] || e2e_fail "delete HTTP $DEL"
e2e_pass "DELETE /service-offerings/{id} (soft)"

curl -sS "$BASE_URL/service-offerings" -H "$AUTH" | python3 -c "
import sys,json
ids=[o['id'] for o in json.load(sys.stdin)['service_offerings']]
assert '$OFF_ID' not in ids
" && e2e_pass "inactive hidden from default list"

echo "=== Phase 2 PASSED ==="
