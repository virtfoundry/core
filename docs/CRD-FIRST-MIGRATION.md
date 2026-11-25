# CRD-first migration plan

> **Status: proposal, awaiting maintainer approval. No code is changed by this document.**
> Companion: [MULTI-REPLICA.md](MULTI-REPLICA.md) (items 5 and 6 share the load balancer step).

## Goal

The API only reads and writes `virtfoundry.io` CRs. Every Kubernetes object that backs them
(namespaces, NetworkPolicies, PVCs, Services, DataVolumes, NADs, snapshots) is created, updated and
deleted by a controller in `operator`. End state: the API ClusterRole in
`helm-charts/charts/virtfoundry/templates/rbac.yaml` shrinks to CRs plus a few read-only grants.

Non-goals: changing REST routes or payload shapes, changing existing CRD schemas (additive fields
only, flagged per step), moving off KubeVirt/Multus/CDI.

## Where things are today

Controllers in `operator/internal/controller`: **Tenant**, **Instance**, **Network** only.
CRDs that exist but have no controller: `VPC`, `SecurityGroup`, `Disk`, `DiskSnapshot`,
`InstanceSnapshot`, `IPAddress` (types in `operator/api/v1alpha1`). `LoadBalancer` and `TargetGroup`
have **no CRD**: the API keeps them in memory (`store/kubernetes_ephemeral.go`).

The API still writes cluster objects directly through `internal/platform/k8s`:

| Core call (`internal/platform/k8s`) | Called from | Object | Owner after migration |
|--------------------------------------|-------------|--------|------------------------|
| `EnsureTenantNamespace` / `DeleteTenantNamespace` (+ quota, CDI importer egress NetworkPolicy) | `service/tenant` | Namespace, ResourceQuota, NetworkPolicy | Tenant controller (already creates the Namespace) |
| `CreateNetworkAttachment`, `CreateSharedNetworkAttachment`, `DeleteNetworkAttachment` | `service/network` | NAD | Network controller (already creates the NAD for isolated Networks) |
| `EnsureVPCNamespace` | `service/network` | Namespace | new VPC controller |
| `ApplySecurityGroup` / `DeleteSecurityGroup` | `service/network` | NetworkPolicy | new SecurityGroup controller |
| `CreatePVC` / `DeletePVC` | `service/storage` | PVC | new Disk controller |
| `CreateVolumeSnapshot` | `service/storage` | VolumeSnapshot | new DiskSnapshot controller |
| `CreateBlankDataVolume`, `CreateHTTPImportDataVolume`, `WaitDataVolumeReady` | `compute/iso.go` | DataVolume | Template/Instance controller |
| `EnsureVMSSHService` | `compute` | Service (NodePort) | Instance controller (`spec.ssh`) |
| `EnsureLBService`, `DeleteLBService`, `SyncLBEndpoints` | `service/loadbalancer` | Service, Endpoints | new LoadBalancer controller |
| VM snapshot calls (`VirtualMachineSnapshot`) | `compute` | VirtualMachineSnapshot | new InstanceSnapshot controller |

**Dual writers today** (the API and a controller both create the same child): Tenant namespace and
Network NAD. These are the first things to retire, because the controller side already exists and
the E2E in `operator/hack/e2e-kubevirt.sh` already exercises both. They are also the risk: two
writers can fight over labels and config (see the 0.13 per-tenant bridge fix, operator#79).

## Controller contract (every step follows it)

- **Finalizer** `virtfoundry.io/<kind>`: remove children, then the finalizer. Deleting a CR with
  stuck children must not leak cluster objects.
- **ownerReferences** (`controller: true`, `blockOwnerDeletion: true`) from every child to its CR,
  except cross-namespace children (VPC/Tenant namespaces), which use the label
  `virtfoundry.io/<kind>-uid` and are cleaned by the finalizer.
- **Status**: `status.phase` stays (`Pending|Ready|Failed|Terminating`, UI and REST depend on it)
  plus `status.conditions` (`Ready`, `Progressing`, `Degraded`) and `status.observedGeneration`.
  Conditions carry reason/message so the UI can show why something is stuck.
- **Idempotent, level-based** reconcile (`CreateOrUpdate`), requeue with backoff on transient errors,
  no sleeping loops. Admission guards from `config/crd` stay authoritative.
- **Tests**: envtest per controller (create, update, delete with finalizer, adopt existing object)
  plus a step in the E2E that creates the resource through the API and asserts the child object.
- **Chart order**: RBAC rule in `helm-charts/charts/virtfoundry-operator` first, then the kubebuilder
  marker in `operator`. CRD changes: `make sync-crds-chart` in helm-charts in the same release.
- **Docs**: each PR updates `docs/ARCHITECTURE.md` (the status table below) and the CHANGELOGs.

## Order (one controller per PR, smallest risk first)

| Step | Scope | Operator PR | Core PR | Why this position |
|------|-------|-------------|---------|-------------------|
| 0 | **Parity audit** for Tenant and Network: list what the API creates that the controller does not (quota, CDI egress NetworkPolicy, shared NAD, labels) | add missing bits to controllers | none | Needed before removing a writer |
| 1 | **Tenant**: API stops calling `EnsureTenantNamespace`/`DeleteTenantNamespace`; waits for `status.phase=Ready` | quota + egress policy in controller | `service/tenant` writes the Tenant CR and polls status | Controller exists; removes a dual writer |
| 2 | **Network**: API stops creating NADs | shared (non-isolated) NAD in controller | `service/network` | Same reasoning |
| 3 | **VPC** | new `VPC` controller (namespace, labels) | `EnsureVPCNamespace` removed | Small, depends on step 2 |
| 4 | **SecurityGroup** | new controller: CR → NetworkPolicy (port `ApplySecurityGroup`) | `ApplySecurityGroup` removed | Pure translation logic, easy to unit test |
| 5 | **Disk + DiskSnapshot** | PVC and VolumeSnapshot controllers | `service/storage` | Needs StorageClass/snapshot class handling from chart values |
| 6 | **Instance extras**: SSH NodePort service, ISO import DataVolume, InstanceSnapshot | extend Instance controller; new InstanceSnapshot controller | `compute` | Touches the hottest controller, so after the small ones |
| 7 | **LoadBalancer + TargetGroup** | **new CRDs** + controller (Service + Endpoints) | `service/loadbalancer`; store stops using in-memory maps | Largest change; also unblocks `api.replicas > 1` |
| 8 | **Shrink the API ClusterRole** | none | none; chart only | Only after 1-7 land |

Final ClusterRole (step 8). Keep: `virtfoundry.io/*` (+`/status`), `nodes` get/list, `metrics.k8s.io/nodes`,
`customresourcedefinitions` get/list (addon strip), `events` get/list/watch, `pods` get/list and `pods/log`
get (logs/console), `subresources.kubevirt.io` vnc/console, API-key `secrets` (or move to an `APIKey`
controller later). Remove: `namespaces`, `services`, `endpoints`, `persistentvolumeclaims`,
`resourcequotas`, `networkpolicies`, `kubevirt.io` VMs/VMIs write verbs (read stays if the
informer-less list path still needs it), `snapshot.kubevirt.io`, `snapshot.storage.k8s.io`,
`cdi.kubevirt.io`, `k8s.cni.cncf.io`. `make verify-api-rbac-contract` already guards drift.

## Behavior that changes (needs approval)

The REST API currently returns the finished object synchronously (for example `POST /vpcs` returns the
VPC and its default network). With a controller doing the work the CR is created first and becomes
`Ready` shortly after. To keep payload shapes, handlers will return **201 with `state: "pending"`**
where the object is not ready yet, and the UI/Terraform must tolerate it. Terraform already
waits on state for VMs (`StateMatches`); volumes, VPCs and networks need the same. This is the one
user-visible change per step and is called out in each PR.

## What to test on the homelab (per step)

- Create the resource through the UI/REST, `kubectl get` the CR and the child; delete the CR and see the child gone.
- Kill the operator pod mid-create; it converges after restart.
- Delete a CR whose child was edited by hand; the controller restores it.
- Upgrade path: existing objects created by the old API path are **adopted** (labels/ownerRefs added), not recreated.

## Questions for the maintainer

1. **Sync vs async REST:** accept `201 pending` for VPC, Network, Volume, Snapshot, SG and LB, or must the API block until `Ready` (timeouts, simpler clients)?
2. **Order:** OK with steps 1-2 first (retire the two dual writers), or do you want LoadBalancer earlier because of multi-replica?
3. **LoadBalancer CRDs:** one `LoadBalancer` CR with inline listeners and targets, or separate `TargetGroup`/`Listener`/`Target` CRs? Proposal: `LoadBalancer` and `TargetGroup` only, listeners and targets inline.
4. **Secrets:** move API-key Secret handling into a controller in this effort, or leave for later?

## Findings before coding step 1 (2026-11-25)

Retiring the Tenant dual writer is **not** behavior-neutral. The API and the Tenant controller each create a
ResourceQuota and NetworkPolicy in the tenant namespace, with different contents:

| | API (`EnsureTenantNamespace`) | Operator (`ensureTenantIsolation`) |
|---|---|---|
| Quota name | `branding.ResourceQuotaName` | `virtfoundry-default` |
| Pods | `MaxVMs*2+10` = 50 | 50 |
| PVCs | 50 | 20 |
| CPU / memory | limits-less `cpu` 32, `memory` 64Gi | requests 20 / 40Gi, limits 40 / 80Gi |
| Services | not limited | 20 |
| Extra policy | CDI importer egress NetworkPolicy | default-deny + DNS + 80/443 egress |

Both quotas are enforced today (the stricter value wins per resource). Removing the API side changes effective
limits for live tenants (PVC 50 to 20; CPU/memory become the operator's requests/limits) and drops the CDI
importer policy unless the operator covers it. Needs a maintainer decision on the target quota values, then:
operator PR (quota values + CDI egress rule, additive), then core PR (stop creating namespace/quota/policy, wait for
`Tenant.status.namespace`), verified on the homelab with an import DataVolume in a fresh tenant.
No code was changed.
