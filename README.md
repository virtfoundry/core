# VirtForge Cloud

Kubernetes-native IaaS control plane — CloudStack-style operations on KubeVirt, Multus, and NetworkPolicy.

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev/)

**Organization:** [github.com/virtforge-cloud](https://github.com/virtforge-cloud)

| Repository | Purpose |
|------------|---------|
| [virtforge](https://github.com/virtforge-cloud/virtforge) | Monorepo — API, worker, UI, Kustomize |
| [charts](https://github.com/virtforge-cloud/charts) | Helm charts for cluster install |
| [website](https://github.com/virtforge-cloud/website) | Landing page and docs site |

Extended documentation: **[Wiki](https://github.com/virtforge-cloud/virtforge/wiki)**

## Features

- Multi-tenant isolation (namespaces per tenant / VPC)
- VMs via KubeVirt (start/stop, console, snapshots)
- Networks via Multus NADs; security groups via NetworkPolicy
- Block storage (PVC) and volume snapshots
- CloudStack migration tool (`cmd/migrate`)
- React UI with realtime events and noVNC console

## Stack

| Component | Technology |
|-----------|------------|
| API | Go + Gorilla Mux |
| Worker | Go (async jobs) |
| Auth | JWT (root + tenant admins) |
| Hypervisor | KubeVirt |
| UI | React + Vite + Tailwind |
| Deploy | Kustomize (`deployments/k8s/`) or [Helm](https://github.com/virtforge-cloud/charts) |

## Quick start (local dev)

```bash
# API
ROOT_PASSWORD=nimbus go run ./cmd/server

# UI
cd ui && npm install && npm run dev

# Login: root / nimbus (change in production)
```

## API (`/api/v1`)

| Resource | Endpoints |
|----------|-----------|
| Auth | `POST /auth/login`, `GET /auth/me` |
| Tenants (root) | `GET/POST /tenants` |
| VPCs / Networks / Security Groups | CRUD under `/vpcs`, `/networks`, `/security-groups` |
| Volumes / Snapshots | `/volumes`, `/snapshots` |
| VMs | `/vms`, start/stop/delete, `/vm-snapshots` |
| Console | WebSocket `/ws/console?name=&namespace=` |
| Realtime | WebSocket `/ws/events` |

Root users must send `X-Tenant-ID` when operating inside a tenant.

## Homelab deploy

```bash
./scripts/setup-homelab-kubevirt.sh
./scripts/deploy-nimbus-homelab.sh
```

## CloudStack migration

```bash
go run ./cmd/migrate cloudstack \
  --dsn "cloud:password@tcp(cloudstack-db:3306)/cloud" \
  --dry-run
```

See the [Wiki — CloudStack migration](https://github.com/virtforge-cloud/virtforge/wiki/CloudStack-Migration) for full options.

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md). Commits must be in **English** and follow **Conventional Commits**.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
