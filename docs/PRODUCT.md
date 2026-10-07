# VirtFoundry — product model

VirtFoundry is an **open-core** private cloud platform. The core is Apache 2.0; commercial extensions are offered separately by [Thurler IT](https://thurlerit.com).

## Open source (VirtFoundry Core)

Free to use, modify, and self-host.

| Capability | Description |
|------------|-------------|
| Multi-tenancy | Tenant namespaces, root impersonation |
| IAM | Users, roles, API keys (`vfd_live_...`) |
| Compute | KubeVirt VMs, templates, offerings (shared + dedicated CPU), **VM** snapshots |
| Network | VPCs, private subnets, public network profile, security groups |
| Storage | Volumes; **volume** snapshots (requires CSI `VolumeSnapshot` + snapshot-capable StorageClass — not `local-path`) |
| Access | SSH keys (required for Linux create), optional one-time `cloud_init_password`, noVNC console, REST API |
| Packaging | Helm chart, optional sideload scripts |

## Enterprise (Thurler IT — separate license)

Paid components; not required to run the open-source core.

| Capability | Status |
|------------|--------|
| SSO (OIDC/SAML) | Planned |
| Billing / quotas / showback | Planned |
| Managed databases (RDS-like on VMs) | Planned |
| Audit log export & retention | Planned |
| Commercial support & SLA | Available on request |

Enterprise code lives outside the public `virtfoundry` GitHub organization.

## What we are not (yet)

- Serverless / Knative — separate future line, not core IaaS
- Managed Kubernetes (VKS) — shipped in 0.10.0 (see Versioning)
- Object storage (S3-like) — backlog

## Versioning

- **0.11.1** — fixes: deploy needs template_id, VM snapshot phase/vm_name, SSH fingerprint, wizard loading state, PT labels; dependency updates
- **0.11.0** — CRDs move to the `virtfoundry-crds` chart (upgraded by Helm, protected from uninstall); umbrella chart `virtfoundry-platform`; VKS chart published as OCI
- **0.10.0** — VKS (Kubernetes clusters): gRPC ClusterService, console create/detail, node image seed; tenant ownership labels; image signing
- **0.9.0** — Realtime/gRPC watch, Multus CRD-first networks, powerState Start/Stop, SSH PEM one-shot, security harden (CORS/SSH/Offerings)
- **0.8.0** — VM create Error / 0 vCPU after operator 0.7.2 (pod-network annotation + list enrichment)
- **0.7.2** — Security release (audit highs closed; non-root images; console tickets; ISO allowlist; CDI importer egress)
- **0.7.1** — VM create hides platform Windows ISO until the tenant uploads one
- **0.7.0** — MySQL/worker/migrate removed; CRD-only store; prerequisites docs
- **0.6.0** — CRD store default, operator chart, homelab cutover
- **0.5.0** — IAM, Helm install, public network, Kind lab
- **1.0.0** — not declared yet; freeze of public API and Helm chart contract
- Premature tags `v1.0.0`–`v1.5.0` were deleted (2026-09-27); they never meant SemVer 1.0
