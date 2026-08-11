# Why VirtFoundry?

VirtFoundry turns an existing **Kubernetes** cluster into a **multi-tenant private cloud**: tenants, VPCs, security groups, VMs, volumes, snapshots, IAM, and a web UI — on top of [KubeVirt](https://kubevirt.io/), Multus, and your StorageClass (we recommend [Longhorn](https://longhorn.io/)).

## Who it is for

- Homelab / lab teams who **outgrew Proxmox** and already run (or want) Kubernetes  
- Platform engineers who want a **CloudStack-like API/UI** without operating OpenStack  
- Anyone who refuses to write raw KubeVirt YAML for every tenant day-2 task  

## Who it is *not* for (yet)

- Single-node “install ISO and forget” with zero Kubernetes interest → **stay on Proxmox**  
- Pure container PaaS (use your existing Kubernetes platform)  
- Drop-in Proxmox Backup Server / ZFS storage admin UX  

## VirtFoundry vs Proxmox

| Dimension | Proxmox VE | VirtFoundry |
|-----------|------------|-------------|
| Mental model | Hypervisor cluster + host GUI | **Cloud** resources: tenant, VPC, offering, API key |
| Runtime | QEMU/KVM, LXC on the host | **KubeVirt** VMs on Kubernetes |
| Multi-tenancy | Datacenter / ACL oriented | First-class **tenants** + IAM |
| Automation | API + community Terraform | First-party REST, UI, and [Terraform provider](https://registry.terraform.io/providers/virtfoundry/virtfoundry) |
| Storage UX | ZFS, Ceph, local disks in the product | Kubernetes **StorageClass** (Longhorn, …) |
| Day-2 ops | Proxmox-centric tooling | GitOps-native (Helm / Argo), Gateway API |
| Best fit | Fast bare-metal virt **appliance** | Private cloud **control plane** on an existing cluster |

**Migration pitch:** keep one platform (Kubernetes). Treat VMs as cloud resources, not a second silo beside the cluster.

### When Proxmox still wins

Be honest with yourself (and adopters):

- You want an **appliance ISO**, not a Kubernetes learning curve  
- You need **PBS**, ZFS datasets, or LXC as first-class product features  
- Your team will not operate KubeVirt / Multus / CSI  
- A handful of VMs with no multi-tenant “cloud” model is enough  

VirtFoundry is the better path when you **already want Kubernetes** and need tenant isolation, API/UI, and GitOps around VMs.

## VirtFoundry vs “just KubeVirt”

KubeVirt is an excellent **hypervisor API**. VirtFoundry is the **product layer**: multi-tenant isolation, networking model, catalog (templates / offerings), snapshots UX, IAM, and an opinionated Helm install.

If you only need a few VMs and are happy applying CRDs by hand, you may not need VirtFoundry.

## VirtFoundry vs Harvester / similar

Harvester is a full HCI appliance experience. VirtFoundry assumes **you bring the cluster** (kubespray, kubeadm, managed Kubernetes, …) and focuses on the **IaaS control plane** and tenant UX.

## Design principles

1. **Compose CNCF building blocks** — do not reinvent the hypervisor or CSI  
2. **API-first** — UI and Terraform are clients of the same control plane  
3. **Honest labs** — `local-path` is for demos; Longhorn (or equivalent) for real disks / snapshots  
4. **Open core** — Apache 2.0 core; optional enterprise services elsewhere  

## Try it

Published on the docs site: **[Why VirtFoundry](https://virtfoundry.github.io/helm-charts/docs/guide/why/)** (keep in sync with this file).

- [Quickstart (under 30 min)](https://virtfoundry.github.io/helm-charts/docs/guide/quickstart/)  
- [Installation guide](https://virtfoundry.github.io/helm-charts/docs/guide/installation/)  
- [Minimum vs production topologies](https://virtfoundry.github.io/helm-charts/docs/guide/topologies/)  
- [Traction / CNCF checklist](CNCF-CHECKLIST.md)  
