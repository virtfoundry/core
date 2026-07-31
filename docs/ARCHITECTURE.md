# VirtForge Cloud — Architecture

Multi-tenant IaaS platform native to Kubernetes. This document describes the current state of the project: what exists, how it fits together, and evolution recommendations.

---

## Overview

```
┌─────────────────────────────────────────────────────────────────┐
│  Browser (React SPA)                                            │
│  REST /api/v1  ·  WS /ws/events  ·  WS /ws/console (noVNC)     │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│  cmd/server          JWT middleware · handlers · WebSocket hub  │
│  cmd/worker          async_jobs · deploy_vm · ReconcileAll      │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│  internal/service/platform.go  (facade)                         │
│    tenant · identity · network · storage · compute · jobs       │
└─┬──────────────┬─────────────────────┬──────────────────────────┘
  │              │                     │
  ▼              ▼                     ▼
store.Repository  platform/k8s.Manager  hypervisor.KubeVirtDriver
(MySQL/Memory)    namespaces, NAD,      VirtualMachine, snapshots,
                  PVC, NetPol, VolSnap  VNC proxy
```

**Product:** VirtForge Cloud  
**Go module:** `github.com/virtforge-cloud/virtforge`  
**Deploy namespace:** `nimbus-system` (legacy prefix — rename planned)  
**Homelab UI:** `http://<node-ip>:30880` (login `root` / default dev password)

---

## Domínios funcionais

| Domínio | Entidades | Backend | UI | K8s / infra |
|---------|-----------|---------|-----|-------------|
| **Identidade** | users, JWT | `auth/login`, `auth/me` | Login | — |
| **Tenant** | tenants | `GET/POST /tenants` (root) | `/tenants` | Namespace `nimbus-tenant-{slug}` |
| **Rede** | vpcs, networks, security_groups | `GET/POST /vpcs`, `/networks`, `/security-groups` | `/vpcs`, `/networks`, `/security-groups` | VPC NS, Multus NAD, NetworkPolicy |
| **Compute** | vms, vm_nics, vm_snapshots, catálogo | `GET/POST /vms`, start/stop/delete, `/vm-snapshots` | `/vms`, `/vms/:name`, `/vm-snapshots`, `/console` | KubeVirt VM, VirtualMachineSnapshot |
| **Storage** | volumes, snapshots | `GET/POST /volumes`, `/snapshots` | `/volumes`, `/snapshots` | PVC, VolumeSnapshot |
| **Jobs** | async_jobs | interno (worker) | — | — |
| **Console** | — | `GET /ws/console` | `/console` (nova aba) | KubeVirt VNC subresource |

### Dependências entre domínios

```
Tenant ─────────────────────────────────────────────┐
  │                                                  │
  ├── VPC ── Network (NAD no NS da VPC)             │
  │         └── Security Group (NetPol no NS tenant) │
  ├── Volume ── Volume Snapshot                      │
  └── VM ── NICs (Multus) ── Network                 │
       ├── Service Offering (CPU/mem)                │
       ├── VM Template (imagem)                      │
       └── VM Snapshot                               │
```

Regra central: **tenant é a unidade de isolamento**. VMs e volumes vivem no namespace do tenant; VPCs têm namespace próprio; redes anexam VMs via Multus NAD.

---

## Camadas do backend

| Camada | Pacote | Responsabilidade |
|--------|--------|------------------|
| Transporte | `internal/api/handler` | HTTP decode, tenant resolution, JSON |
| Transporte | `internal/api/middleware` | JWT, CORS, logging |
| Transporte | `internal/api/ws` | Hub de eventos realtime |
| Transporte | `internal/api/handler/console_handler.go` | Proxy WebSocket VNC (KubeVirt) |
| Auth | `internal/auth` | JWT, bcrypt |
| Serviço | `internal/service/` | Facade `platform.go` + domínios (`tenant/`, `network/`, `compute/`, …) |
| Persistência | `internal/platform/store` | Interface `Repository` + MySQL/Memory |
| Modelos | `internal/platform/models.go` | Entidades compartilhadas |
| Infra K8s | `internal/platform/k8s` | tenant, vpc, securitygroup, volume, snapshot |
| Hypervisor | `internal/infra/hypervisor` | Interface `Driver` + KubeVirt |
| **Migration** | `internal/migrate`, `cmd/migrate` | Import CloudStack → VirtForge |

### Fluxo típico: deploy de VM

1. Handler valida JWT e resolve `tenant_id`
2. `PlatformService.DeployVM` lê offering/template do catálogo (opcional)
3. Garante namespace do tenant via `k8s.Manager`
4. Monta NICs a partir de `network_ids` → refs Multus NAD
5. Cria `VirtualMachine` via `KubeVirtDriver` (video virtio exceto Cirros)
6. Persiste `vms` + `vm_nics` no store
7. Broadcast `vm.created` no WebSocket hub
8. Modo async: enfileira job `deploy_vm` no worker

### Worker (`cmd/worker`)

- Poll `async_jobs` a cada 3s
- Executa `deploy_vm`, `reconcile`
- `ReconcileAll` a cada 15s: adota VMs órfãs do KubeVirt, marca `Destroyed` se sumiram
- `SyncAllVMStates`: sincroniza fase KubeVirt → DB + eventos WS

---

## API REST (`/api/v1`)

| Recurso | Métodos | Auth |
|---------|---------|------|
| `/auth/login` | POST | público |
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
| `/vm-templates` | GET | JWT |
| `/vms` | GET, POST | JWT |
| `/vms/{name}` | GET, PATCH | JWT |
| `/vms/start`, `/stop`, `/delete` | POST | JWT |

**WebSockets**

| Path | Uso |
|------|-----|
| `/ws/events` | Eventos realtime (`vm.created`, `vm.updated`, …) |
| `/ws/console?name=&namespace=` | Proxy VNC noVNC |

**Multi-tenancy:** usuário root envia header `X-Tenant-ID` para operar em um tenant.

### Lacunas conhecidas da API

- CRUD de usuários (só bootstrap root + admin na criação de tenant)
- Delete/update de VPC, network, volume, snapshot, security group
- Endpoint de status de jobs async
- Auth no WebSocket do console (hoje só `name` + `namespace`)

---

## Frontend (`ui/`)

| Rota | Página | Funcionalidades |
|------|--------|-----------------|
| `/login` | Login | JWT |
| `/dashboard` | Dashboard | Contagens, atalhos |
| `/tenants` | Tenants | Root: criar/listar |
| `/vms` | VMs | Listar, deploy, start/stop/delete, console |
| `/vms/:name` | VMDetail | Overview, NICs, snapshots, edit cpu/mem |
| `/volumes` | Volumes | Listar/criar |
| `/vpcs` | VPCs | Listar/criar |
| `/networks` | Networks | Listar/criar |
| `/security-groups` | SecurityGroups | Listar/criar + regras |
| `/snapshots` | Snapshots | Volume snapshots |
| `/vm-snapshots` | VMSnapshots | VM snapshots CRUD + restore |
| `/console` | VMConsole | noVNC full-screen, comandos de teclado |

**Stack UI:** React 18, Vite, TypeScript, Tailwind, TanStack Query, React Router, noVNC.

**Convenções:** query keys em `lib/query-keys.ts`; API client em `lib/platform-api.ts`.

### Lacunas conhecidas da UI

- Catálogo hardcoded (`VM_IMAGES`, `VM_SIZES`) — API `/service-offerings` e `/vm-templates` existem mas não são usadas
- Sem seletor de rede no deploy de VM (backend já suporta `network_ids`)
- Sem toggle de deploy async
- Sem página de usuários

---

## Console VNC

- Abre em **nova aba** (`/console?name=&namespace=`), estilo AWS
- Proxy WebSocket em `console_handler.go` (padrão KubeVirt `CopyFrom`/`CopyTo`)
- Comandos: Ctrl+Alt+Del, Esc, Tab, Enter, F1–F12, colar texto
- **Cirros:** VGA fixo ~720×400 — funciona com `scaleViewport`; aviso amarelo é esperado
- **Ubuntu/Fedora:** `video: virtio` + resize remoto quando KubeVirt anuncia suporte
- Feature gate `VideoConfig` habilitado no homelab via script de deploy

---

## Banco de dados

Schema: `internal/platform/store/migrations/schema.sql`

| Tabela | Descrição |
|--------|-----------|
| `users` | root / tenant_admin |
| `tenants` | slug, namespace, campos de import |
| `vpcs` | VPC por tenant |
| `networks` | NAD Multus (namespace + nome) |
| `security_groups` | Regras em JSON |
| `service_offerings` | Catálogo CPU/mem |
| `vm_templates` | Imagens container disk |
| `vms` | Metadados KubeVirt |
| `vm_nics` | NICs por VM |
| `volumes` | PVCs |
| `snapshots` | VolumeSnapshots |
| `vm_snapshots` | VirtualMachineSnapshots |
| `async_jobs` | Fila do worker |

Store: MySQL se `database.dsn` configurado; senão Memory com seed de catálogo.

---

## Kubernetes deploy

Manifests live in [virtforge-chart](https://github.com/virtforge-cloud/virtforge-chart):

```
virtforge-chart/kustomize/base/              # Kustomize base
virtforge-chart/kustomize/overlays/homelab/  # NodePort 30880, no Ingress
virtforge-chart/charts/virtforge/            # Helm chart (production)
```

| Workload | Image | Role |
|----------|-------|------|
| `nimbus-api` | `nimbus/iaas-api` | `./server` |
| `nimbus-worker` | `nimbus/iaas-api` | `./worker` |
| `nimbus-ui` | `nimbus/iaas-ui` | nginx + SPA |
| `nimbus-mysql` | mysql:8 | StatefulSet |

Homelab deploy: `virtforge-chart/deploy-homelab.sh` (build images from this repo, apply kustomize, import to containerd, restart).

---

## Estrutura de diretórios

```
virtforge/
├── cmd/
│   ├── server/           # REST API + WebSockets
│   ├── worker/           # Async jobs + reconciliation
│   └── migrate/          # CloudStack import CLI
├── internal/
│   ├── api/              # Handlers, middleware, WS
│   ├── auth/             # JWT
│   ├── config/           # Viper
│   ├── service/          # Domain services (tenant, network, compute, …)
│   ├── platform/
│   │   ├── store/        # Repository + MySQL/Memory
│   │   └── k8s/          # K8s provisioning per resource
│   ├── infra/hypervisor/ # KubeVirt driver
│   └── migrate/          # Import logic
├── ui/src/               # React SPA
├── docker/               # Dockerfiles + nginx config (images only)
├── docs/
├── TODO.md
└── AGENTS.md
```

K8s manifests and homelab scripts: [virtforge-chart](https://github.com/virtforge-cloud/virtforge-chart).

**Monorepo decision:** domain packages in `internal/service/`, single deploy unit. No separate microservice repos until API boundaries stabilize.

---

## Roadmap

See [TODO.md](../TODO.md) for the current backlog.
