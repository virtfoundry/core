# VKS Console Refine + Packaging Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship catalog-driven VKS create, worker Instance listing in Nodes, guest Workloads summary (C→A), umbrella Helm **docs only**, Terraform `virtfoundry_vks_cluster`, and an official compatibility matrix.

**Architecture:** UI reuses IaaS list APIs for create pickers; Nodes filters VMs by name prefix/`vks.virtfoundry.io/cluster`; `GetClusterSummary` builds a short-lived client-go from the admin kubeconfig Secret; Helm 0.9.x stays three Apps with a future umbrella doc; TF mirrors REST ClusterService.

**Tech Stack:** Go (core service/gRPC/REST), React+TanStack Query (UI), Terraform Plugin Framework, Helm docs Markdown, `vks-image-factory` matrix Markdown.

**Spec:** `core/docs/superpowers/specs/2026-10-01-vks-console-refine-design.md`

## Global Constraints

- gRPC is canonical; REST is a thin shim; UI may use REST.
- Ship via PR → merge → digest write-back → Argo; no `:latest` / kubectl image patch.
- Guest access: server-side kubeconfig only; IAM `vks:read`; timeout 5–10s; no proxy/exec/apply.
- Helm umbrella chart is **not** shipped in this plan — documentation only; 0.9.x install path unchanged.
- Conventional Commits; scopes `feat(ui)`, `feat(vks)`, `feat(tf)`, `docs(helm)`.
- Prefer separate PRs per wave (A core UI → B summary → C helm docs → D TF+matrix) so each is reviewable.

## File map

| Path | Role |
|------|------|
| `core/ui/src/lib/vks-catalog.ts` | Map node templates ↔ `v1.x.y`; intersect with published versions |
| `core/ui/src/pages/Clusters.tsx` | Create form selects (network, version, offering, SSH) |
| `core/ui/src/pages/VKSClusterDetail.tsx` | Nodes list + Workloads tab |
| `core/ui/src/lib/platform-api.ts` | `getVKSClusterSummary`; VM filter helpers if needed |
| `core/ui/src/lib/i18n.tsx` | New keys pt/en |
| `core/api/proto/.../cluster.proto` | `GetClusterSummary` RPC + messages |
| `core/internal/service/vks/summary.go` | Build summary from guest kubeconfig |
| `core/internal/api/grpc/cluster.go` | Wire summary RPC |
| `core/internal/api/handler/vks_handler.go` | `GET .../summary` |
| `helm-charts/docs/project/umbrella-chart.md` | Future deps chart |
| `helm-charts/docs/project/versioning.md` | Link to umbrella |
| `vks-image-factory/COMPATIBILITY.md` | Product ↔ node ↔ template matrix |
| `vks-image-factory/VERSIONS.md` | Keep as pin source; link COMPATIBILITY |
| `terraform-provider-virtfoundry/internal/virtfoundry/vks.go` | HTTP client for clusters |
| `terraform-provider-virtfoundry/internal/provider/vks_cluster_resource.go` | Resource |
| `terraform-provider-virtfoundry/internal/provider/vks_kubeconfig_data_source.go` | Sensitive kubeconfig |
| `terraform-provider-virtfoundry/examples/vks-cluster/` | Example |

---

## Wave A — Create catalogs + Nodes tab (core)

### Task 1: `vks-catalog` helper + unit tests

**Files:**
- Create: `core/ui/src/lib/vks-catalog.ts`
- Create: `core/ui/src/lib/vks-catalog.test.ts`

**Interfaces:**
- Produces: `PUBLISHED_NODE_VERSIONS: { k8s: string; templatePrefix: string }[]`, `parseTemplateK8sVersion(templateName: string): string | null`, `resolveVKSVersionOptions(templates: { name: string }[]): { kubernetes_version: string; template: string }[]`

- [ ] **Step 1: Write failing tests**

```ts
import { describe, expect, it } from 'vitest';
import { parseTemplateK8sVersion, resolveVKSVersionOptions } from './vks-catalog';

describe('parseTemplateK8sVersion', () => {
  it('parses ubuntu-node-1-36-5', () => {
    expect(parseTemplateK8sVersion('ubuntu-node-1-36-5')).toBe('v1.36.5');
  });
  it('returns null for unrelated templates', () => {
    expect(parseTemplateK8sVersion('ubuntu-2204')).toBeNull();
  });
});

describe('resolveVKSVersionOptions', () => {
  it('intersects published pins with templates', () => {
    const opts = resolveVKSVersionOptions([
      { name: 'ubuntu-node-1-36-5' },
      { name: 'ubuntu-2204' },
    ]);
    expect(opts).toEqual([{ kubernetes_version: 'v1.36.5', template: 'ubuntu-node-1-36-5' }]);
  });
});
```

- [ ] **Step 2: Run tests — expect FAIL**

Run: `cd core/ui && npx vitest run src/lib/vks-catalog.test.ts`  
Expected: FAIL module not found

- [ ] **Step 3: Implement**

```ts
/** Keep in sync with vks-image-factory/VERSIONS.md published node tags. */
export const PUBLISHED_NODE_VERSIONS = [
  { k8s: 'v1.36.5', templateHint: 'ubuntu-node-1-36-5' },
] as const;

const NODE_RE = /^ubuntu-node-(\d+)-(\d+)-(\d+)$/;

export function parseTemplateK8sVersion(templateName: string): string | null {
  const m = NODE_RE.exec(templateName);
  if (!m) return null;
  return `v${m[1]}.${m[2]}.${m[3]}`;
}

export function resolveVKSVersionOptions(
  templates: Array<{ name: string }>,
): Array<{ kubernetes_version: string; template: string }> {
  const byVersion = new Map<string, string>();
  for (const t of templates) {
    const ver = parseTemplateK8sVersion(t.name);
    if (!ver) continue;
    if (!PUBLISHED_NODE_VERSIONS.some((p) => p.k8s === ver)) continue;
    if (!byVersion.has(ver)) byVersion.set(ver, t.name);
  }
  return [...byVersion.entries()]
    .sort(([a], [b]) => b.localeCompare(a, undefined, { numeric: true }))
    .map(([kubernetes_version, template]) => ({ kubernetes_version, template }));
}
```

- [ ] **Step 4: Run tests — expect PASS**

- [ ] **Step 5: Commit** `feat(ui): add vks version/template catalog helper`

---

### Task 2: Create form — network, version, offering, SSH

**Files:**
- Modify: `core/ui/src/pages/Clusters.tsx`
- Modify: `core/ui/src/lib/i18n.tsx` (keys under `vks.form.*`)

**Interfaces:**
- Consumes: `listNetworks`, `listSSHKeys`, `listVMTemplates`, `listServiceOfferings`, `resolveVKSVersionOptions`
- Produces: create payload with `workers.ssh_key_refs`, selected network/template/version

- [ ] **Step 1: Load catalogs in Clusters page**

Use `useQuery` (enabled when create modal open or always with clusters list):

```ts
const { data: networksData } = useQuery({ queryKey: queryKeys.networks, queryFn: listNetworks, enabled: !needsTenant });
const { data: templatesData } = useQuery({ queryKey: queryKeys.templates, queryFn: listVMTemplates, enabled: !needsTenant });
const { data: offeringsData } = useQuery({ queryKey: queryKeys.offerings /* existing key */, queryFn: listServiceOfferings, enabled: !needsTenant });
const { data: sshData } = useQuery({ queryKey: queryKeys.sshKeys, queryFn: listSSHKeys, enabled: !needsTenant });
```

(Use existing `queryKeys` entries; add if missing.)

- [ ] **Step 2: Replace free-text fields with selects**

- Network: options from `networks`; if length 0, value `default` (hidden or single option).
- Version: from `resolveVKSVersionOptions(templates)`; on change set `kubernetes_version` + `template`.
- Offering: select; prefer `medium` default if present.
- SSH: multi-select checkboxes → `ssh_key_refs: selected.map(n => ({ name: n }))`.
- If version options empty: disable submit + `t('vks.form.noNodeImage')`.

- [ ] **Step 3: i18n** pt/en for new strings (`noNodeImage`, `sshKeys`, select placeholders).

- [ ] **Step 4: Manual check** `npm run build` in `core/ui`.

- [ ] **Step 5: Commit** `feat(ui): catalog-driven VKS cluster create form`

---

### Task 3: Nodes tab — list worker VMs + link

**Files:**
- Modify: `core/ui/src/pages/VKSClusterDetail.tsx`
- Modify: `core/ui/src/lib/vks-display.ts` (optional filter helper)
- Modify: `core/ui/src/lib/i18n.tsx`

**Interfaces:**
- Consumes: `listVMs()` → `PlatformVM[]` with `name`, `status`/`phase`, `service_offering_id` / offering fields, IP fields as already on DTO
- Filter: `vm.name.startsWith(`${cluster.name}-worker-`)` (matches operator naming). Prefer label filter if/when DTO exposes labels.

- [ ] **Step 1: Add helper**

```ts
export function isVKSWorkerVM(clusterName: string, vmName: string): boolean {
  return vmName.startsWith(`${clusterName}-worker-`);
}
```

- [ ] **Step 2: In Nodes tab**, `useQuery(listVMs)` filtered by helper; table with Link to `/vms/${encodeURIComponent(vm.name)}`; show phase + offering.

- [ ] **Step 3: Keep pool metadata card** (count/template/offering/network/ssh from cluster CR) above the Instance table.

- [ ] **Step 4: Build + commit** `feat(ui): list VKS worker VMs on cluster Nodes tab`

---

## Wave B — Guest summary API + Workloads UI

### Task 4: Proto `GetClusterSummary`

**Files:**
- Modify: `core/api/proto/virtfoundry/vks/v1alpha1/cluster.proto`
- Regenerate: `./scripts/generate-proto.sh` → `api/gen/...`

**Interfaces:**
- Produces RPC:

```protobuf
message ClusterSummary {
  repeated NamespaceSummary namespaces = 1;
  PodTotals pod_totals = 2;
  repeated GuestNode guest_nodes = 3;
  string message = 4; // optional soft warning
}
message NamespaceSummary {
  string name = 1;
  int32 pod_count = 2;
}
message PodTotals {
  int32 running = 1;
  int32 pending = 2;
  int32 failed = 3;
  int32 other = 4;
}
message GuestNode {
  string name = 1;
  bool ready = 2;
}
message GetClusterSummaryRequest { string name = 1; }
message GetClusterSummaryResponse { ClusterSummary summary = 1; }

// on ClusterService:
rpc GetClusterSummary(GetClusterSummaryRequest) returns (GetClusterSummaryResponse);
```

- [ ] **Step 1: Edit proto + run** `./scripts/generate-proto.sh`

- [ ] **Step 2: Commit** `feat(vks): add GetClusterSummary proto`

---

### Task 5: Domain `Summary` + tests

**Files:**
- Create: `core/internal/service/vks/summary.go`
- Modify: `core/internal/service/vks/service_test.go` (or `summary_test.go`)

**Interfaces:**
- Produces: `func (s *Service) GetSummary(ctx context.Context, tenantID, name string) (*Summary, error)`
- Types mirror proto JSON tags for REST

- [ ] **Step 1: Failing test** with fake clientset Secret + optional fake guest (or unit-test parsing helpers with mocked rest.Config via injectable `guestClientFor` function var).

Minimal approach: extract `buildSummary(ctx, clientset kubernetes.Interface) (*Summary, error)` and test with `fake.NewSimpleClientset` populated with Namespace + Pod + Node objects.

- [ ] **Step 2: Implement**

```go
// outline
func (s *Service) GetSummary(ctx context.Context, tenantID, name string) (*Summary, error) {
  raw, err := s.GetKubeconfig(ctx, tenantID, name) // same Ready/ControlPlaneReady gate
  // rest.Config from clientcmd.RESTConfigFromKubeConfig(raw)
  // kubernetes.NewForConfig with QPS limits
  // context.WithTimeout(ctx, 8*time.Second)
  // List namespaces, pods (all ns), nodes
  // aggregate
}
```

- [ ] **Step 3: Tests pass; commit** `feat(vks): implement guest cluster summary`

---

### Task 6: gRPC + REST shim

**Files:**
- Modify: `core/internal/api/grpc/cluster.go` — extend Backend interface + `GetClusterSummary`
- Modify: `core/internal/api/handler/vks_handler.go` — `GetClusterSummary`
- Modify: route registration (search `vks/clusters` in `platform_handler` / mux setup)

- [ ] **Step 1: Wire gRPC method** mapping domain → proto.

- [ ] **Step 2: REST** `GET /api/v1/vks/clusters/{name}/summary` → `{ "summary": ... }`; permission same as Get (`vks:read`).

- [ ] **Step 3: `go test ./internal/service/vks/... ./internal/api/...`

- [ ] **Step 4: Commit** `feat(vks): expose GetClusterSummary over gRPC and REST`

---

### Task 7: UI Workloads tab

**Files:**
- Modify: `core/ui/src/lib/platform-api.ts` — `getVKSClusterSummary(name)`
- Modify: `core/ui/src/lib/query-keys.ts`
- Modify: `core/ui/src/pages/VKSClusterDetail.tsx`
- Modify: `core/ui/src/lib/i18n.tsx`

- [ ] **Step 1: API client**

```ts
export async function getVKSClusterSummary(name: string) {
  return platformFetch<{ summary: VKSClusterSummary }>(
    `/vks/clusters/${encodeURIComponent(name)}/summary`,
  );
}
```

- [ ] **Step 2: Tab `workloads`** — enable query when phase is Ready or ControlPlaneReady; show namespace table + pod totals + guest nodes; on error show message + kubeconfig CTA.

- [ ] **Step 3: Build; commit** `feat(ui): VKS Workloads tab with guest summary`

- [ ] **Step 4: PR Wave A+B** (or A then B) → merge → Argo → smoke Ready cluster Workloads + create form

---

## Wave C — Helm umbrella docs

### Task 8: Umbrella chart documentation

**Files:**
- Create: `helm-charts/docs/project/umbrella-chart.md`
- Modify: `helm-charts/docs/project/versioning.md` — short “Future: umbrella” link

**Content must include:**
- 0.9.x remains three charts / three Argo Apps
- Future `virtfoundry-platform` Chart.yaml `dependencies:` on operator, virtfoundry, virtfoundry-vks
- Nested values keys
- CRD upgrade caveat unchanged
- Migration sketch: one Application → umbrella; delete old Apps after

- [ ] **Step 1: Write doc**
- [ ] **Step 2: Link from versioning.md**
- [ ] **Step 3: Commit + PR** `docs(helm): document future platform umbrella chart`

---

## Wave D — Terraform + compatibility matrix

### Task 9: Compatibility matrix

**Files:**
- Create: `vks-image-factory/COMPATIBILITY.md`
- Modify: `vks-image-factory/VERSIONS.md` — link at top
- Modify: `vks-image-factory/README.md` — link

Table columns: Product / Date | node-ubuntu tag + digest | template seed name | kubernetes_version string | Kamaji max | core/helm/TF notes

Seed row for current 1.36.5 publish from VERSIONS.md.

- [ ] **Step 1–3: Write, link, commit** `docs(vks): add official node/template compatibility matrix`

---

### Task 10: TF client + `virtfoundry_vks_cluster`

**Files:**
- Create: `terraform-provider-virtfoundry/internal/virtfoundry/vks.go`
- Create: `terraform-provider-virtfoundry/internal/provider/vks_cluster_resource.go`
- Create: `terraform-provider-virtfoundry/internal/provider/vks_kubeconfig_data_source.go`
- Modify: `terraform-provider-virtfoundry/internal/provider/provider.go` — register
- Create: `examples/vks-cluster/main.tf` + README
- Create: `docs/resources/vks_cluster.md`, `docs/data-sources/vks_kubeconfig.md`

**Schema (resource):**
- Required: `name`, `kubernetes_version`, `workers` nested (count, template_ref, offering_ref, network_ref, ssh_key_refs set)
- Optional: `control_plane` nested (service_type, address, port)
- Computed: `phase`, `control_plane_endpoint`, `ready_workers`, `namespace`, `id`

**Data source kubeconfig:** `name` → `kubeconfig` (sensitive) via GET kubeconfig endpoint.

- [ ] **Step 1: HTTP helpers** Create/Get/Delete/List mirroring `platform.go` style (`/api/v1/vks/clusters`).

- [ ] **Step 2: Resource CRUD** + Read refresh; Delete wait optional (simple delete OK).

- [ ] **Step 3: Kubeconfig data source**

- [ ] **Step 4: `go test ./...`**; example README

- [ ] **Step 5: Commit + PR** `feat(tf): add virtfoundry_vks_cluster resource`

---

## Spec coverage checklist

| Spec item | Task |
|-----------|------|
| Create network/version/SSH/offering catalogs | 2 |
| Version ∩ published node images | 1–2 |
| Nodes = worker VMs + link | 3 |
| GetClusterSummary C→A | 4–7 |
| Umbrella doc only | 8 |
| TF resource + kubeconfig DS | 10 |
| Compatibility matrix | 9 |

## Placeholder / consistency self-review

- Naming: `GetSummary` domain ↔ `GetClusterSummary` RPC ↔ `getVKSClusterSummary` UI — consistent.
- No TBD steps; open points from spec resolved as: name-prefix filter for workers; kubeconfig as **data source**; published versions constant synced to VERSIONS.md.
