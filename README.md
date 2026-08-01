# VirtForge Cloud

Kubernetes-native IaaS control plane — CloudStack-style operations on KubeVirt, Multus, and NetworkPolicy.

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev/)

**Organization:** [github.com/virtforge-cloud](https://github.com/virtforge-cloud)

| Repository | Purpose |
|------------|---------|
| [virtforge](https://github.com/virtforge-cloud/virtforge) | Application — API, worker, UI, migration CLI |
| [virtforge-chart](https://github.com/virtforge-cloud/virtforge-chart) | Helm chart, cluster config, homelab deploy |
| [virtforge-website](https://github.com/virtforge-cloud/virtforge-website) | Landing page and docs site |

Extended documentation: **[Wiki](https://github.com/virtforge-cloud/virtforge/wiki)**

## Repository layout

```
cmd/                 # server, worker, migrate CLIs
internal/            # API, services, store, KubeVirt/K8s integration
ui/                  # React SPA (Vite + Tailwind)
docker/              # Dockerfiles for API/worker and UI images
config/              # Local dev config only (see config/README.md)
docs/                # Architecture reference
```

Kubernetes manifests, Helm values, and deploy scripts live in **[virtforge-chart](https://github.com/virtforge-cloud/virtforge-chart)**.

## Features

- Multi-tenant isolation (namespaces per tenant / VPC)
- VMs via KubeVirt (start/stop, console, snapshots)
- Networks via Multus NADs; security groups via NetworkPolicy
- Block storage (PVC) and volume snapshots
- CloudStack migration tool (`cmd/migrate`)
- React UI with realtime events and noVNC console

## Quick start (local dev)

```bash
cp config/config.yaml.example config/config.yaml   # optional — defaults work for memory store

# API
ROOT_PASSWORD=virtforge go run ./cmd/server

# UI (separate terminal)
cd ui && npm install && npm run dev

# Login: root / virtforge (change in production)
```

## Deploy to Kubernetes

Use [virtforge-chart](https://github.com/virtforge-cloud/virtforge-chart):

```bash
helm upgrade --install virtforge ./charts/virtforge \
  -n virtforge-system --create-namespace \
  --set secrets.rootPassword='your-root-password' \
  --set secrets.jwtSecret='your-jwt-secret'
```

Homelab profile and optional image-build workflow: see the chart repo README.

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
