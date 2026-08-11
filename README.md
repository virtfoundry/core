# VirtFoundry

> Declarative cloud-native IaaS and private cloud on Kubernetes

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-v1.28%2B-326CE5?logo=kubernetes)](https://kubernetes.io)
[![KubeVirt](https://img.shields.io/badge/KubeVirt-Native-red)](https://kubevirt.io)
[![Discussions](https://img.shields.io/badge/Discussions-join-brightgreen?logo=github)](https://github.com/virtfoundry/core/discussions)
[![Project](https://img.shields.io/badge/Project-Traction-blueviolet?logo=github)](https://github.com/orgs/virtfoundry/projects/1)

VirtFoundry turns Kubernetes clusters into a multi-tenant private cloud: VPCs, subnets, security groups, VMs, snapshots, IAM, and a web UI — built on KubeVirt and Multus.

Leaving Proxmox (or avoiding raw KubeVirt YAML)? Start with [Why VirtFoundry](https://virtfoundry.github.io/helm-charts/docs/guide/why/) ([source](docs/WHY.md)).

**Website / docs (canonical front door):** [virtfoundry.github.io/helm-charts/docs](https://virtfoundry.github.io/helm-charts/docs/) — Quickstart, topologies, and project docs. Custom domain (`virtfoundry.dev`) is optional later; until then this URL is the official entry point.

## Key features

- **Multi-tenancy** — isolated namespaces per tenant
- **Networking** — VPCs, private subnets, optional public IP pool, security groups (NetworkPolicy)
- **Compute** — KubeVirt VMs, templates, offerings, VM snapshots, noVNC console
- **Storage** — PVC volumes; volume snapshots (CSI `VolumeSnapshot` required — not `local-path`)
- **IAM** — users, roles, API keys (`vfd_live_...`)
- **Packaging** — official Helm chart

## Quick start

**Under 30 minutes** (cluster already has KubeVirt, Multus, CDI): follow the [Quickstart](https://virtfoundry.github.io/helm-charts/docs/guide/quickstart/).

Full prerequisites and platform setup: [installation guide](https://virtfoundry.github.io/helm-charts/docs/guide/installation/).

```bash
helm repo add virtfoundry https://virtfoundry.github.io/helm-charts
helm repo update
helm install virtfoundry virtfoundry/virtfoundry \
  --version 1.4.1 \
  -n virtfoundry-system --create-namespace \
  --set secrets.rootPassword='change-me' \
  --set secrets.jwtSecret='change-me'
```

Login: `root` / your root password. For volume snapshots, set `platform.storage.snapshotClass` (e.g. `longhorn`) — see [Configuration — Snapshots](https://virtfoundry.github.io/helm-charts/docs/guide/configuration/#snapshots-vm-vs-volume).

## Repositories

| Repository | Purpose |
|------------|---------|
| [virtfoundry/core](https://github.com/virtfoundry/core) | API, worker, UI (this repo) |
| [virtfoundry/helm-charts](https://github.com/virtfoundry/helm-charts) | Helm chart, docs, deploy scripts |

## Product model

Open-source **core** (Apache 2.0) + optional **enterprise** components from Thurler IT. See [docs/PRODUCT.md](docs/PRODUCT.md).

## Enterprise support

Migration from CloudStack/VMware, SSO, billing, and SLA support: **Thurler IT Consultancy** — [thurlerit.com](https://thurlerit.com)

## Development

```bash
ROOT_PASSWORD=virtfoundry go run ./cmd/server
cd ui && npm install && npm run dev
```

## Documentation

| Doc | Description |
|-----|-------------|
| [docs/CI.md](docs/CI.md) | Required PR checks (Go, UI, Helm, Terraform) |
| [docs/WHY.md](docs/WHY.md) | Positioning vs Proxmox / KubeVirt / Harvester |
| [ROADMAP.md](ROADMAP.md) | Near-term product themes |
| [docs/CNCF-CHECKLIST.md](docs/CNCF-CHECKLIST.md) | Traction & CNCF Sandbox checklist |
| [GOVERNANCE.md](GOVERNANCE.md) | How decisions are made |
| [ADOPTERS.md](ADOPTERS.md) | Who runs VirtFoundry |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design and API overview |
| [docs/VM-TEMPLATES.md](docs/VM-TEMPLATES.md) | VM template catalog, ISO import, container disks |
| [docs/PRODUCT.md](docs/PRODUCT.md) | Product model (core vs enterprise) |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security reports: [SECURITY.md](SECURITY.md). Conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

Questions and ideas: [GitHub Discussions](https://github.com/virtfoundry/core/discussions). Traction board: [VirtFoundry Traction](https://github.com/orgs/virtfoundry/projects/1).

## License

Apache License 2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).