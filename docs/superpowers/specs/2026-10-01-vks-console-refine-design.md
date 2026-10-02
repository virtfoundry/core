# VKS console refine + platform packaging

**Date:** 2026-10-01  
**Status:** Approved — plan `docs/superpowers/plans/2026-10-01-vks-console-refine.md`  
**Repos:** `virtfoundry/core` (API/UI), `virtfoundry/terraform-provider-virtfoundry`, `virtfoundry/helm-charts` (docs + future umbrella), `virtfoundry/vks-image-factory` (matrix), `Matheus-Thurler/argo-homelab` (later App migration only)  
**Related:** Phase 4 `2026-10-01-vks-phase4-grpc-design.md` (shipped), `vks-image-factory/VERSIONS.md`, CRD `VKSCluster`

## Problem

Phase 4 shipped list/create/detail with free-text create fields and IaaS-only cluster metadata. Users need GKE-like create (catalog pickers), worker Instance visibility, guest workload summary, Terraform parity, a clear version/compatibility matrix, and a documented path to a single Helm install — without breaking the 0.9.x three-App Argo layout.

## Goals

1. **Create UX:** network / K8s version / SSH (and offering/template) from catalogs; version = official matrix ∩ templates with published node images.
2. **Nodes tab:** list worker Instances for the cluster; link to existing VM detail.
3. **Guest summary (C→A):** when Ready (or CP ready enough), read-only namespaces + pod counts (+ guest nodes) via server-side kubeconfig; IAM `vks:read`.
4. **Helm:** keep 0.9.x as three charts/Apps; document future umbrella chart via Helm dependencies in a **separate** doc; link from versioning.
5. **Terraform:** `virtfoundry_vks_cluster` mirroring Create/Get/Delete (+ attrs); examples.
6. **Compatibility matrix:** expand official list (node tag, template seed, Kamaji max, core/TF/chart pins) for each bump.

## Non-goals

- Generic guest API proxy, exec, log stream, apply (Phase 5+).
- Shipping umbrella chart / collapsing Argo Apps in this iteration.
- Browser → guest kube-apiserver.
- New VKS catalog gRPC in MVP (reuse IaaS list endpoints; optional `ListClusterOptions` later).
- Changing CRD worker naming/labels (already `{cluster}-worker-{i}` + `vks.virtfoundry.io/cluster`).

## Decisions (from dialogue)

| Topic | Decision |
|-------|----------|
| Guest pods/namespaces | **C→A**: summary when Ready; banner otherwise; not B-only |
| Create catalogs | Client-side intersection of VERSIONS + templates; no new options RPC in MVP |
| Helm unify | Doc + future `virtfoundry-platform` deps; **0.9.x stays split** |
| TF | New resource aligned to ClusterService |

## Architecture

```
UI create ──► listNetworks / listSSHKeys / listVMTemplates / listOfferings
         └──► CreateCluster (existing)

UI Nodes ──► list VMs / Instances filtered by label vks.virtfoundry.io/cluster
         └──► Link /vms/:name

UI Workloads ──► GetClusterSummary ──► core opens admin kubeconfig Secret
                                    └── client-go List ns/pods/nodes (timeout)

TF virtfoundry_vks_cluster ──► same ClusterService / REST
```

### Section 1 — Create + Nodes

**Create form**

| Field | Behavior |
|-------|----------|
| Network | `<select>` from `listNetworks`; empty → `default` |
| Kubernetes version | Options = intersection of (a) versions in `vks-image-factory/VERSIONS.md` / published node tags and (b) tenant templates matching node image (e.g. `ubuntu-node-1-36-5` ↔ `v1.36.5`). One → preselect; many → dropdown; none → block create + message |
| Template | Bound to chosen version (node image template) |
| Offering | IaaS offerings list; default `medium` if present |
| SSH keys | Multi-select → `workers.ssh_key_refs` (already in proto) |
| NodePort | Optional (unchanged) |

**Nodes tab**

- Resolve workers via platform VM/Instance list filtered by `vks.virtfoundry.io/cluster=<name>` (or name prefix `{cluster}-worker-` if labels not on list DTO — prefer labels; extend list DTO if needed).
- Columns: name, phase, offering, IP if available.
- Row link → `/vms/:name`.

### Section 2 — Guest summary

**RPC:** `GetClusterSummary(name)` on `ClusterService` (+ REST `GET /api/v1/vks/clusters/{name}/summary`).

**Payload (MVP):**

- `namespaces[]`: name, pod_count
- `pod_totals`: running / pending / failed / other
- `guest_nodes[]`: name, ready (bool) — complementary to IaaS worker Instances
- errors: clear message if kubeconfig missing or guest API unreachable

**Guards:** IAM `vks:read`; server-only Secret read; short timeout (5–10s); no arbitrary path proxy.

**UI:** tab **Workloads** (or Overview section); gated on Ready / best-effort on ControlPlaneReady.

### Section 3 — Helm umbrella (future)

**0.9.x unchanged:** Argo Apps `virtfoundry-operator` → `virtfoundry` → `virtfoundry-vks`; install docs stay operator-first.

**Future chart** `virtfoundry-platform` (name locked in umbrella doc): Helm dependencies on the three charts; nested values; CRDs remain on operator chart.

**Ship now:** `helm-charts/docs/project/umbrella-chart.md` + one-line pointer from `versioning.md`. Do **not** add the chart or change Argo Apps in this iteration.

### Section 4 — Terraform + matrix

**Resource** `virtfoundry_vks_cluster`:

- Required/optional args mirror `CreateClusterRequest`
- Computed: phase, control_plane_endpoint, ready_workers, namespace, etc.
- Kubeconfig: sensitive attribute or separate data source (prefer data source `virtfoundry_vks_kubeconfig` to avoid state bloat — decide in plan; default **data source**)

**Matrix:** keep `vks-image-factory/VERSIONS.md` as pin source; add `COMPATIBILITY.md` (or section) mapping product release ↔ node tag/digest ↔ template name ↔ Kamaji max ↔ recommended core/helm/TF versions. Link from TF README + helm versioning.

## Shipping

- Conventional Commits; PR → merge → digest write-back → Argo (core/UI); TF Registry release when provider ships; helm-charts docs-only PR for umbrella + links.
- Smoke: create with network+SSH pickers; Nodes tab shows workers; Workloads tab on Ready `demo`; TF apply example against homelab optional.

## Open points for plan (not blockers)

- Exact template↔version naming convention helper (single shared TS/Go mapping).
- Whether VM list API already returns VKS labels (extend if not).
- TF kubeconfig as attribute vs data source (recommend data source).
