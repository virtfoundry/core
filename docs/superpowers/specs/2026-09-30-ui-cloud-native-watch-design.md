# UI cloud-native mode — CRD watch gateway

**Date:** 2026-09-30  
**Status:** Approved — Phase 1 implementing  
**Repos:** `virtfoundry/core` (UI + API gateway), `virtfoundry/operator` (CRD truth), board UI #4 + Roadmap #3  
**Related:** Epic A CRD-first, core#160 realtime invalidation, core#164 gRPC WatchInstances, `/ws/events`

## Problem

The UI still behaves like a REST client: list + poll (+ invalidate on WS). Homelab smoke showed cold `GET /vm-templates` ~20s; VMs page still polls even when `/ws/events` is healthy. Product goal is **cloud-native mode**: CRDs are the source of truth; the console should **subscribe to state**, not chat product REST for truth.

“No REST/API” for the product surface means: **no chatty facade as the primary data path**. The browser still needs a transport (WS / gRPC-web). It must **not** talk to kube-apiserver directly (auth, CORS, multi-tenant RBAC).

## Goals

1. VMs page (spike): **one snapshot + watch**; no periodic `listVMs` while hub is connected.
2. Gateway path that can evolve to **Instance CR watch** (not only hub `vm.*` events).
3. Keep login, console ticket, and mutating actions on existing endpoints until a later phase.
4. Measurable: with WS healthy, VMs list `refetchInterval === false` except transitional fallback.

## Non-goals (this spike)

- Rewriting all pages (Templates, Networks, …).
- Browser → Kubernetes API.
- Killing REST for Terraform / external clients (gRPC spike remains for those).
- Replacing noVNC console transport.

## Approaches considered

| # | Approach | Pros | Cons |
|---|----------|------|------|
| A | **Hub-first watch UI** — enrich `/ws/events` payloads; UI applies patches to React Query cache; poll off when connected | Ships on today’s hub; small surface | Hub events still thin (name/state); not full CR |
| B | **CRD watch gateway** — API watches `Instance` in tenant ns; WS/gRPC stream full objects | True CN mode; matches operatorReconcile | More backend work; informer lifecycle |
| C | **gRPC-web only from UI** | One protocol with TF/CLI | Browser tooling heavier; still needs gateway |

**Recommendation:** **A now (spike)** → **B next** (same WS contract, richer payload). C optional later for SDK uniformity.

## Design — Phase 1 (spike): VMs watch-only

```
UI (VMs.tsx)
  │  initial: GET /vms  (one shot) OR first WS snapshot message
  │  live:    /ws/events  vm.created|updated|deleted
  │  poll:    false while useRealtimeConnected(); transitional only if WS down
  ▼
core Hub  ◄── compute/operator status publishers (existing)
```

### UI

- `realtimePollInterval(..., { healthyMs: false })` on VMs list when not transitional; transitional + WS down keeps short poll.
- On `vm.*`, **merge-patch** React Query `queryKeys.vms`: only overwrite keys present in the event payload; never replace the whole row with a thin object (today hub often sends only `id`/`name`/`state`). `vm.deleted` removes by `name`.
- Do **not** `invalidateQueries(vms)` on the happy path (that forces list refetch). Optional: skip volumes/dashboard invalidate on pure `vm.updated` state changes.
- `vm.created` with incomplete sizing/IP: one-shot `GET /vms/{name}` **or** hub enrich (Phase 1 Task 1) — pick one; start/stop only needs `state` merge.
- Document that wizard templates stay on REST until a separate cold-path fix.

### API / hub (minimal for spike)

- Audit `vm.created|updated|deleted` publishers. Prefer enriching `state` (always) and `ip` when known; cpu/memory optional if merge-patch preserves existing row fields.
- No new CRD informer required for Phase 1.

### Acceptance (Phase 1)

- [x] Code: with `/ws/events` connected and no transitional VMs, list `refetchInterval` is `false` (`healthyMs: false`).
- [x] Code: Start/Stop/Destroy do not `invalidateQueries(vms)`; row updates via optimistic + merge-patch (create may one-shot `getVM`).
- [x] Code: WS disconnect keeps transitional poll via `POLL_WS_DOWN_MS`.
- [ ] Homelab smoke checklist (below) — run after Argo digest.

### Homelab smoke (Phase 1)

1. Open `http://virtfoundry.homelab` → VMs; DevTools Network filter `/vms`.
2. Confirm `/ws/events` connected (WS tab).
3. Idle 30s with no transitional VMs → **no** periodic `GET /vms` (manual Refresh OK).
4. Start then Stop a Running VM → row state updates; **no** list `GET /vms` on settle (optimistic + `vm.updated`).
5. Destroy a throwaway VM → row disappears via `vm.deleted` without list refetch.
6. Kill WS (offline) with a transitional VM → short poll (`POLL_WS_DOWN_MS`) resumes.

## Design — Phase 2: Instance CR watch gateway

```
UI ──WS──► core watch gateway ──informers──► Instance (tenant ns)
                      │
                      └── maps full Instance → typed events (not status-only)
```

- Stream **full CR semantics**: `spec.powerState` (desired) + `status.phase` / `status.ip` (observed). Status-only mapping breaks Start/Stop UX under `operatorReconcile`.
- Normalize phase vocabulary for UI (`InstancePhaseToPlatformState`); do not leak raw CR aliases (`Ready` vs `Running`) inconsistently.
- CPU/mem: join Offering (or keep last known from list cache) — not on Instance status today.
- **Evolve** gRPC `WatchInstances` off hub-only path onto the same informer publisher (one stream, no dual hub+informer fanout). Until then, do not claim “shared contract” means today’s thin hub events.
- REST `GET /vms` becomes bootstrap-only or disappears from UI.
- Operator remains reconciler; UI never dual-writes KV.

## Security

- Same auth as `/ws/events` (ticket or header); tenant scope mandatory.
- No JWT in query string (already fixed #133).
- Gateway must not widen RBAC beyond today’s `vms:read`.

## Agents (validation)

Local Cursor agents (not committed to GitHub):

| Agent | Role |
|-------|------|
| **vf-ui-cn** | Spec/plan adherence for UI CN mode; VMs watch-only checklist |
| **vf-react** | Implement UI cache patch + poll off |
| **vf-cloud-native** | Validate CRD/watch semantics vs cluster |
| **vf-core** | Hub enrichment / gateway wiring |
| **vf-security** | Ticket/tenant scope on watch streams |

## Open questions

1. Enrich hub payloads vs one GET on `vm.created` — prefer enrich if cheap.
2. Phase 2 informer in-process in `core` vs sidecar — default in-process with existing kube client.

## References

- `ui/src/hooks/useRealtimeEvents.ts`
- `ui/src/lib/realtime-invalidation.ts`
- `internal/api/ws/hub.go`
- `internal/api/grpc/instance.go` (WatchInstances)
- `docs/superpowers/specs/2026-09-01-crd-operator-design.md`
