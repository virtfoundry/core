# Roadmap

High-level product themes. Detailed CNCF/traction checklist: [docs/CNCF-CHECKLIST.md](docs/CNCF-CHECKLIST.md).

Status: **M** = must for traction · **S** = stretch

## Now (next 4–6 weeks)

| Theme | Items | Pri |
|-------|--------|-----|
| **Docs & narrative** | Why VirtFoundry (vs Proxmox), quickstart, topologies | M |
| **Install UX** | One clear happy path (Longhorn + Gateway); fail loudly without CSI snapshots | M |
| **Stability** | Volume snapshots + VM snapshots documented; E2E on homelab | M |
| **Networking** | AWS-style L4 LB (VIP + listener + target group); public + private targets when reachable — [plan](docs/LOAD-BALANCER-PLAN.md) · [#63](https://github.com/virtfoundry/core/issues/63) | M |
| **Community** | Discussions, good first issues, ADOPTERS template | M |
| **GitOps** | Document Argo/Helm install as recommended production path | S |

## Next (quarter)

| Theme | Items | Pri |
|-------|--------|-----|
| **Onboarding** | Kind/k3s demo script; sample tenant bootstrap | M |
| **Proxmox migrants** | Mental model guide + feature parity matrix (honest gaps) | M |
| **Provider** | Terraform provider examples aligned with UI flows | S |
| **Observability** | Metrics/dashboards for control plane | S |

## Later

| Theme | Items |
|-------|--------|
| SSO / OIDC | Enterprise-adjacent; keep core clean |
| Billing hooks | Optional / enterprise |
| CNCF Sandbox | After traction checklist Phase 1 exit |

## Non-goals (near term)

- Replacing every Proxmox VE feature (ZFS UX, PBS, etc.)
- Bundling KubeVirt/Multus/CDI as hard chart dependencies by default
- Claiming CNCF membership before Sandbox acceptance

## Feedback

Use [GitHub Issues](https://github.com/virtfoundry/core/issues) or Discussions. Roadmap changes are maintainer-driven with community input.
