#!/usr/bin/env bash
# Phase 3: template catalog dedup and CRUD smoke (requires PR #17 deployed).
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=scripts/e2e/common.sh
source "$SCRIPT_DIR/common.sh"

echo "=== Phase 3 E2E: templates ==="
e2e_login

curl -sS "$BASE_URL/vm-templates" -H "$AUTH" | python3 -c "
import sys, json
from collections import Counter
templates = json.load(sys.stdin).get('vm_templates', [])
names = [t['name'] for t in templates]
dupes = [n for n, c in Counter(names).items() if c > 1]
assert not dupes, f'duplicate names: {dupes}'
platform = {t['name'] for t in templates if not t.get('tenant_id')}
overlap = [t['name'] for t in templates if t.get('tenant_id') and t['name'] in platform]
assert not overlap, f'tenant/platform overlap: {overlap}'
print(f'ok: {len(templates)} templates, no dupes')
" && e2e_pass "template catalog dedup"

TNAME="e2e-tmpl-$(date +%s | tail -c 6)"
CREATE=$(curl -sS -w "\n%{http_code}" "$BASE_URL/vm-templates" -X POST -H "$AUTH" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"$TNAME\",\"display_name\":\"E2E\",\"image\":\"quay.io/kubevirt/cirros-container-disk-demo\",\"source_type\":\"container\",\"os_type\":\"linux\"}")
CODE=$(echo "$CREATE" | tail -1)
[ "$CODE" = "201" ] || e2e_fail "create template HTTP $CODE"
TID=$(echo "$CREATE" | sed '$d' | python3 -c "import sys,json; print(json.load(sys.stdin)['vm_template']['id'])")
e2e_pass "POST /vm-templates"

DEL=$(curl -sS -o /dev/null -w "%{http_code}" "$BASE_URL/vm-templates/$TID" -X DELETE -H "$AUTH")
[ "$DEL" = "200" ] || e2e_fail "delete template HTTP $DEL"
e2e_pass "DELETE /vm-templates/{id}"

REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
[ -f "$REPO_ROOT/docs/VM-TEMPLATES.md" ] && e2e_pass "docs/VM-TEMPLATES.md present" || e2e_skip "docs/VM-TEMPLATES.md (check from repo checkout)"

echo "=== Phase 3 PASSED ==="
