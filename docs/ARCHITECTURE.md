# VirtFoundry — Architecture

Multi-tenant IaaS platform native to Kubernetes. This document describes the current state of the project: what exists, how it fits together, and evolution recommendations.

---

## Overview

```
┌─────────────────────────────────────────────────────────────────┐
│  Browser (React SPA) · Terraform provider                       │
│  REST /api/v1  ·  WS /ws/events  ·  WS /ws/console (noVNC)     │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│  cmd/server          JWT · handlers · WebSocket hub             │
│  store.Repository    memory (dev) | kubernetes (CRDs)            │
└────────────────────────────┬────────────────────────────────────┘
                             │  dynamic client (kubernetes store)
┌────────────────────────────▼────────────────────────────────────┐
│  virtfoundry.io CRs in etcd (Tenant, Instance, VPC, …)         │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│  virtfoundry-operator    Tenant → Namespace                     │
│                          Instance → status from KubeVirt (WIP) │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│  KubeVirt · Multus · CDI · PVC · NetworkPolicy (infra)          │
└─────────────────────────────────────────────────────────────────┘
```

**Target (spec):** operator owns all infra reconciliation; API is a REST facade over CRs only. **Today:** API still uses `hypervisor.KubeVirtDriver` and `platform/k8s.Manager` for VM lifecycle, NAD, and console while controllers are ported.

**Prerequisites:** [KubeVirt](https://kubevirt.io/), [Multus](https://github.com/k8snetworkplumbingwg/multus-cni), [CDI](https://github.com/kubevirt/containerized-data-importer) — see [Platform prerequisites](https://virtfoundry.github.io/helm-charts/docs/guide/prerequisites/).

**Product:** VirtFoundry  
**Go module:** `github.com/virtfoundry/core`  
**Deploy:** [helm-charts](https://github.com/virtfoundry/helm-charts) — install **virtfoundry-operator** (CRDs) then **virtfoundry** (API+UI)

---

## Functional domains

| Domain | Entities | Backend | UI | K8s / infra |
|--------|----------|---------|-----|-------------|
| **Identity** | users, JWT | `auth/login`, `auth/me` | Login | — |
| **Tenant** | tenants | `GET/POST /tenants` (root) | `/tenants` | Namespace `virtfoundry-tenant-{slug}` |
| **Network** | vpcs, networks, security_groups | `GET/POST /vpcs`, `/networks`, `/security-groups` | `/vpcs`, `/networks`, `/security-groups` | VPC NS, Multus NAD, NetworkPolicy; **default VPC** (`10.0.0.0/16`) per tenant |
| **Compute** | vms, vm_nics, vm_snapshots, catalog | `GET/POST /vms`, start/stop/delete, `/vm-snapshots` | `/vms`, `/vms/:name`, `/vm-snapshots`, `/console` | KubeVirt VM, VirtualMachineSnapshot |
| **Storage** | volumes, snapshots | `GET/POST /volumes`, `/snapshots` | `/volumes`, `/snapshots` | PVC; VolumeSnapshot (needs CSI snapshotter — not `local-path`) |
| **Console** | — | `GET /ws/console` | `/console` (new tab) | KubeVirt VNC subresource |

### Domain dependencies

```
Tenant ─────────────────────────────────────────────┐
  │                                                  │
  ├── VPC ── Network (NAD in VPC namespace)         │
  │         └── Security Group (NetPol in tenant NS) │
  ├── Volume ── Volume Snapshot                      │
  └── VM ── NICs (Multus) ── Network                 │
       ├── Service Offering (CPU/mem)                │
       ├── VM Template (image)                       │
       └── VM Snapshot                               │
```

**Core rule:** tenant is the isolation unit. VMs and volumes live in the tenant namespace; VPCs have their own namespace; networks attach to VMs via Multus NAD.

---

## Backend layers

| Layer | Package | Responsibility |
|-------|---------|----------------|
| Transport | `internal/api/handler` | HTTP decode, tenant resolution, JSON |
| Transport | `internal/api/middleware` | JWT, CORS, logging |
| Transport | `internal/api/ws` | Realtime event hub |
| Transport | `internal/api/handler/console_handler.go` | KubeVirt VNC WebSocket proxy |
| Auth | `internal/auth` | JWT, bcrypt |
| Service | `internal/service/` | Facade `platform.go` + domains (`tenant/`, `network/`, `compute/`, …) |
| Persistence | `internal/platform/store` | `Repository` — Memory (dev) or Kubernetes CRDs |
| Models | `internal/platform/models.go` | Shared entities |
| K8s infra | `internal/platform/k8s` | tenant, vpc, securitygroup, volume, snapshot, load balancer |
| Hypervisor | `internal/infra/hypervisor` | `Driver` interface + KubeVirt |

### Typical flow: VM deploy

1. Handler validates JWT and resolves `tenant_id`
2. `PlatformService.DeployVM` reads offering/template from catalog (optional)
3. Ensures tenant namespace via `k8s.Manager`
4. Builds NICs from `network_ids` → Multus NAD refs
5. Creates `VirtualMachine` via `KubeVirtDriver` with guest `domain.cpu.cores` from the offering; shared VMs omit CPU request so KubeVirt `cpuAllocationRatio` can overcommit; `dedicated_cpu` (flag or dedicated offering) sets Guaranteed QoS (`request=limit`)
6. Cloud-init / Multus static IP when public network is attached
7. Persists `vms` + `vm_nics` in store
8. Broadcasts `vm.created` on WebSocket hub

---

## REST API (`/api/v1`)

| Resource | Methods | Auth |
|----------|---------|------|
| `/auth/login` | POST | public |
| `/auth/me` | GET | JWT |
| `/tenants` | GET, POST | JWT + root |
| `/vpcs` | GET, POST | JWT |
| `/networks` | GET, POST | JWT |
| `/security-groups` | GET, POST | JWT |
| `/volumes` | GET, POST | JWT |
| `/snapshots` | GET, POST | JWT |
| `/vm-snapshots` | GET, POST | JWT |
| `/vm-snapshots/delete` | POST | JWT |
| `/vm-snapshots/restore` | POST | JWT |
| `/service-offerings` | GET | JWT |
| `/vm-templates` | GET, POST | JWT |
| `/vm-templates/{id}` | PATCH, DELETE | JWT |
| `/vms` | GET, POST | JWT |
| `/vms/{name}` | GET, PATCH | JWT |
| `/vms/start`, `/stop`, `/delete` | POST | JWT |

**WebSockets**

| Path | Purpose |
|------|---------|
| `/ws/events` | Realtime events (`vm.created`, `vm.updated`, …) |
| `/ws/console?name=&namespace=` | noVNC proxy |

**Multi-tenancy:** root users send header `X-Tenant-ID` to operate inside a tenant.

### Known API gaps

- User CRUD (only root bootstrap + admin on tenant create)
- Delete/update for VPC, network, volume, snapshot, security group
- Async job status endpoint
- Console WebSocket auth (today only `name` + `namespace`)

---

## Frontend (`ui/`)

| Route | Page | Features |
|-------|------|----------|
| `/login` | Login | JWT |
| `/dashboard` | Dashboard | Counts, shortcuts |
| `/tenants` | Tenants | Root: create/list |
| `/vms` | VMs | List, deploy, start/stop/delete, console |
| `/vms/:name` | VMDetail | Overview, NICs, snapshots, edit cpu/mem |
| `/volumes` | Volumes | List/create |
| `/vpcs` | VPCs | List/create |
| `/networks` | Networks | List/create |
| `/security-groups` | SecurityGroups | List/create + rules |
| `/snapshots` | Snapshots | Volume snapshots |
| `/vm-snapshots` | VMSnapshots | VM snapshots CRUD + restore |
| `/templates` | Templates | List/create/edit tenant templates; platform catalog read-only |
| `/console` | VMConsole | noVNC full-screen, keyboard commands |

**UI stack:** React 18, Vite, TypeScript, Tailwind, Redux Toolkit, TanStack Query, React Router, noVNC.

**Conventions:** query keys in `lib/query-keys.ts`; API client in `lib/platform-api.ts`.

### Known UI gaps

- VM deploy uses default VPC subnet automatically; optional extra subnets only (no full network picker)
- No async deploy toggle

---

## VNC console

- Opens in a **new tab** (`/console?name=&namespace=`), AWS-style
- WebSocket proxy in `console_handler.go` (KubeVirt `CopyFrom`/`CopyTo` pattern)
- Commands: Ctrl+Alt+Del, Esc, Tab, Enter, F1–F12, paste text
- **Cirros:** fixed VGA ~720×400 — works with `scaleViewport`; yellow warning is expected
- **Ubuntu/Fedora:** `video: virtio` + remote resize when KubeVirt advertises support
- `VideoConfig` feature gate can be enabled via helm-charts `platform.kubevirt.featureGates`

---

## Persistence (CRD store)

Production and homelab use `store.driver=kubernetes`: platform entities are `virtfoundry.io` CRs in etcd (see [operator CRD design](superpowers/specs/2026-09-01-crd-operator-design.md)).

Local `go run` without a cluster uses the in-memory store with seeded catalog.

**Not used:** MySQL, Vitess, or `cmd/worker`.

---

## Kubernetes deploy

Manifests and Helm values live in [helm-charts](https://github.com/virtfoundry/helm-charts):

```
helm-charts/charts/virtfoundry/                    # Helm chart + values profiles
helm-charts/scripts/sideload/import-pod.yaml     # Image sideload (no registry)
```

| Workload | Image | Role |
|----------|-------|------|
| `virtfoundry-api` | `ghcr.io/virtfoundry/core` | `./server` |
| `virtfoundry-ui` | `ghcr.io/virtfoundry/ui` | nginx + SPA |
| `virtfoundry-operator` | `ghcr.io/virtfoundry/operator` | CRD controllers |

Install order: platform prerequisites → **virtfoundry-operator** → **virtfoundry**. See [Installation](https://virtfoundry.github.io/helm-charts/docs/guide/installation/).

---

## Repository layout

```
virtfoundry/
├── cmd/server/           # REST API + WebSockets
├── internal/
│   ├── api/              # Handlers, middleware, WS
│   ├── auth/             # JWT
│   ├── config/           # Viper
│   ├── service/          # Domain services (tenant, network, compute, …)
│   ├── platform/
│   │   ├── store/        # Repository — Memory / Kubernetes
│   │   └── k8s/          # K8s provisioning per resource
│   └── infra/hypervisor/ # KubeVirt driver
├── ui/src/               # React SPA
├── config/               # Local dev config (cluster config → helm-charts)
├── docker/               # Dockerfiles + nginx config (images only)
├── docs/
└── ROADMAP.md
```

**Monorepo decision:** domain packages in `internal/service/`, single deploy unit. No separate microservice repos until API boundaries stabilize.

---

## Roadmap

See [ROADMAP.md](../ROADMAP.md) and [CNCF-CHECKLIST.md](CNCF-CHECKLIST.md).
