# UI CN mode Phase 1 — VMs watch-only implementation plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** VMs list uses one snapshot + `/ws/events` patches; no periodic `GET /vms` while WS healthy.

**Architecture:** Approach A from `docs/superpowers/specs/2026-09-30-ui-cloud-native-watch-design.md`.

**Tech:** React Query cache patch in `realtime-invalidation.ts` / new `applyVmEvent.ts`; `VMs.tsx` poll policy; optional hub payload enrich in core.

---

### Task 1: Hub payload audit

**Files:** `internal/api/ws/`, compute broadcast sites  
**Steps:** List fields on `vm.created|updated|deleted`. If missing `state`/`ip`/`cpu`/`memory_mi`, enrich at publish OR document one GET on create.  
**Verify:** Unit test or log sample from homelab Watch/WS.

### Task 2: Cache merge-patch helper

**Files:** `ui/src/lib/realtime-invalidation.ts` (or `ui/src/lib/vm-event-cache.ts`)  
**Steps:**
1. Write failing test: existing row `{name, state: Running, ip, cpu, memory_mi}` + event `{name, state: Stopped}` → row keeps `ip`/`cpu`/`memory_mi`, updates `state`.
2. Implement merge-patch `setQueryData(queryKeys.vms)`: upsert by `name`; only overwrite keys present in payload; `vm.deleted` filters out.
3. Remove happy-path `invalidateQueries(vms)` for `vm.*` (detail `queryKeys.vm(name)` optional).
4. `vm.created` incomplete → one GET **or** rely on Task 1 enrich (document choice in PR).
5. Narrow side-invalidates: pure state `vm.updated` should not invalidate volumes/dashboard.

**Verify:** unit test green; manual Start/Stop without `/vms` list refetch while WS up.

### Task 3: VMs poll off

**Files:** `ui/src/pages/VMs.tsx`  
**Steps:** `realtimePollInterval(wsConnected, transitional, { healthyMs: false })`.  
**Verify:** Network tab — no `/vms` while idle + WS up.

### Task 4: Smoke + PR

Homelab: connect WS, start/stop VM, confirm row updates without list poll.  
PR: `feat(ui): VMs watch-only when realtime connected` linking Epic issue.

---

### Out of scope here

Phase 2 Instance informer gateway; Templates cold path (separate issue).
