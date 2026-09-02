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
  --version 0.6.0 \
  -n virtfoundry-system --create-namespace \
  --set secrets.rootPassword='change-me' \
  --set secrets.jwtSecret='change-me'
```

Login: `root` / your root password. For volume snapshots, set `platform.storage.snapshotClass` (e.g. `longhorn`) — see [Configuration — Snapshots](https://virtfoundry.github.io/helm-charts/docs/guide/configuration/#snapshots-vm-vs-volume).

## Repositories

| Repository | Purpose |
|------------|---------|
| [virtfoundry/core](https://github.com/virtfoundry/core) | REST API, UI (this repo) |
| [virtfoundry/operator](https://github.com/virtfoundry/operator) | `virtfoundry.io` CRDs and Kubernetes operator |
| [virtfoundry/helm-charts](https://github.com/virtfoundry/helm-charts) | Helm charts, docs, deploy scripts |
| [virtfoundry/terraform-provider-virtfoundry](https://github.com/virtfoundry/terraform-provider-virtfoundry) | Terraform provider |

## Product model

Open-source **core** (Apache 2.0) + optional **enterprise** components from Thurler IT. See [docs/PRODUCT.md](docs/PRODUCT.md).

## Enterprise support

Migration from CloudStack/VMware, SSO, billing, and SLA support: **Thurler IT Consultancy** — [thurlerit.com](https://thurlerit.com)

## Development

**MySQL (legacy):** default local config still uses `database.driver: mysql`.

**CRD store (recommended for operator work):** install CRDs from [virtfoundry/operator](https://github.com/virtfoundry/operator) or `helm-charts/charts/virtfoundry-operator`, then:

```bash
export VIRTFOUNDRY_STORE=kubernetes
ROOT_PASSWORD=virtfoundry go run ./cmd/server
cd ui && npm install && npm run dev
```

Requires in-cluster kubeconfig or `KUBECONFIG` pointing at a cluster with `virtfoundry.io` CRDs. The API reads/writes platform state as CRs; KubeVirt is still used for VM lifecycle and VNC until the operator owns all infra controllers.

See [docs/superpowers/specs/2026-09-01-crd-operator-design.md](docs/superpowers/specs/2026-09-01-crd-operator-design.md) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Documentation

| Doc | Description |
|-----|-------------|
| [docs/CI.md](docs/CI.md) | Required PR checks (Go, UI, Helm, Terraform) |
| [docs/WHY.md](docs/WHY.md) | Positioning vs Proxmox / KubeVirt / Harvester |
| [ROADMAP.md](ROADMAP.md) | Near-term product themes |
| [docs/CNCF-CHECKLIST.md](docs/CNCF-CHECKLIST.md) | Traction & CNCF Sandbox checklist |
| [GOVERNANCE.md](GOVERNANCE.md) | How decisions are made |
| [MAINTAINERS.md](MAINTAINERS.md) | Lead maintainer (Matheus) and maintainers (Rodrigo) |
| [ADOPTERS.md](ADOPTERS.md) | Who runs VirtFoundry |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design and API overview |
| [docs/VM-TEMPLATES.md](docs/VM-TEMPLATES.md) | VM template catalog, ISO import, container disks |
| [docs/PRODUCT.md](docs/PRODUCT.md) | Product model (core vs enterprise) |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security: [SECURITY.md](SECURITY.md). Conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). Maintainers: [MAINTAINERS.md](MAINTAINERS.md).

Questions and ideas: [GitHub Discussions](https://github.com/virtfoundry/core/discussions). Traction board: [VirtFoundry Traction](https://github.com/orgs/virtfoundry/projects/1).

## License

Apache License 2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).