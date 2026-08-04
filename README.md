# VirtFoundry

> Declarative cloud-native IaaS and private cloud on Kubernetes

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-v1.28%2B-326CE5?logo=kubernetes)](https://kubernetes.io)
[![KubeVirt](https://img.shields.io/badge/KubeVirt-Native-red)](https://kubevirt.io)

VirtFoundry turns Kubernetes clusters into a multi-tenant private cloud: VPCs, subnets, security groups, VMs, snapshots, IAM, and a web UI — built on KubeVirt and Multus.

## Key features

- **Multi-tenancy** — isolated namespaces per tenant
- **Networking** — VPCs, private subnets, optional public IP pool, security groups (NetworkPolicy)
- **Compute** — KubeVirt VMs, templates, offerings, snapshots, noVNC console
- **Storage** — PVC volumes and volume snapshots
- **IAM** — users, roles, API keys (`vfd_live_...`)
- **Packaging** — official Helm chart

## Quick start

**Prerequisites:** KubeVirt, Multus, and CDI on the cluster. See the [installation guide](https://virtfoundry.github.io/helm-charts/docs/guide/installation/).

```bash
helm repo add virtfoundry https://virtfoundry.github.io/helm-charts
helm repo update
helm install virtfoundry virtfoundry/virtfoundry \
  --version 1.1.0 \
  -n virtfoundry-system --create-namespace \
  --set secrets.rootPassword='change-me' \
  --set secrets.jwtSecret='change-me'
```

Login: `root` / your root password.

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

## License

Apache License 2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).