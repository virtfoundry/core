# Running the API with more than one replica (design)

> **Status: proposal, awaiting maintainer approval. No code is changed by this document.**
> Scope: `core` (API) and the `virtfoundry` chart in `helm-charts`. Today the chart ships `api.replicas: 1`.

## Goal

Allow `api.replicas > 1` (rolling updates without downtime, node drain tolerance) without
silent data loss, stale reads that users notice, or weakened security guarantees.

Non-goals: horizontal scaling for throughput, multi-region, changing the REST contract or the CRD schema.

## Inventory of process-local state

Found by reading `internal/service/compute`, `internal/auth`, `internal/api/ws`,
`internal/platform/store` and `cmd/server/main.go` at `main`.

| # | State | Where | Multi-replica effect | Severity |
|---|-------|-------|----------------------|----------|
| 1 | Load balancers, target groups, listeners, targets | `store/kubernetes_ephemeral.go` (`lbListeners`, `lbTargets`, …) | Written on replica A, invisible on B; lost on restart | **Blocker** |
| 2 | Audit events | same file (`SaveAuditEvent`) | Each replica has its own partial log; lost on restart | **Blocker** (compliance) |
| 3 | Async jobs (`jobs` map) | same file + `service/jobs` | `GET` of a job on another replica returns 404. `ProcessPendingJobs` is **never called** from `cmd/server`, so `async: true` deploys are accepted and never run (verify; this is true today even with 1 replica) | Bug today, blocker later |
| 4 | `vmListCache` (20 s TTL per tenant) | `compute/vm_cache.go` | Read-after-write on a different replica can miss a fresh VM for up to one sync tick (5 s) | Medium |
| 5 | `vmStates` signature map + `broadcastVM` | `compute/service.go` | Each replica diffs and broadcasts to its own hub only | Low (see below) |
| 6 | WebSocket hub (clients, subscribers) | `internal/api/ws/hub.go` | A client only sees events that its replica produced | Low, depends on #5 |
| 7 | `ConsoleTicketStore` JTI burn map | `auth/console_ticket.go` | Tickets are HMAC-signed with the shared `JWT_SECRET`, so any replica can redeem them. Single use is enforced only on the minting replica; replay on another replica is bounded by the 30 s TTL (already documented in core#133) | Low, security-relevant |
| 8 | `LoginThrottle` counters | `auth/throttle.go` | Brute-force budget is multiplied by the replica count | Low |
| 9 | Tenant and role-permission caches (30 s TTL) | `store/kubernetes_cache.go` | Invalidated only locally; another replica sees a tenant or role change up to 30 s late | Low |
| 10 | Background loops | `cmd/server/main.go:292` (`SyncAllVMStates`, 5 s), `ReconcileAll` (job only), per-request ISO import goroutine | See "Leader election" | Medium |

Things that are **already** replica-safe: JWT sessions (stateless, shared secret), CR-backed
store objects (Tenant, Instance, Network, Disk, …), the Instance informer (each replica runs its own).

## Design

Principle: **put shared state in the Kubernetes API (CRs / Leases), not in a new datastore, and
prefer informers over polling.** This keeps the "CRs are the source of truth" rule from `docs/ARCHITECTURE.md`.

### A. Move ephemeral objects to CRs (fixes #1, #2, #3)

- Load balancer family: the operator already owns the LB-related CRDs? **No.** Item 6 of the
  roadmap (CRD-first migration) covers moving `LoadBalancer` to a CRD + controller. Multi-replica
  for LBs therefore depends on that migration; until then, `api.replicas > 1` must be refused
  (see "Guard rail").
- Audit events: new namespaced CR `AuditEvent` is too heavy (high write rate, unbounded growth).
  Proposal: write audit events as Kubernetes `Event` objects plus structured JSON on stdout
  (`audit=true` field) and make the in-memory ring buffer a read cache only. The UI "recent
  activity" list reads Events. **Decision needed from the maintainer** (see questions).
- Jobs: either delete the `async` path (REST behavior change, needs approval) or persist
  `AsyncJob` as a CR and process it under a Lease (see C). Deleting is the smaller change
  because nothing processes jobs today.

### B. Caches and realtime (fixes #4, #5, #6, #9)

- **VM list cache:** when `operatorReconcile` is on, serve `ListVMs`/`GetVM` from the Instance
  informer's lister (event-driven, sub-second, already running in every replica) and drop the
  20 s cache for that mode. The cache stays for the legacy hypervisor path.
- **Hub:** no shared pub/sub is needed. Every replica runs the informer, so every replica
  observes every Instance change and fans out to its own clients. This only holds if the
  informer is enabled; with `replicas > 1` the informer becomes mandatory (guard rail).
  Remove REST-originated `broadcastVM` for CRD-first mode to avoid duplicate events (client sees
  informer event only). Redis/NATS pub/sub is rejected: new infrastructure for no gain.
- **Tenant/role caches:** shorten TTL to 5 s when `replicas > 1`, or back them with a tenant
  informer. Start with the informer for Tenant (cheap, same pattern).

### C. Leader election (fixes #10)

Use `k8s.io/client-go/tools/leaderelection` with a `coordination.k8s.io/Lease` named
`virtfoundry-api-leader` in `virtfoundry-system`. Only loops that **write** need it:

| Loop | Writes? | Needs leader? |
|------|---------|---------------|
| `SyncAllVMStates` (CRD-first) | No (read + local broadcast) | No, runs on every replica (after B it reads the lister) |
| `SyncAllVMStates` (hypervisor mode) | Cache only | No |
| `ReconcileAll` | Yes (`SaveVM`, marks `Destroyed`) | **Yes** |
| Job processor (if kept) | Yes | **Yes** |
| ISO import goroutine | DataVolume create, per request | No (request-scoped; retried by the next deploy) |

Chart: add `Role` rule on `leases` (`coordination.k8s.io`, verbs get/create/update) to the API
Role in `templates/rbac.yaml` (rule goes in the chart first, per the RBAC ordering rule).

### D. Security-relevant counters (fixes #7, #8)

- **Console tickets:** keep HMAC tickets. Add optional strict single use: on redeem, create a
  `Lease` named `ct-<sha256(jti)[:16]>`; `AlreadyExists` means replay → reject. One extra API
  write per console connection, a janitor deletes leases older than the TTL. Gate with
  `consoleTicket.strictSingleUse` (default `false`, so single-replica installs pay nothing).
- **Login throttle:** accept the ×N bound for `replicas ≤ 3` and document it; set
  `sessionAffinity: ClientIP` on the API Service as a cheap mitigation (effective only when the
  ingress preserves client IP). A shared counter (Lease/CR per key) is possible later but a
  write per failed login is a DoS amplifier, so it is not proposed now.

### Guard rail

At startup, if `replicas > 1` is requested the chart renders `api.replicas` normally, but the API
checks `VIRTFOUNDRY_REPLICAS_SAFE` (set by the chart only when the prerequisites below hold) and
**refuses to start** with a clear message if it detects another live leader Lease holder while
ephemeral LB/audit/job state is still in memory. Simpler variant (preferred): the chart fails
`helm template` when `api.replicas > 1` and `api.multiReplica.enabled` is false. Each phase below
flips what `multiReplica.enabled` is allowed to turn on.

## Rollout (one PR each, in order)

1. **core:** remove the async job path or persist it (decision Q2); add the `Lease` leader election helper
   and wrap `ReconcileAll`. No behavior change with one replica.
2. **core:** serve `ListVMs`/`GetVM` from the Instance lister in CRD-first mode; Tenant informer.
3. **core + operator (roadmap item 6):** LoadBalancer family to CRDs. Audit events to Events + stdout.
4. **helm-charts:** `leases` RBAC rule, `PodDisruptionBudget` (`minAvailable: 1`, rendered only when
   `replicas > 1`), `topologySpreadConstraints` default, `sessionAffinity: ClientIP` option,
   `api.multiReplica.enabled` guard, `replicas > 1` allowed once 1-3 have landed.
5. **core:** optional strict single-use console tickets (Q3).
6. **e2e:** extend `operator/hack/e2e-kubevirt.sh` to run with `api.replicas=2` and assert: a VM
   deployed via replica A is visible via replica B immediately; kill the leader and see another take over.

## What to test on the homelab

- `replicas: 2`: deploy a VM, list from both pods (`kubectl port-forward` to each) within 1 s.
- Delete the leader pod: `ReconcileAll` resumes on the other pod within the Lease duration (15 s).
- WebSocket client on pod B receives `vm.updated` for a change made through pod A.
- Rolling update with `maxUnavailable: 0`: no failed `/health` and no dropped console session beyond the pod being replaced.

## Questions for the maintainer

1. **LB and audit blockers:** OK to require roadmap item 6 (LoadBalancer CRD) before allowing `replicas > 1`, instead of shipping a half-working multi-replica mode?
2. **Async jobs:** delete the `async: true` deploy path (nothing processes it today; REST behavior change) or persist and process it behind the leader Lease?
3. **Console tickets:** is the documented 30 s replay window acceptable, or do you want strict single use (extra Lease write per console connect)?
4. **Audit log:** Kubernetes `Event` + stdout acceptable as the audit store, or do you want an `AuditEvent` CR with retention?
