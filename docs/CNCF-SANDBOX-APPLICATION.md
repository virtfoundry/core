# CNCF Sandbox application — draft answers

Living draft for the [cncf/sandbox](https://github.com/cncf/sandbox) issue form.  
**Do not submit** until the pre-submission checklist below is green (especially **repo age ≥6 months**).

Process: [Sandbox README](https://github.com/cncf/sandbox/blob/main/README.md) · TOC lifecycle: [cncf/toc process](https://github.com/cncf/toc/blob/main/process/README.md)  
Form: [New Sandbox Application](https://github.com/cncf/sandbox/issues/new?template=application.yml)

---

## Pre-submission checklist (2026)

| Gate | Status | Notes |
|------|--------|-------|
| Full Apache-2.0 `LICENSE` (not “convert later”) | ✅ | Official text in all official repos; copyright in `NOTICE` |
| `MAINTAINERS.md` Name / GitHub ID / Company + `/blob/` link | ✅ | https://github.com/virtfoundry/core/blob/main/MAINTAINERS.md |
| Repository **6+ months** old with active development | ⬜ | `core` created **2026-08-03** → earliest ~**2027-02-03** |
| Reusable project (not reference architecture) | ✅ | Control plane + operator + Helm + Terraform |
| Not a KubeVirt “operator-only” subproject candidate without narrative | 🟡 | See overlap / Why not KubeVirt subproject below |
| CoC + CONTRIBUTING + SECURITY | ✅ | Linked below |
| Contribution Agreement signatory identified | 🟡 | Fill before submit (legal entity TBD) |

---

## Form field draft

### Project summary

Kubernetes-native multi-tenant private cloud (IaaS control plane) on KubeVirt — tenants, VPCs, VMs, volumes, IAM, REST API, and UI.

### Project description

VirtFoundry turns an existing Kubernetes cluster into a multi-tenant private cloud: tenants, networking (VPC / security groups), VMs, volumes, snapshots, IAM, REST API, and a web UI. The hypervisor is [KubeVirt](https://kubevirt.io/); source of truth is `virtfoundry.io` CRDs managed by the VirtFoundry operator. Charts and docs ship from `virtfoundry/helm-charts`; automation via a first-party Terraform provider.

It targets teams leaving Proxmox (or avoiding raw KubeVirt YAML) who already run Kubernetes and want a CloudStack-like day-2 experience with GitOps (Helm / Argo CD).

**Not a reference architecture:** it is a reusable product (API, UI, operator, charts, provider), not a demo of wiring existing tools.

### Org repo URL

https://github.com/virtfoundry

### Project repo URL in scope

https://github.com/virtfoundry/core

### Additional repos in scope

- https://github.com/virtfoundry/operator  
- https://github.com/virtfoundry/helm-charts  
- https://github.com/virtfoundry/terraform-provider-virtfoundry  

### Website URL

https://virtfoundry.github.io/helm-charts/docs/

### Roadmap

https://github.com/virtfoundry/core/blob/main/ROADMAP.md

### Roadmap context

Near-term focus: install UX, snapshot/volume reliability, L4 load balancer UX, docs for Proxmox migrants, and community onboarding (Discussions, good first issues, adopters). CNCF Sandbox submission is planned only after traction checklist Phase 1 exit and the 6-month maturity gate.

### Contributing guide

https://github.com/virtfoundry/core/blob/main/CONTRIBUTING.md

### Code of Conduct

https://github.com/virtfoundry/core/blob/main/CODE_OF_CONDUCT.md

### Adopters

https://github.com/virtfoundry/core/blob/main/ADOPTERS.md

### Maintainers file

https://github.com/virtfoundry/core/blob/main/MAINTAINERS.md

### Security policy file

https://github.com/virtfoundry/core/blob/main/SECURITY.md

### Standard or specification?

N/A — VirtFoundry is an implementation (control plane + CRDs), not a standards body or specification project.

### Business product or service to project separation

This project is unrelated to any commercial product or service of the maintainers’ employers. Maintainers contribute as individuals. There is no proprietary “enterprise fork” controlling the roadmap; optional future enterprise-adjacent features (e.g. SSO) are called out as non-blocking and kept out of the open core unless contributed under Apache-2.0.

### Why CNCF?

CNCF is the natural home for a Kubernetes-native IaaS control plane that composes KubeVirt, Multus, CSI, and GitOps tooling. Membership would give vendor-neutral governance, trademark/IP clarity, discoverability in the landscape, and a clearer path for external contributors who are wary of company-owned “private cloud” projects. We chose CNCF over a language foundation because the primary integration surface is Kubernetes and CNCF projects.

### Benefit to the landscape

The landscape has strong container platforms and a mature VM runtime (KubeVirt), but a gap remains for **multi-tenant private-cloud UX** (tenant / VPC / catalog / IAM / UI) that is GitOps-native and assumes “bring your own cluster.” VirtFoundry fills that gap without reinventing the hypervisor or shipping another HCI appliance ISO.

### Cloud native 'fit'

VirtFoundry is cloud native by composition: declarative CRDs, operator pattern, Helm delivery, Kubernetes as the platform for networking/storage/scheduling, and API-first control plane suitable for Terraform and GitOps. Workloads are VMs scheduled by KubeVirt on the same cluster fabric as containers.

### Cloud native 'integration'

- **KubeVirt** — VM runtime  
- **Kubernetes** — API, RBAC, CSI, networking primitives  
- **Multus / CNI ecosystem** — tenant networking  
- **Helm / Argo CD** — install and GitOps  
- Optional: Longhorn or other CSI for disks/snapshots; Gateway API / MetalLB for ingress and VIP patterns  

### Cloud native overlap

- **KubeVirt** — VirtFoundry *depends on* KubeVirt; it does not replace it. Overlap is intentional layering (product/control plane vs hypervisor API).  
- **Cluster API** — different problem (cluster lifecycle vs tenant IaaS on an existing cluster).  
- Other CNCF projects may provide pieces (networking, storage, observability); VirtFoundry orchestrates tenant-facing IaaS resources on top.

### Similar projects

- **Harvester** — full HCI appliance; VirtFoundry assumes BYO Kubernetes.  
- **Proxmox VE** — hypervisor appliance (not CNCF); primary migration narrative.  
- **OpenStack** — full IaaS; VirtFoundry targets a thinner, K8s-native control plane.  
- **KubeVirt + custom operators** — teams can build this themselves; VirtFoundry packages the multi-tenant product layer.

### Landscape

Not listed yet on https://landscape.cncf.io/ (to be requested after Sandbox acceptance or when eligible).

### Insights

Not listed yet on https://insights.linuxfoundation.org/ (to be enabled before/at application).

### Will the project require a license exception?

N/A — Project uses Apache 2.0 license.

### Project "Domain Technical Review"

Not yet. Planned: Day 0 of the [General Technical Review](https://github.com/cncf/toc/blob/main/toc_subprojects/project-reviews-subproject/general-technical-questions.md) and an async intro to TAG Runtime / KubeVirt community before submission.

### Application contact email(s)

**TODO before submit:** primary maintainer email(s). Suggested placeholder until confirmed: Matheus Thurler (see GitHub profile / MAINTAINERS).

### Contributing or sponsoring entity signatory information

**TODO before submit:** legal entity (or individual) that will sign the [CNCF Contribution Agreement](https://github.com/cncf/foundation/blob/main/agreements/Sample%20Contribution%20Agreement%20(2025).pdf). Not yet finalized.

### Why not a KubeVirt subproject? (for TOC / comments)

VirtFoundry is a multi-repo product (API, UI, Helm docs, Terraform) with its own tenant model and governance. It *consumes* KubeVirt as the hypervisor API rather than extending a single KubeVirt controller. We will engage the KubeVirt community for alignment; if maintainers there prefer a tighter relationship later, we remain open — but Sandbox as an independent project matches the current scope and contributor surface.

---

## After TOC approval

1. Sign Contribution Agreement (do **not** claim “donated to CNCF” before this).  
2. Follow sandbox onboarding issues from CNCF staff.  
3. Donate trademarks/accounts per IP policy checkboxes on the form.
