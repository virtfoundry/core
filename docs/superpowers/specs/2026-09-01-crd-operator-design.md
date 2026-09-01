# VirtFoundry CRD operator — design spec

Date: 2026-09-01  
Status: draft (awaiting maintainer review)  
Repos: `virtfoundry/operator` (new), `virtfoundry/core`, `virtfoundry/helm-charts`, `Matheus-Thurler/argo-homelab`

This spec records locked decisions from the 2026-09-01 design session. Implementation starts only after this file is reviewed and an implementation plan is written.

---

## 1. Goal

Replace MySQL as VirtFoundry control-plane state with Kubernetes CRDs. The operator is the only actuator against cluster infrastructure (namespaces, Multus NAD, NetworkPolicy, PVC, VolumeSnapshot, KubeVirt). Core keeps a stable REST `/api/v1` so UI and Terraform do not change. Homelab cutover is greenfield: no MySQL dump, recreate resources after deploy.

**Success:** a cluster with VirtFoundry installed has zero MySQL pods, all platform entities exist as CRs (plus Secrets for credential hashes), REST contract for UI/Terraform stays, GitOps can apply the same CRs, tests cover operator reconcile, REST mapping, Helm, and e2e.

## 2. Non-goals

- MySQL dual-write, dual-read, or a MySQL exporter.
- Keeping `cmd/worker` after the operator owns reconcile.
- Changing Terraform provider resources (they keep calling REST).
- Removing the REST `/api/v1` surface (adoption layer; see §4.0).
- OIDC / Kubernetes-native auth replacement for JWT.
- CNCF Sandbox application (checklist Phase 2 stays later).
- Bundling KubeVirt/Multus/CDI as hard chart dependencies (unchanged).
- Kind name `VirtualMachine` (KubeVirt already owns that GVK in `kubevirt.io`).
- Glued kubectl shortNames (`vfvpc`); use hyphenated `vf-*` instead.

## 3. Locked decisions

| Topic | Decision |
|-------|----------|
| Canonical API | CRDs (`virtfoundry.io`) are the source of truth + GitOps surface. |
| Adoption API | REST `/api/v1` stays (UI, Terraform, Proxmox migrants). Facade over CRs — not a second brain. |
| Remove REST? | No. Does not help CNCF Sandbox; hurts Phase 1 traction (demo, adopters, quickstart). |
| Operator home | New repo `virtfoundry/operator` (kubebuilder + controller-runtime). |
| Reconcile owner | Operator creates/updates/deletes infra. Core never talks to KubeVirt/NAD/PVC. |
| Credentials | User / Role / APIKey are CRs. `password_hash` and API key `secret_hash` live only in Secrets (`secretRef`). |
| Homelab | Greenfield rebuild. No live MySQL→CR migration tool. |
| Worker | Deleted from core and Helm after operator ships. |
| In-process Memory store | Deleted. Tests use fake client / envtest. |
| Chart layout | Operator chart is a prerequisite (Argo wave 5). VirtFoundry chart installs API+UI only (wave 6). Two Argo Applications. VirtFoundry chart does not vendor CRD YAML. |
| CRD API | Group `virtfoundry.io`, version `v1alpha1`. |
| kubectl shortNames | Hyphenated `vf-*` (e.g. `vf-vpc`, `vf-instance`). Not glued forms like `vfvpc`. |
| Cutover | No MySQL beside CRDs. `mysql.enabled` removed, not defaulted false. |

## 4. Architecture

```
Browser / Terraform
        │  REST /api/v1  (JWT, unchanged JSON)
        ▼
virtfoundry-api  (core cmd/server)
        │  controller-runtime client
        ▼
CRs in etcd  (virtfoundry.io/v1alpha1)
        │
        ▼
virtfoundry-operator
        │
        ├── Tenant            → Namespace virtfoundry-tenant-{slug}
        ├── VPC               → VPC Namespace
        ├── Network           → Multus NetworkAttachmentDefinition
        ├── SecurityGroup     → NetworkPolicy
        ├── Disk              → PVC
        ├── DiskSnapshot      → VolumeSnapshot
        ├── Instance          → kubevirt.io VirtualMachine
        ├── InstanceSnapshot  → snapshot.kubevirt.io VirtualMachineSnapshot
        ├── Template (ISO)    → CDI / PVC import
        └── User/APIKey       → Secret (hashes only)
```

GitOps (Argo/kubectl) may apply the same CRs. REST is a convenience writer, not a second source of truth.

### 4.0 API layers (CNCF narrative)

| Layer | Role | Audience |
|-------|------|----------|
| **CRD + operator** | Desired state, reconcile, GitOps | Platform / SRE / Argo |
| **REST + UI** | Ergonomics, demo, Terraform | Humans leaving Proxmox |
| **MySQL** | Gone | — |

Sandbox proposal language: *“Desired state lives in `virtfoundry.io` CRs; REST and UI are optional clients of that API.”*

Removing REST is **out of scope**. It does not accelerate CNCF Sandbox (checklist Phase 1 needs demo/UI/adopters). Cloud-native signal comes from CRDs + operator + no external DB for platform state.

Docs must show both paths: `kubectl apply` / Argo YAML **and** REST/UI quickstart.

### 4.1 Repository split

| Repo | Owns | Does not own |
|------|------|----------------|
| `virtfoundry/operator` | CRDs, generated clients, controllers, operator chart, envtest | REST, UI, JWT |
| `virtfoundry/core` | REST, UI, JWT, mapping domain structs ↔ CRs | MySQL, worker, KubeVirt driver, k8s.Manager |
| `virtfoundry/helm-charts` | Control-plane chart (API + UI), docs | Operator binary, CRD YAML source of truth |
| `argo-homelab` | Applications + homelab values, digest pins | Chart templates |

### 4.2 Identity of objects

- Kubernetes `metadata.uid` is the REST `id` (UUID). Greenfield: no `spec.id` duplicate.
- `metadata.name` is the user-facing name (tenant slug, VM name, offering name). DNS-1123.
- Import fields stay: `spec.import.externalUUID`, `spec.import.source`.
- Cross-resource refs use `name` + namespace (and optional uid) not MySQL UUIDs in new writes.
- REST JSON keeps `tenant_id`, `vpc_id`, … by reading the referenced CR uid so UI/Terraform stay stable.

## 5. CRD catalog

API group: `virtfoundry.io`  
Version: `v1alpha1` (no conversion webhook in v1 of this work)  
Printer columns: name, phase, age.

**Naming layers (do not confuse):**

| Layer | Rule | Example |
|-------|------|---------|
| Kind | PascalCase, no `Vf` prefix | `VPC`, `Instance` |
| Plural (API path) | kubebuilder default | `vpcs.virtfoundry.io` |
| shortName | Hyphenated `vf-*` | `vf-vpc`, `vf-instance` |
| `metadata.name` | User-facing DNS-1123; **no** forced `vf-` prefix | `acme-web` |

`vf-*` shortNames must not collide with KubeVirt (`vm`) or core Kubernetes types. Kind stays `Instance` (never `VirtualMachine`).

```bash
kubectl get vf-vpc -n virtfoundry-tenant-acme
kubectl get vf-instance,vf-disk -A
```

### 5.1 Scope

| Kind | Scope | Namespace | Short name |
|------|-------|-----------|------------|
| Tenant | Cluster | — | `vf-tenant` |
| Offering | Cluster | — | `vf-offering` |
| Template | Namespaced | platform catalog in `virtfoundry-system`; tenant catalog in tenant NS | `vf-template` |
| User | Cluster | — | `vf-user` |
| Role | Namespaced | system roles in `virtfoundry-system`; tenant roles in tenant NS | `vf-role` |
| APIKey | Namespaced | tenant NS (root keys in `virtfoundry-system`) | `vf-apikey` |
| VPC | Namespaced | tenant NS | `vf-vpc` |
| Network | Namespaced | tenant NS | `vf-network` |
| SecurityGroup | Namespaced | tenant NS | `vf-sg` |
| Disk | Namespaced | tenant NS | `vf-disk` |
| DiskSnapshot | Namespaced | tenant NS | `vf-disksnap` |
| Instance | Namespaced | tenant NS | `vf-instance` |
| InstanceSnapshot | Namespaced | tenant NS | `vf-isnap` |
| SSHKey | Namespaced | tenant NS | `vf-sshkey` |
| IPAddress | Namespaced | tenant NS | `vf-ip` |

Do not create a Job CR. Async work is the operator workqueue; REST job list can be derived from `status.conditions` or omitted until a later spec.

Do not create an Audit CR. REST audit list reads Kubernetes Events in the tenant namespace (`reportingController=virtfoundry`).

### 5.2 Spec vs status (pattern)

Every CR follows Kubernetes API conventions:

- `spec`: desired state only (what the user asked).
- `status.phase`: high-level (`Pending`, `Ready`, `Failed`, `Terminating`).
- `status.conditions`: `[]metav1.Condition` with types `Ready`, `Reconciling`, `Error`.
- `status` also holds observed infra names (`namespace`, `nadName`, `pvcName`, `kubevirtName`, guest IP).
- Fields that are today stored as “state” on the MySQL row move to `status` unless the user can set them (then they stay in spec).

Finalizer: `virtfoundry.io/finalizer` on every CR that owns infra. Controllers remove owned objects then drop the finalizer.

Owner references:

- Tenant (cluster) does not own the tenant Namespace via controller OwnerReference if that would garbage-collect the namespace too aggressively on CR delete — use the finalizer to delete the namespace explicitly after dependents are gone.
- Namespaced CRs: VPC / Network / SecurityGroup / Disk / Instance owned by nothing mandatory at create time; Instance may owner-ref Disks it created. Prefer labels `virtfoundry.io/tenant=<slug>` on all owned infra.

Validation: OpenAPI v3 schema on the CRD plus CEL where needed (CIDR format, CPU > 0, name DNS-1123). No validating webhook in v1 unless CEL cannot express a rule.

### 5.3 Kind mapping from today’s models

| MySQL / Go type | Kind | Spec (desired) | Status (observed) | Secret |
|-----------------|------|----------------|-------------------|--------|
| Tenant | Tenant | name, slug, import | phase, namespace | — |
| VPC | VPC | name, cidr | phase, namespace | — |
| Network | Network | name, type, cidr, gateway, vpcRef | phase, nadNamespace, nadName | — |
| SecurityGroup | SecurityGroup | name, description, vpcRef, rules[] | phase, networkPolicyName | — |
| Volume | Disk | name, sizeGi, storageClass, instanceRef | phase, pvcName | — |
| Snapshot | DiskSnapshot | name, diskRef | phase, volumeSnapshotName | — |
| PlatformVM | Instance | name, displayName, offeringRef, templateRef, nics[], sshKeyRefs[], dedicatedCPU | phase, kubevirtName, ip, nics[], errorMessage | — |
| VMSnapshot | InstanceSnapshot | name, instanceRef | phase, kubevirtSnapshotName | — |
| ServiceOffering | Offering | displayName, cpu, memoryMi, dedicatedCPU, storageTags | phase | — |
| VMTemplate | Template | displayName, image, sourceType, osType, cloudInitUserData, iso sizes, storageClass | phase, importState, isoVolumeName | — |
| User | User | username, email, roleRef, tenantRef, state | phase | Secret `password_hash` |
| RoleRecord | Role | description, isSystem, permissions[] | — | — |
| APIKey | APIKey | userRef, name, prefix, scopes, expiresAt | lastUsedAt, revokedAt | Secret `secret_hash` |
| SSHKeyPair | SSHKey | publicKey | fingerprint | — |
| IPAddress | IPAddress | networkRef, address (optional auto) | phase, instanceNicRef | — |
| AsyncJob | — | dropped | operator queue | — |
| AuditEvent | Event | — | Kubernetes Event | — |

Instance NIC in spec: `name`, `networkRef`. MAC/IP observed in status (operator + KubeVirt).

### 5.4 Secrets

- Naming: `vf-<kind>-<cr-name>` in the same namespace as the CR (`virtfoundry-system` for cluster-scoped User).
- Type: `Opaque`. Keys: `password_hash` or `secret_hash`.
- CR spec holds only `secretRef.name` (and optional `secretRef.key`).
- Operator or API creates the Secret before/with the CR. OwnerReference from Secret to User/APIKey so deleting the CR GCs the Secret.
- RBAC: API ServiceAccount can create/get Secrets in tenant namespaces and `virtfoundry-system`. UI never reads hashes.

### 5.5 Labels and annotations

Required labels on every VirtFoundry CR and owned infra:

- `app.kubernetes.io/part-of=virtfoundry`
- `virtfoundry.io/tenant=<slug>` (omit on cluster catalog Offerings with no tenant)

Annotation for REST display names that are not DNS-1123: `virtfoundry.io/display-name`.

## 6. Operator (`virtfoundry/operator`)

### 6.1 Bootstrap

- Apache-2.0 `LICENSE`
- Copy community files from core: `CODE_OF_CONDUCT.md`, `SECURITY.md`, `CONTRIBUTING.md` (point commits/PRs at this repo)
- `go.mod` module `github.com/virtfoundry/operator`
- kubebuilder init: domain `virtfoundry.io`, group `virtfoundry`
- Image: `ghcr.io/virtfoundry/operator`
- Chart in-repo: `charts/virtfoundry-operator` (CRDs via kubebuilder `config/crd`, Helm `crds/` generated at release)

### 6.2 Controllers (one reconcile loop per Kind that owns infra)

| Controller | Creates | Deletes on finalizer |
|------------|---------|----------------------|
| Tenant | Namespace `virtfoundry-tenant-{slug}` | Namespace after children gone |
| VPC | Namespace for VPC dataplane (today’s VPC NS) | that Namespace |
| Network | Multus NAD | NAD |
| SecurityGroup | NetworkPolicy in tenant NS | NetworkPolicy |
| Disk | PVC | PVC |
| DiskSnapshot | VolumeSnapshot | VolumeSnapshot |
| Instance | kubevirt.io VirtualMachine (+ NICs, cloud-init) | VirtualMachine |
| InstanceSnapshot | VirtualMachineSnapshot | VirtualMachineSnapshot |
| Template | CDI DataVolume / PVC when `sourceType=iso` | those objects |
| User / APIKey | Secret if API created hash and Secret missing | Secret via GC |

Offering, Role, SSHKey, IPAddress: little or no infra; still a controller to set `Ready` and, for IPAddress, allocate from the network CIDR using the CR as the lock (create with spec.address empty → status/spec fill with conflict retry).

### 6.3 Concurrency and IPAM

IP allocation must not use MySQL unique keys. The IPAddress CR name or a per-address object is the lock. Allocate: list IPAddress for `networkRef`, compute free address from CIDR, create CR; on already-exists, retry. Network controller does not write a giant allocated map in status as the source of truth.

### 6.4 Leader election

Enabled. Replicas: 1 in homelab values; chart allows HA later.

## 7. Core changes

### 7.1 Delete

- `internal/platform/store/mysql.go` and `migrations/`
- `store.Memory`
- `cmd/worker` and worker Helm template
- `internal/infra/hypervisor` KubeVirt driver usage from API/worker
- `internal/platform/k8s` Manager usage from API (logic moves to operator)
- `database.dsn` / MySQL config keys
- go-sql-driver/mysql dependency

Move, do not invent a second copy: hypervisor and k8s packages move into the operator repo (adapt to controller-runtime). Core does not keep a stale copy “for later”.

### 7.2 Add

- `internal/platform/store/kubernetes.go` implementing `Repository` via typed/untyped client against `virtfoundry.io`
- Mapping functions: `platform.Tenant` ↔ `Tenant` CR, including uid as `id` and Secret load for `PasswordHash` on login only
- In-cluster config (default) and kubeconfig for local `go run`
- RBAC for the API ServiceAccount: CRUD on all `virtfoundry.io` resources, create/get Secrets, list Events
- Bootstrap: if no User CR with root role, create User + Secret from `secrets.rootPassword` (same Helm secret as today)

### 7.3 REST contract

Handlers and JSON tags stay. Service layer keeps using `platform.*` structs. Only the Repository implementation changes. UI and Terraform provider are out of scope.

Known gap: REST still has no job status endpoint. Do not add one in this work unless an existing client requires `async_jobs`.

### 7.4 Local dev

`cmd/server` against a kind cluster with CRDs installed (operator chart or `make install` from operator repo). Document in core README. No Docker Compose MySQL.

## 8. Helm charts and Argo

### 8.1 Operator chart (`virtfoundry/operator`)

Installs CRDs, Deployment, ServiceAccount, ClusterRole, leader-election Role. Values: image digest, replicas, resources, probes.

### 8.2 VirtFoundry chart (`virtfoundry/helm-charts`)

- Remove `mysql:` values, `templates/mysql.yaml`, worker Deployment, DSN env.
- Description: API + UI (operator is a prerequisite).
- API env: no DSN; in-cluster Kubernetes.
- RBAC: verbs on `virtfoundry.io` and Secrets as in §7.2.
- Docs: installation order — KubeVirt/Multus/CDI (existing) → operator chart → virtfoundry chart.
- `values-kind.yaml`: no MySQL PVC.

### 8.3 Argo homelab

1. New Application `virtfoundry-operator`, sync-wave **5**, source operator chart (Git repo or published Helm).
2. Existing Application `virtfoundry` stays wave **6**; values drop MySQL; ignoreDifferences for `virtfoundry-mysql` StatefulSet removed.
3. Digest write-back: extend CI so operator image digest is pinned in `platform/virtfoundry/values-homelab.yaml` (or a sibling values file). Same `ARGO_HOMELAB_TOKEN` pattern as core.
4. Greenfield: uninstall/prune MySQL PVC after merge; recreate tenants/VMs via UI or GitOps CRs. `prune: false` today — document a one-time manual PVC delete so MySQL disk does not linger.

Do not default to `kubectl rollout restart`. Ship via PR/merge.

## 9. Testing

| Layer | Where | What |
|-------|--------|------|
| Unit | operator | Reconcile table tests with fake client: create Tenant → Namespace; delete with finalizer. |
| envtest | operator CI | CRDs load; one controller per Kind that owns infra, happy path + fail status. |
| Unit | core | `Repository` against fake client: Save/Get/List/Delete Tenant, User+Secret login hash, Instance list by tenant NS. |
| Contract | core | HTTP tests: login, create tenant, list VMs — JSON fields `id`, `name`, `tenant_id` unchanged. |
| Helm | helm-charts | `helm template` has no MySQL/StatefulSet/worker; API RBAC includes `virtfoundry.io`. |
| kind | operator CI | kind cluster, `make deploy`, apply sample Tenant YAML, assert Namespace exists. |
| e2e | core `scripts/e2e` | After Argo cutover: login, tenant, VPC, VM against homelab `BASE_URL`. No MySQL assertions. |

CI on `virtfoundry/operator`: `make test` (envtest) + kind smoke on PR.  
CI on `core`: existing Go tests + UI build; DSN-based tests removed.  
No merge of core cutover PR until operator CRDs are released and the Helm chart documents the prerequisite.

## 10. Git workflow

Do not commit to `main`. Conventional Commits, English, subject under 72 characters.

### 10.1 This spec

- Branch on `core`: `docs/crd-operator-design`
- Commit: `docs: add CRD operator design spec`
- PR to `virtfoundry/core` before implementation PRs

### 10.2 Implementation order (separate PRs / repos)

1. **Create** `github.com/virtfoundry/operator` (Apache-2.0, community files, kubebuilder skeleton). First commit on `main` of the empty repo, then feature branches.
2. **operator** `feat/crd-v1alpha1`: CRD types + generated manifests + envtest for types.
3. **operator** `feat/controllers`: reconcile loops + kind smoke.
4. **operator** chart + GHCR image + digest write-back to argo-homelab.
5. **helm-charts** `feat/drop-mysql-add-operator-prereq`: drop MySQL/worker, API RBAC, docs.
6. **core** `feat/crd-store`: Kubernetes Repository, delete MySQL/worker/hypervisor, REST contract tests.
7. **argo-homelab**: operator Application wave 5, drop MySQL ignoreDifferences/values, greenfield rebuild.

Each PR is independently reviewable and tested. Do not combine operator scaffold and core MySQL deletion in one PR.

### 10.3 Branch naming

Per core `CONTRIBUTING.md`: `feat/<name>`, `fix/<name>`, `docs/<name>`, `chore/<name>`. Same on operator and helm-charts.

## 11. CNCF alignment (this work)

In scope:

- CRDs as the **canonical** API for desired state (`spec` / `status` / conditions / finalizers).
- Operator pattern (controller-runtime, leader election).
- Secrets not in CR spec.
- REST kept as **adoption layer** (see §4.0); docs show kubectl/GitOps + REST/UI.
- Apache-2.0 on the new repo; SECURITY/COC/CONTRIBUTING.
- No floating `:latest` in homelab overlay.

Out of scope here: dropping REST, Sandbox proposal draft, multi-maintainer, conversion webhooks, Operator SDK scorecard as a merge gate (optional follow-up).

**Sandbox path (order):** finish CNCF checklist Phase 1 (demo, adopters, installability) while shipping this CRD cutover; Phase 2 proposal cites CRDs-as-truth + optional REST clients. Do not block Phase 1 on removing the HTTP API.

Update `docs/CNCF-CHECKLIST.md` and `docs/ARCHITECTURE.md` in the core PR that lands the store switch. Update helm-charts installation docs in the chart PR.

## 12. Risks

| Risk | Mitigation |
|------|------------|
| Split-brain if core still writes KubeVirt | Delete driver from core in the same core PR that enables the Kubernetes store. |
| REST `id` change vs old MySQL UUID | Greenfield; document that ids are Kubernetes uids after cutover. |
| `kubectl get vm` ambiguity | Kind `Instance`, shortName `vf-instance` (not `vm`). |
| Removing REST “for CNCF” | Rejected — hurts traction; CRD migration is the cloud-native signal. |
| Helm CRD upgrade | CRDs owned by operator chart/release; virtfoundry chart does not copy CRD YAML. |
| Homelab data loss | Accepted (greenfield). Announce before merge. |
| API RBAC too wide | ClusterRole limited to `virtfoundry.io` + Secrets in labeled namespaces; no cluster-admin. |

## 13. Open follow-ups (explicitly not this spec)

- Job CR if a public async job API is needed later.
- `spec.id` stable UUID if CloudStack import must round-trip pre-Kubernetes ids.
- Conversion webhook when promoting `v1alpha1` → `v1`.
- Kubernetes Events retention vs a dedicated audit store.
- Dropping or shrinking REST after Sandbox traction (only if evidence says kubectl-only is enough).

These are deferred on purpose (YAGNI). Changing them requires a spec revision, not silent scope creep.
