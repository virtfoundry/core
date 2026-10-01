# VKS Phase 4 — gRPC-canonical Cluster API + UI

**Date:** 2026-10-01  
**Status:** Approved — implementing (plan `docs/superpowers/plans/2026-10-01-vks-phase4-grpc.md`)  
**Repos:** `virtfoundry/core` (service + gRPC + REST shim + UI), consumes CRD from `virtfoundry/vks`  
**Related:** `docs/agent-local/2026-10-01-design-vks.md`, VKS guia Phase 4, `docs/GRPC-SPIKE.md` (InstanceService), CRD `virtfoundry.io/v1alpha1.VKSCluster`

## Problem

Phase 3 delivers `VKSCluster` via kubectl only. Tenants cannot create clusters or download kubeconfig from `http://virtfoundry.homelab`. IaaS already has a gRPC spike (`InstanceService`); VKS must not repeat “REST-first forever” — gRPC is the product contract from day one, with REST as a transitional UI adapter.

## Goals

1. **Canonical API:** `virtfoundry.vks.v1alpha1.ClusterService` (List/Get/Create/Delete/Watch/GetKubeconfig) on cmux `:8080`.
2. **Single domain logic:** `internal/service/vks` CRD-first (create/list/get/delete `VKSCluster` in tenant ns; kubeconfig from Kamaji/admin Secret referenced by status).
3. **REST shim:** `/api/v1/vks/clusters*` maps 1:1 onto the same service (UI + curl smoke today).
4. **IAM:** `vks:read`, `vks:write`, `vks:kubeconfig` (root bypass; AutoPermission parity with other resources).
5. **UI:** Clusters page — list, create (form defaults), phase/status, delete, download kubeconfig.
6. Ship via PR → merge → digest write-back → Argo; smoke on homelab.

## Non-goals (this phase)

- Quota / cascade hardening docs (Phase 5).
- Migrating IaaS UI or Terraform to gRPC.
- Public Ingress / Gateway API for gRPC (stay cmux ClusterIP like Instance spike).
- Browser talking to kube-apiserver.
- core talking CAPI/Kamaji directly (only `VKSCluster` CR + reading kubeconfig Secret).
- Hiding CR fields behind homelab-only defaults in the service layer.

## Approaches considered

| # | Approach | Pros | Cons |
|---|----------|------|------|
| A | REST-only Phase 4; gRPC later | Fastest UI | Wrong habit; second rewrite |
| B | gRPC-only (no REST); UI via grpc-web now | One protocol | Browser tooling + bigger UI change |
| C | **Domain service + gRPC canonical + thin REST shim** | Correct contract; UI ships now; migrate later | Two transports briefly |

**Decision:** **C**.

## Architecture

```
UI (fetch REST) ──► REST /api/v1/vks/*  ──┐
                                          ├──► internal/service/vks ──► VKSCluster CR
grpcurl / future UI gRPC ──► ClusterService ──┘         │
                                                         ▼
                                              virtfoundry/vks operator
                                              (Kamaji TCP + Instances + Flannel)
```

- **Source of truth:** `VKSCluster` CR (`virtfoundry.io`).
- **core never** creates TenantControlPlane or Instance for VKS workers (operator does).
- **Kubeconfig:** read Secret named by `status.kubeconfigSecretRef` (or Kamaji `*-admin-kubeconfig` in TCP ns / copied into tenant ns — implement against live Phase 3 status fields). Do not log contents.

## Proto sketch (`api/proto/virtfoundry/vks/v1alpha1/cluster.proto`)

```protobuf
syntax = "proto3";
package virtfoundry.vks.v1alpha1;
option go_package = "github.com/virtfoundry/core/api/gen/virtfoundry/vks/v1alpha1;vksv1alpha1";

message LocalObjectRef { string name = 1; }

message WorkersSpec {
  int32 count = 1;                 // 1–3
  LocalObjectRef template_ref = 2;
  LocalObjectRef offering_ref = 3;
  LocalObjectRef network_ref = 4;
  repeated LocalObjectRef ssh_key_refs = 5;
}

message ControlPlaneSpec {
  string service_type = 1; // NodePort
  string address = 2;      // optional; empty → operator default
  int32 port = 3;          // optional NodePort
}

message Cluster {
  string name = 1;
  string tenant_id = 2;
  string namespace = 3;
  string kubernetes_version = 4;
  ControlPlaneSpec control_plane = 5;
  WorkersSpec workers = 6;
  string phase = 7;
  string control_plane_endpoint = 8;
  int32 ready_workers = 9;
  string kubeconfig_secret_ref = 10;
  // conditions as repeated Condition if needed for UI
}

message CreateClusterRequest {
  string name = 1;
  string kubernetes_version = 2;
  ControlPlaneSpec control_plane = 3;
  WorkersSpec workers = 4;
}

message GetKubeconfigRequest { string name = 1; }
message GetKubeconfigResponse {
  bytes kubeconfig = 1; // admin.conf YAML
}

service ClusterService {
  rpc ListClusters(ListClustersRequest) returns (ListClustersResponse);
  rpc GetCluster(GetClusterRequest) returns (GetClusterResponse);
  rpc CreateCluster(CreateClusterRequest) returns (CreateClusterResponse);
  rpc DeleteCluster(DeleteClusterRequest) returns (DeleteClusterResponse);
  rpc WatchClusters(WatchClustersRequest) returns (stream WatchClustersResponse);
  rpc GetKubeconfig(GetKubeconfigRequest) returns (GetKubeconfigResponse);
}
```

Auth metadata (same as Instance spike): `authorization: Bearer …`, `x-tenant-id` for root.

## REST shim (transitional)

| Method | Path | Perm | Service |
|--------|------|------|---------|
| GET | `/api/v1/vks/clusters` | `vks:read` | List |
| GET | `/api/v1/vks/clusters/{name}` | `vks:read` | Get |
| POST | `/api/v1/vks/clusters` | `vks:write` | Create (JSON ≈ CreateClusterRequest) |
| DELETE | `/api/v1/vks/clusters/{name}` | `vks:write` | Delete |
| GET | `/api/v1/vks/clusters/{name}/kubeconfig` | `vks:kubeconfig` | GetKubeconfig (attachment / raw YAML) |

No dual business logic in handlers.

## IAM

Add to `internal/auth/permissions.go` and role maps (admin/operator/viewer as appropriate):

- `vks:read` — list/get/watch
- `vks:write` — create/delete
- `vks:kubeconfig` — download (narrower than write)

Wire AutoPermission / RequirePermission like networks/vms.

## Service (`internal/service/vks`)

- Resolve tenant → `virtfoundry-tenant-<slug>` namespace.
- Create: build `VKSCluster` unstructured or typed client; validate name DNS-1123; workers 1–3; require template/offering/network refs.
- List/Get: from Kubernetes API (or store if kubernetes store already lists custom types — prefer direct client for VKSCluster only).
- Delete: delete CR; rely on vks finalizer for cascade.
- GetKubeconfig: follow `status.kubeconfigSecretRef` / Phase 3 Secret layout; return `admin.conf` bytes; 404/409 if not Ready.
- Watch: informer or poll+hub events `vks.*` — MVP may poll CR list on interval inside stream or reuse hub if we emit events on reconcile-visible updates; prefer watch on VKSCluster GVR when feasible.

## UI

- Nav entry **Clusters** (or **Kubernetes**).
- List table: name, phase, endpoint, workers, actions (kubeconfig, delete).
- Create dialog: name, workers count, version + template/offering/network selectors (defaults prefilled for homelab).
- i18n pt/en.
- Call REST shim only in this ship (no grpc-web yet).

## Migration path (post–Phase 4)

1. Keep REST shim until UI (or grpc-web) speaks `ClusterService`.
2. Deprecate REST in changelog when UI migrates.
3. Terraform provider later: gRPC or keep REST until TF gRPC support is desired.
4. Optionally align InstanceService from “spike” → same “canonical gRPC” policy.

## Test plan

- Unit: service create validation; perm denied; kubeconfig refuse when not Ready.
- gRPC: grpcurl List/Create/GetKubeconfig against cmux (in-cluster or port-forward).
- REST: curl smoke with JWT like Networks.
- UI: create → wait Ready → download kubeconfig → delete.
- Homelab: Argo digests for `core` + `ui`.

## Open points (resolve in plan, not blockers)

1. Exact Secret location for kubeconfig today (`status.kubeconfigSecretRef` vs `vks-default-demo/demo-admin-kubeconfig`) — implement against live CR status after Phase 3.
2. Watch transport MVP: K8s watch vs hub `vks.updated` — prefer K8s watch if client already in core.
3. Whether CRD Go types are vendored from `virtfoundry/vks` or duplicated as unstructured — prefer generated/client from published module or unstructured with GVR to avoid tight version pin pain.

## DoD checklist

- [ ] Proto + `./scripts/generate-proto.sh` committed under `api/gen/virtfoundry/vks/`
- [ ] `ClusterService` registered on cmux alongside InstanceService
- [ ] Service + REST shim + perms
- [ ] UI Clusters page i18n
- [ ] Homelab smoke PASS
- [ ] Digests pinned in argo-homelab
