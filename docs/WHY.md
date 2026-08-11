# Why VirtFoundry?

VirtFoundry turns an existing **Kubernetes** cluster into a **multi-tenant private cloud**: tenants, VPCs, security groups, VMs, volumes, snapshots, IAM, and a web UI — on top of [KubeVirt](https://kubevirt.io/), Multus, and your StorageClass (we recommend [Longhorn](https://longhorn.io/)).

## Who it is for

- Homelab / lab teams who **outgrew Proxmox** and already run (or want) Kubernetes  
- Platform engineers who want a **CloudStack-like API/UI** without operating OpenStack  
- Anyone who refuses to “kubectl apply VirtualMachine YAML” for every tenant day-2 task  

## Who it is *not* for (yet)

- Single-node “install ISO and forget” with zero Kubernetes interest → **Proxmox still wins**  
- Pure container PaaS (use your existing K8s platform)  
- Drop-in PBS / ZFS storage admin UX  

## VirtFoundry vs Proxmox

| | Proxmox VE | VirtFoundry |
|--|------------|-------------|
| Mental model | Hypervisor cluster + GUI | **Cloud** (tenant, VPC, offering, API key) |
| Under the hood | QEMU/KVM, LXC | **KubeVirt** on Kubernetes |
| Multi-tenancy | Datacenter/ACL oriented | First-class **tenants** + IAM |
| Automation | API / Terraform community | REST + UI + Terraform provider |
| Storage | ZFS, Ceph, local | Kubernetes **StorageClass** (Longhorn, …) |
| Day-2 cloud native | Add-ons | Native GitOps (Helm/Argo), Gateway API |
| Best at | Fast bare-metal virt appliance | Private cloud **control plane** on K8s |

**Migration pitch:** keep learning one platform (Kubernetes). Get VMs as cloud resources, not as another silo beside the cluster.

## VirtFoundry vs “just KubeVirt”

KubeVirt is an excellent **hypervisor API**. VirtFoundry is the **product layer**: multi-tenant isolation, networking model, catalog (templates/offerings), snapshots UX, IAM, and an opinionated Helm install.

If you only need a few VMs and are happy writing CRDs, you may not need VirtFoundry.

## VirtFoundry vs Harvester / similar

Harvester is a full HCI appliance experience. VirtFoundry assumes **you bring the cluster** (kubespray, kubeadm, managed K8s, …) and focuses on the **IaaS control plane** and tenant UX.

## Design principles

1. **Compose CNCF building blocks** — do not reinvent the hypervisor or CSI  
2. **API-first** — UI and Terraform are clients of the same control plane  
3. **Honest labs** — `local-path` is for demos; Longhorn (or equivalent) for real disks/snapshots  
4. **Open core** — Apache 2.0 core; optional enterprise services elsewhere  

## Try it

- [Installation guide](https://virtfoundry.github.io/helm-charts/docs/guide/installation/)  
- [Minimum vs production topologies](https://virtfoundry.github.io/helm-charts/docs/guide/topologies/)  
- [Traction / CNCF checklist](CNCF-CHECKLIST.md)  
