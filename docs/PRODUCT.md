# VirtFoundry — product model

VirtFoundry is an **open-core** private cloud platform. The core is Apache 2.0; commercial extensions are offered separately by [Thurler IT](https://thurlerit.com).

## Open source (VirtFoundry Core)

Free to use, modify, and self-host.

| Capability | Description |
|------------|-------------|
| Multi-tenancy | Tenant namespaces, root impersonation |
| IAM | Users, roles, API keys (`vfd_live_...`) |
| Compute | KubeVirt VMs, templates, offerings (shared + dedicated CPU), snapshots |
| Network | VPCs, private subnets, public network profile, security groups |
| Storage | Volumes, volume snapshots |
| Access | SSH keys, noVNC console, REST API |
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
- Managed Kubernetes (CKE) — backlog
- Object storage (S3-like) — backlog

## Versioning

- **1.0.0** — IAM; clean cluster install (`virtfoundry-system` namespace)
