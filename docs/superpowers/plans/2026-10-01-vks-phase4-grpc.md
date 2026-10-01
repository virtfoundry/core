# VKS Phase 4 — gRPC ClusterService + REST shim + UI

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (inline) or subagent-driven-development. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Tenants create/list/delete VKS clusters and download kubeconfig via gRPC-canonical `ClusterService`, with a thin REST shim + UI page.

**Architecture:** `internal/service/vks` owns CRD CRUD on `VKSCluster` (unstructured + GVR). `ClusterService` is the product contract on cmux `:8080`. REST `/api/v1/vks/clusters*` is a 1:1 adapter for today's UI. Kubeconfig reads tenant-ns Secret `{name}-admin-kubeconfig` key `admin.conf` (copied by vks operator; also in `status.kubeconfigSecretRef`).

**Tech Stack:** Go, protobuf/grpc-go, cmux, dynamic client, React + TanStack Query, i18n.

**Spec:** `docs/superpowers/specs/2026-10-01-vks-phase4-grpc-design.md`

## Global Constraints

- core never creates TCP/Instance for VKS — only `VKSCluster` CR
- gRPC = canonical; REST = transitional shim (no duplicated business logic)
- Auth gRPC: Bearer + `x-tenant-id` fail-closed (same as InstanceService)
- Perms: `vks:read`, `vks:write`, `vks:kubeconfig`
- Ship: PR → merge → digest write-back → Argo (no kubectl image patch)
- Unstructured GVR (do not vendor `virtfoundry/vks` module yet)

## File map

| Path | Role |
|------|------|
| `api/proto/virtfoundry/vks/v1alpha1/cluster.proto` | Contract |
| `api/gen/virtfoundry/vks/v1alpha1/*` | Generated |
| `scripts/generate-proto.sh` | Add cluster.proto |
| `internal/platform/store/mapping/gvr.go` | `VKSClusterGVR` |
| `internal/service/vks/*.go` | Domain service |
| `internal/api/grpc/cluster.go` + tests | gRPC server |
| `internal/api/grpc/server.go` | Register ClusterService |
| `internal/api/handler/vks_handler.go` | REST shim |
| `internal/auth/permissions.go` | Perms + role lists |
| `internal/api/middleware/auto_permission.go` | Map `vks` |
| `cmd/server/main.go` | Wire service + routes + gRPC backend |
| `ui/src/pages/Clusters.tsx` + nav + i18n + api client | UI |

---

### Task 1: Proto + codegen

**Files:**
- Create: `api/proto/virtfoundry/vks/v1alpha1/cluster.proto`
- Modify: `scripts/generate-proto.sh`
- Generate: `api/gen/virtfoundry/vks/v1alpha1/`

- [ ] **Step 1:** Write full proto (messages + service) matching the spec sketch; include all request/response types (`ListClustersRequest`, etc.).
- [ ] **Step 2:** Update `generate-proto.sh` to also compile `cluster.proto`.
- [ ] **Step 3:** Run `./scripts/generate-proto.sh` (or install plugins into `.tools` if missing).
- [ ] **Step 4:** Commit `feat(api): add vks ClusterService proto`

---

### Task 2: Domain service `internal/service/vks`

**Files:**
- Create: `internal/platform/store/mapping/gvr.go` — add `VKSClusterGVR = {Group:virtfoundry.io, Version:v1alpha1, Resource:vksclusters}`
- Create: `internal/service/vks/types.go` — `Cluster` struct (name, tenantID, namespace, version, controlPlane, workers, phase, endpoint, readyWorkers, kubeconfigSecretRef)
- Create: `internal/service/vks/service.go` — `Service` with `dynamic.Interface` + `store.Repository` (for `shared.TenantNamespace`)
- Create: `internal/service/vks/service_test.go`

**Interfaces:**
```go
type CreateInput struct {
  Name, KubernetesVersion string
  ControlPlane ControlPlaneSpec // ServiceType, Address, Port
  Workers WorkersSpec           // Count, TemplateRef, OfferingRef, NetworkRef, SSHKeyRefs
}

func (s *Service) List(ctx context.Context, tenantID string) ([]Cluster, error)
func (s *Service) Get(ctx context.Context, tenantID, name string) (*Cluster, error)
func (s *Service) Create(ctx context.Context, tenantID string, in CreateInput) (*Cluster, error)
func (s *Service) Delete(ctx context.Context, tenantID, name string) error
func (s *Service) GetKubeconfig(ctx context.Context, tenantID, name string) ([]byte, error)
```

**Behavior:**
- Create validates DNS-1123 name, workers 1–3, non-empty version + three refs; builds unstructured `VKSCluster` in tenant ns; Create via dynamic client.
- GetKubeconfig: Get cluster; if `status.kubeconfigSecretRef` empty → conflict/not ready; Get Secret in tenant ns; return `data["admin.conf"]` (never log).
- Watch helper optional in Task 3 (poll list every 2s in stream is OK for MVP).

- [ ] **Step 1:** Failing tests — invalid workers count; missing ref; kubeconfig when no secret.
- [ ] **Step 2:** Implement service + GVR.
- [ ] **Step 3:** `go test ./internal/service/vks/...`
- [ ] **Step 4:** Commit `feat(vks): add CRD-first cluster service`

---

### Task 3: IAM + AutoPermission

**Files:**
- Modify: `internal/auth/permissions.go` — add `PermVKSRead/Write/Kubeconfig`; append to TenantAdmin + Operator; Viewer gets read only (not kubeconfig).
- Modify: `internal/api/middleware/auto_permission.go` — `"vks": "vks"`.
- Modify: `internal/api/middleware/auto_permission_test.go` — allow GET `/api/v1/vks/clusters`.
- Kubeconfig route uses **separate** `RequirePermission(PermVKSKubeconfig)` subrouter (like console), so GET does not grant download via `:read` alone.

- [ ] **Step 1:** Tests for perm map + HasPermission.
- [ ] **Step 2:** Implement.
- [ ] **Step 3:** Commit `feat(auth): add vks IAM permissions`

---

### Task 4: gRPC `ClusterService`

**Files:**
- Create: `internal/api/grpc/cluster.go`
- Create: `internal/api/grpc/cluster_test.go`
- Modify: `internal/api/grpc/server.go` — `NewGRPCServer` accepts `ClusterBackend` (or `*vks.Service`), register `RegisterClusterServiceServer`
- Modify: `DualStackOptions` + `cmd/server/main.go` wiring

**RPC → perm:**
- List/Get/Watch → `vks:read`
- Create/Delete → `vks:write`
- GetKubeconfig → `vks:kubeconfig`

Watch MVP: initial List snapshot as MODIFIED, then ticker re-list and diff ADDED/MODIFIED/DELETED (no hub required for Phase 4).

- [ ] **Step 1:** Unit test List requires `vks:read`.
- [ ] **Step 2:** Implement server + wire cmux.
- [ ] **Step 3:** `go test ./internal/api/grpc/...`
- [ ] **Step 4:** Commit `feat(grpc): register VKS ClusterService`

---

### Task 5: REST shim

**Files:**
- Create: `internal/api/handler/vks_handler.go`
- Modify: `cmd/server/main.go` — routes under `/vks/clusters`; kubeconfig on dedicated subrouter with `RequirePermission(PermVKSKubeconfig)`

JSON create body mirrors CreateInput field names (snake_case JSON tags consistent with other handlers).

- [ ] **Step 1:** Handler tests with httptest + fake service if easy; else table-test decode.
- [ ] **Step 2:** Wire routes.
- [ ] **Step 3:** Commit `feat(api): add REST shim for VKS clusters`

---

### Task 6: UI Clusters page

**Files:**
- Create: `ui/src/pages/Clusters.tsx`
- Modify: `ui/src/App.tsx` (route), `SidebarNav.tsx`, `lib/i18n.tsx` (pt/en), API helper (`lib/api.ts` or equivalent)

UI defaults (form only): `kubernetes_version=v1.36.5`, template `ubuntu-node-1-36-5`, offering `medium`, network `default`, workers=1. Control plane address/port left empty (operator defaults) unless advanced fields shown.

- [ ] **Step 1:** Page list + create dialog + delete + download kubeconfig (blob).
- [ ] **Step 2:** i18n keys.
- [ ] **Step 3:** `npm test` / typecheck as repo requires.
- [ ] **Step 4:** Commit `feat(ui): add VKS Clusters page`

---

### Task 7: Docs + PR + ship

- [ ] Update `docs/GRPC-SPIKE.md` or add `docs/VKS-API.md` noting ClusterService is **canonical** (not spike).
- [ ] Update spec status → Implementing/Done.
- [ ] PR on `virtfoundry/core`; merge; wait digest write-back; smoke:
  - REST create/list/kubeconfig with JWT
  - UI create → Ready → download
  - Optional grpcurl ListClusters

---

## Spec coverage check

| Spec item | Task |
|-----------|------|
| Proto ClusterService | 1 |
| service/vks CRD-first | 2 |
| IAM vks:* | 3 |
| gRPC on cmux | 4 |
| REST shim | 5 |
| UI + i18n | 6 |
| Homelab digests / smoke | 7 |
| Watch | 4 (poll diff MVP) |
| Kubeconfig Secret in tenant ns | 2 + 5 |

## Out of scope

Phase 5 quota; grpc-web UI; TF gRPC; Ingress gRPC; vendoring vks Go module.
