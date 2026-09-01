# CRD Operator Cutover Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace MySQL platform state with `virtfoundry.io` CRDs; ship `virtfoundry/operator` as sole infra reconciler; keep REST `/api/v1` as adoption facade.

**Architecture:** New kubebuilder operator owns CRDs + controllers (Namespace, NAD, NetPol, PVC, KubeVirt). Core implements `store.Repository` via Kubernetes client. Helm drops MySQL/worker. Homelab greenfield via Argo (operator wave 5, API wave 6).

**Tech Stack:** Go 1.25+, kubebuilder v4, controller-runtime, envtest, kind, Helm 3, k8s client-go aligned with core (`v0.34.x`).

**Spec:** [`docs/superpowers/specs/2026-09-01-crd-operator-design.md`](../specs/2026-09-01-crd-operator-design.md)

## Progress (2026-09-01 homelab slice)

| Area | Status |
|------|--------|
| Operator repo + v1alpha1 CRDs | Shipped on `virtfoundry/operator` |
| Tenant namespace reconciler | Done |
| Instance status sync (KubeVirt → CR) | Done |
| Core `store.Repository` kubernetes driver | Done (`feat/crd-store`) |
| Helm `store.driver=kubernetes`, drop mysql/worker | Done |
| Argo homelab operator app + values | Done |
| Homelab validation (no MySQL, VMs Running, VNC, API latency) | Done |
| VPC/Network/Disk/Instance create-delete controllers | Not started |
| Remove hypervisor from API path | Not started |
| Operator CI + digest write-back to argo-homelab | Not started |

## Global Constraints

- API group `virtfoundry.io`, version `v1alpha1`
- kubectl shortNames hyphenated `vf-*` (e.g. `vf-tenant`, `vf-vpc`, `vf-instance`)
- Kind names PascalCase without `Vf` prefix; never Kind `VirtualMachine`
- Finalizer `virtfoundry.io/finalizer` on CRs that own infra
- Labels: `app.kubernetes.io/part-of=virtfoundry`, `virtfoundry.io/tenant=<slug>` when applicable
- Credential hashes only in Secrets (`secretRef` on User/APIKey)
- REST `/api/v1` stays; no MySQL dual-write; no live MySQL→CR exporter
- Conventional Commits, English; no commit to `main` without PR
- **Local gate:** run and pass tests locally before opening PRs
- Homelab deploy still via Argo after merge (no default `kubectl rollout restart`)

## File structure (target)

```
virtfoundry/operator/                    # NEW repo (local first)
├── api/v1alpha1/                        # CRD Go types
├── internal/controller/                 # one file per Kind
├── cmd/main.go
├── config/crd/bases/
├── charts/virtfoundry-operator/
├── test/                                # envtest + kind smoke
├── LICENSE, CONTRIBUTING.md, SECURITY.md, CODE_OF_CONDUCT.md
└── Makefile

virtfoundry/core/
├── internal/platform/store/kubernetes.go
├── internal/platform/store/mapping/     # CR ↔ platform.* 
├── (delete mysql.go, memory.go, migrations/, cmd/worker)

virtfoundry/helm-charts/charts/virtfoundry/
├── (delete mysql.yaml, worker.yaml; expand rbac.yaml)
```

## Execution notes

- Work locally under `/Users/matheusthurler/Documents/github/virtfoundry/`
- Operator: create GitHub repo only when opening the first PR (after local approval)
- Core docs branch `docs/crd-operator-design` holds this plan + spec until docs PR approved
- Implementation on core uses a new branch `feat/crd-store` when Phase 4 starts (do not mix with unrelated dirty files on working tree)

---

### Task 1: Scaffold operator repo (local)

**Files:**
- Create: `/Users/matheusthurler/Documents/github/virtfoundry/operator/` (entire tree via kubebuilder)
- Create: `operator/LICENSE`, `operator/CONTRIBUTING.md`, `operator/SECURITY.md`, `operator/CODE_OF_CONDUCT.md`, `operator/README.md`

**Interfaces:**
- Consumes: nothing
- Produces: module `github.com/virtfoundry/operator`, `make test` / `make build` baselines

- [ ] **Step 1: Create directory and kubebuilder project**

```bash
mkdir -p /Users/matheusthurler/Documents/github/virtfoundry/operator
cd /Users/matheusthurler/Documents/github/virtfoundry/operator
kubebuilder init \
  --domain virtfoundry.io \
  --repo github.com/virtfoundry/operator \
  --project-name virtfoundry-operator \
  --license apache2 \
  --owner "The VirtFoundry Authors"
```

Expected: `go.mod`, `Makefile`, `cmd/main.go`, `config/`, `PROJECT` exist.

- [ ] **Step 2: Pin Go / k8s versions to match core**

In `go.mod`, ensure:

```
module github.com/virtfoundry/operator

go 1.25.0
```

Then:

```bash
go get k8s.io/api@v0.34.3 k8s.io/apimachinery@v0.34.3 k8s.io/client-go@v0.34.3
go mod tidy
```

- [ ] **Step 3: Add community files**

Copy Apache-2.0 text from `core/LICENSE` (keep Copyright 2026 The VirtFoundry Authors).

`CONTRIBUTING.md`: English, Conventional Commits, branch `feat|fix|docs|chore`, point issues to `virtfoundry/operator`.

`SECURITY.md`: private advisory; remove MySQL deployment notes; say platform state is CRDs + Secrets.

`CODE_OF_CONDUCT.md`: copy from `core/CODE_OF_CONDUCT.md`.

`README.md` minimum:

```markdown
# VirtFoundry Operator

Kubernetes operator for VirtFoundry private cloud (`virtfoundry.io` CRDs).

Canonical desired state lives in CRs. The REST API / UI in
[virtfoundry/core](https://github.com/virtfoundry/core) are optional clients.

## Develop

```bash
make generate manifests
make test
make build
```

## Install (kind)

```bash
make install
make deploy IMG=ghcr.io/virtfoundry/operator:dev
```
```

- [ ] **Step 4: Verify baseline**

```bash
cd /Users/matheusthurler/Documents/github/virtfoundry/operator
make test
make build
```

Expected: PASS (scaffold suite), binary builds.

- [ ] **Step 5: Git init local + first commit (no push)**

```bash
cd /Users/matheusthurler/Documents/github/virtfoundry/operator
git init -b main
git add .
git commit -m "$(cat <<'EOF'
chore: scaffold virtfoundry operator with kubebuilder

Apache-2.0 operator repo for virtfoundry.io CRDs.
EOF
)"
git checkout -b feat/crd-v1alpha1
```

---

### Task 2: Tenant CRD (`vf-tenant`)

**Files:**
- Create: `operator/api/v1alpha1/tenant_types.go`
- Create: `operator/api/v1alpha1/groupversion_info.go` (if not present)
- Create: `operator/api/v1alpha1/zz_generated.deepcopy.go` (generated)
- Create: `operator/config/crd/bases/virtfoundry.io_tenants.yaml` (generated)
- Modify: `operator/cmd/main.go` (scheme registration after create API)

**Interfaces:**
- Consumes: kubebuilder scaffold
- Produces: types `Tenant`, `TenantSpec`, `TenantStatus`, `TenantList`; shortName `vf-tenant`

- [ ] **Step 1: Create API**

```bash
cd /Users/matheusthurler/Documents/github/virtfoundry/operator
kubebuilder create api \
  --group virtfoundry \
  --version v1alpha1 \
  --kind Tenant \
  --resource \
  --controller=false \
  --namespaced=false
```

Answer yes to create resource; skip controller for this step (Task 3 adds it) **or** create both and replace controller in Task 3.

- [ ] **Step 2: Write Tenant types**

Replace generated `api/v1alpha1/tenant_types.go` with:

```go
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantSpec is the desired state of a Tenant.
type TenantSpec struct {
	// Display name shown in UI/REST.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// DNS-1123 slug; drives Namespace virtfoundry-tenant-{slug}.
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=63
	Slug string `json:"slug"`

	// ImportMetadata optional CloudStack/other import identity.
	// +optional
	Import *ImportMeta `json:"import,omitempty"`
}

// ImportMeta carries external identity for migrations.
type ImportMeta struct {
	// +optional
	ExternalUUID string `json:"externalUUID,omitempty"`
	// +optional
	Source string `json:"source,omitempty"`
}

// TenantStatus is the observed state of a Tenant.
type TenantStatus struct {
	// Phase is Pending|Ready|Failed|Terminating.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Namespace is the tenant workload namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=vf-tenant
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.status.namespace`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Tenant is the Schema for the tenants API.
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantSpec   `json:"spec,omitempty"`
	Status TenantStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TenantList contains a list of Tenant.
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tenant{}, &TenantList{})
}
```

- [ ] **Step 3: Generate manifests**

```bash
make generate manifests
```

Expected: CRD YAML contains `shortNames: ["vf-tenant"]` and `scope: Cluster`.

- [ ] **Step 4: Commit**

```bash
git add api/ config/
git commit -m "feat(api): add Tenant CRD with vf-tenant shortName"
```

---

### Task 3: Tenant controller (TDD)

**Files:**
- Create: `operator/internal/controller/tenant_controller.go`
- Create: `operator/internal/controller/tenant_controller_test.go`
- Create: `operator/internal/controller/suite_test.go` (envtest suite if missing)
- Modify: `operator/cmd/main.go` — register reconciler

**Interfaces:**
- Consumes: `v1alpha1.Tenant`
- Produces: Namespace `virtfoundry-tenant-{slug}`; status.phase `Ready`; finalizer `virtfoundry.io/finalizer`

- [ ] **Step 1: Write failing reconcile test**

```go
package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	virtfoundryv1alpha1 "github.com/virtfoundry/operator/api/v1alpha1"
)

var _ = Describe("Tenant Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("when creating a Tenant", func() {
		It("creates namespace virtfoundry-tenant-{slug} and sets Ready", func() {
			ctx := context.Background()
			key := types.NamespacedName{Name: "acme"}

			tenant := &virtfoundryv1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{Name: "acme"},
				Spec: virtfoundryv1alpha1.TenantSpec{
					Name: "Acme Corp",
					Slug: "acme",
				},
			}
			Expect(k8sClient.Create(ctx, tenant)).To(Succeed())

			By("reconciling")
			r := &TenantReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).NotTo(HaveOccurred())

			Eventually(func(g Gomega) {
				ns := &corev1.Namespace{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "virtfoundry-tenant-acme"}, ns)).To(Succeed())
				g.Expect(ns.Labels["app.kubernetes.io/part-of"]).To(Equal("virtfoundry"))
				g.Expect(ns.Labels["virtfoundry.io/tenant"]).To(Equal("acme"))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				got := &virtfoundryv1alpha1.Tenant{}
				g.Expect(k8sClient.Get(ctx, key, got)).To(Succeed())
				g.Expect(got.Status.Phase).To(Equal("Ready"))
				g.Expect(got.Status.Namespace).To(Equal("virtfoundry-tenant-acme"))
				g.Expect(got.Finalizers).To(ContainElement("virtfoundry.io/finalizer"))
			}, timeout, interval).Should(Succeed())
		})
	})
})
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
make test
```

Expected: FAIL (missing reconciler logic / Namespace not created).

- [ ] **Step 3: Implement TenantReconciler**

```go
package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	virtfoundryv1alpha1 "github.com/virtfoundry/operator/api/v1alpha1"
)

const tenantFinalizer = "virtfoundry.io/finalizer"

type TenantReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	tenant := &virtfoundryv1alpha1.Tenant{}
	if err := r.Get(ctx, req.NamespacedName, tenant); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	nsName := fmt.Sprintf("virtfoundry-tenant-%s", tenant.Spec.Slug)

	if !tenant.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(tenant, tenantFinalizer) {
			ns := &corev1.Namespace{}
			err := r.Get(ctx, client.ObjectKey{Name: nsName}, ns)
			if err == nil {
				if err := r.Delete(ctx, ns); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{Requeue: true}, nil
			}
			if !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(tenant, tenantFinalizer)
			if err := r.Update(ctx, tenant); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(tenant, tenantFinalizer) {
		controllerutil.AddFinalizer(tenant, tenantFinalizer)
		if err := r.Update(ctx, tenant); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ns, func() error {
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		ns.Labels["app.kubernetes.io/part-of"] = "virtfoundry"
		ns.Labels["virtfoundry.io/tenant"] = tenant.Spec.Slug
		return nil
	})
	if err != nil {
		logger.Error(err, "failed to ensure namespace")
		tenant.Status.Phase = "Failed"
		_ = r.Status().Update(ctx, tenant)
		return ctrl.Result{}, err
	}

	tenant.Status.Phase = "Ready"
	tenant.Status.Namespace = nsName
	if err := r.Status().Update(ctx, tenant); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&virtfoundryv1alpha1.Tenant{}).
		Complete(r)
}
```

Register in `cmd/main.go` via kubebuilder scaffold or manual `SetupWithManager`.

- [ ] **Step 4: Run test — expect PASS**

```bash
make test
```

Expected: PASS including Tenant Controller suite.

- [ ] **Step 5: Commit**

```bash
git add internal/controller/ cmd/
git commit -m "feat(controller): reconcile Tenant to namespace"
```

---

### Task 4: Kind smoke (local)

**Files:**
- Create: `operator/config/samples/virtfoundry_v1alpha1_tenant.yaml`
- Create: `operator/test/e2e/tenant_kind_smoke.sh` (optional thin script)

**Interfaces:**
- Consumes: Task 2–3 artifacts
- Produces: documented kind verify path

- [ ] **Step 1: Sample YAML**

```yaml
apiVersion: virtfoundry.io/v1alpha1
kind: Tenant
metadata:
  name: demo
  labels:
    app.kubernetes.io/part-of: virtfoundry
spec:
  name: Demo Tenant
  slug: demo
```

- [ ] **Step 2: kind cluster + install CRDs + run operator**

```bash
kind create cluster --name virtfoundry-op || true
cd /Users/matheusthurler/Documents/github/virtfoundry/operator
make install
make run
# other terminal:
kubectl apply -f config/samples/virtfoundry_v1alpha1_tenant.yaml
kubectl get vf-tenant demo -o yaml
kubectl get ns virtfoundry-tenant-demo
```

Expected: Tenant `status.phase=Ready`, Namespace exists.

- [ ] **Step 3: Cleanup note in README**

Document `kind delete cluster --name virtfoundry-op`.

- [ ] **Step 4: Commit**

```bash
git add config/samples/ README.md test/e2e/ 2>/dev/null || true
git commit -m "test: add Tenant kind smoke sample"
```

---

### Task 5: Remaining CRD types (v1alpha1)

**Files:**
- Create under `operator/api/v1alpha1/`:
  - `offering_types.go` (cluster, shortName `vf-offering`)
  - `template_types.go` (namespaced, `vf-template`)
  - `user_types.go` (cluster, `vf-user`) — `secretRef` only
  - `role_types.go` (namespaced, `vf-role`)
  - `apikey_types.go` (namespaced, `vf-apikey`) — `secretRef` only
  - `vpc_types.go` (namespaced, `vf-vpc`)
  - `network_types.go` (namespaced, `vf-network`)
  - `securitygroup_types.go` (namespaced, `vf-sg`)
  - `disk_types.go` (namespaced, `vf-disk`)
  - `disksnapshot_types.go` (namespaced, `vf-disksnap`)
  - `instance_types.go` (namespaced, `vf-instance`)
  - `instancesnapshot_types.go` (namespaced, `vf-isnap`)
  - `sshkey_types.go` (namespaced, `vf-sshkey`)
  - `ipaddress_types.go` (namespaced, `vf-ip`)
  - Shared: `common_types.go` with `ImportMeta`, `LocalObjectRef`, `SecretKeyRef`, phase constants

**Interfaces:**
- Spec/status fields per design spec §5.3
- Object refs: `{name string, namespace string}` where cross-NS needed; same-NS use name only

- [ ] **Step 1: Add `common_types.go`**

```go
package v1alpha1

// Phase constants for status.phase.
const (
	PhasePending     = "Pending"
	PhaseReady       = "Ready"
	PhaseFailed      = "Failed"
	PhaseTerminating = "Terminating"
)

// LocalObjectRef names an object in the same namespace (or cluster scope).
type LocalObjectRef struct {
	Name string `json:"name"`
}

// SecretKeyRef points at a Secret key holding a credential hash.
type SecretKeyRef struct {
	Name string `json:"name"`
	// +optional
	Key string `json:"key,omitempty"`
}
```

Move `ImportMeta` here if duplicated; Tenant imports it.

- [ ] **Step 2: For each Kind, run**

```bash
kubebuilder create api --group virtfoundry --version v1alpha1 --kind <Kind> \
  --resource --controller=false --namespaced=<true|false>
```

Then fill Spec/Status from design §5.3 and markers:

```go
// +kubebuilder:resource:scope=Namespaced,shortName=vf-vpc
```

(Use table in design §5.1 for scope + shortName.)

Minimum InstanceSpec:

```go
type InstanceSpec struct {
	// +optional
	DisplayName string `json:"displayName,omitempty"`
	// +optional
	OfferingRef *LocalObjectRef `json:"offeringRef,omitempty"`
	// +optional
	TemplateRef *LocalObjectRef `json:"templateRef,omitempty"`
	// +optional
	Nics []InstanceNicSpec `json:"nics,omitempty"`
	// +optional
	SSHKeyRefs []LocalObjectRef `json:"sshKeyRefs,omitempty"`
	// +optional
	DedicatedCPU bool `json:"dedicatedCPU,omitempty"`
	// +optional
	Import *ImportMeta `json:"import,omitempty"`
}

type InstanceNicSpec struct {
	Name       string          `json:"name"`
	NetworkRef LocalObjectRef  `json:"networkRef"`
}

type InstanceStatus struct {
	Phase         string             `json:"phase,omitempty"`
	KubeVirtName  string             `json:"kubevirtName,omitempty"`
	IP            string             `json:"ip,omitempty"`
	ErrorMessage  string             `json:"errorMessage,omitempty"`
	Conditions    []metav1.Condition `json:"conditions,omitempty"`
}
```

UserSpec must include:

```go
type UserSpec struct {
	Username  string         `json:"username"`
	// +optional
	Email     string         `json:"email,omitempty"`
	RoleRef   LocalObjectRef `json:"roleRef"`
	// +optional
	TenantRef *LocalObjectRef `json:"tenantRef,omitempty"`
	// +optional
	State     string         `json:"state,omitempty"`
	SecretRef SecretKeyRef   `json:"secretRef"`
}
```

- [ ] **Step 3: Generate + test compile**

```bash
make generate manifests
go test ./api/...
```

Expected: PASS / compile OK; each CRD YAML has correct `shortNames`.

- [ ] **Step 4: Commit**

```bash
git add api/ config/crd/
git commit -m "feat(api): add remaining virtfoundry.io v1alpha1 CRDs"
```

---

### Task 6: Controllers for infra-owning kinds

**Files:**
- Create: `operator/internal/controller/{vpc,network,securitygroup,disk,disksnapshot,instance,instancesnapshot,template,user,apikey,ipaddress,offering,role,sshkey}_controller.go`
- Create: matching `*_controller_test.go` for kinds that own infra (at least VPC, Network, Disk, Instance)

**Interfaces:**
- Consumes: CRD types from Task 5; Multus NAD API; KubeVirt API; snapshot APIs
- Produces: reconcile loops per design §6.2

- [ ] **Step 1: Add Go deps**

```bash
go get kubevirt.io/api@v1.8.4
go get github.com/k8snetworkplumbingwg/network-attachment-definition-client@latest
go mod tidy
```

- [ ] **Step 2: VPC controller (pattern)**

Test first: create VPC in namespace `virtfoundry-tenant-acme` with `spec.cidr=10.0.0.0/16` → creates Namespace for VPC dataplane named from status convention used in core today (document exact name in code comment: match `internal/platform/k8s` VPC NS naming before deleting that package).

Implement CreateOrUpdate Namespace + finalizer + `status.phase=Ready`.

- [ ] **Step 3: Network → NAD; SecurityGroup → NetworkPolicy; Disk → PVC; DiskSnapshot → VolumeSnapshot**

Each gets envtest happy-path test. Prefer fake client when CRD of Multus/KubeVirt is heavy; use envtest + install those CRDs for Instance.

- [ ] **Step 4: Instance → kubevirt.io VirtualMachine**

Port logic from `core/internal/infra/hypervisor` into `operator/internal/kubevirt/` (move, adapt). Do not leave a second copy in core after Phase 4.

- [ ] **Step 5: IPAddress allocation**

List IPAddress for `networkRef`; pick free address from Network CIDR; conflict → requeue. Test with two concurrent creates (serial fake client asserting uniqueness).

- [ ] **Step 6: User/APIKey Secret ensure**

If Secret missing and API did not create it, set `Failed` condition (API owns hash write). If Secret exists, set `Ready`.

- [ ] **Step 7: `make test` PASS + commit**

```bash
git commit -m "feat(controller): reconcile platform CRDs to cluster infra"
```

---

### Task 7: Operator Helm chart + image build (local)

**Files:**
- Create: `operator/charts/virtfoundry-operator/` (Deployment, SA, ClusterRole, CRDs via `crds/` or `make helm`)
- Create: `operator/Dockerfile` (if not scaffolded)
- Modify: `operator/Makefile` `deploy` target

**Interfaces:**
- Consumes: Task 2–6
- Produces: `helm template` installable chart; local image `virtfoundry-operator:dev`

- [ ] **Step 1: Chart values**

```yaml
image:
  repository: ghcr.io/virtfoundry/operator
  tag: "0.1.0-dev"
  digest: ""
  pullPolicy: IfNotPresent
replicas: 1
leaderElection: true
resources:
  requests:
    cpu: 50m
    memory: 128Mi
```

- [ ] **Step 2: Sync CRDs into chart**

```bash
# after make manifests
mkdir -p charts/virtfoundry-operator/crds
cp config/crd/bases/*.yaml charts/virtfoundry-operator/crds/
```

- [ ] **Step 3: Local build & kind load**

```bash
docker build -t virtfoundry-operator:dev .
kind load docker-image virtfoundry-operator:dev --name virtfoundry-op
helm upgrade --install virtfoundry-operator ./charts/virtfoundry-operator \
  -n virtfoundry-system --create-namespace \
  --set image.repository=virtfoundry-operator --set image.tag=dev --set image.pullPolicy=IfNotPresent
kubectl -n virtfoundry-system rollout status deploy/virtfoundry-operator
```

Expected: Deployment ready; `kubectl get crd tenants.virtfoundry.io` exists.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(chart): add virtfoundry-operator Helm chart"
```

**Do not push / do not open PR yet.**

---

### Task 8: Core Kubernetes `Repository` (branch `feat/crd-store`)

**Files:**
- Create: `core/internal/platform/store/kubernetes.go`
- Create: `core/internal/platform/store/mapping/*.go`
- Create: `core/internal/platform/store/kubernetes_test.go`
- Modify: `core/cmd/server/main.go` — wire K8s store; drop MySQL/Memory
- Modify: `core/internal/config/config.go` — remove DSN; add kubeconfig optional path
- Delete: `mysql.go`, `memory.go`, `migrations/`, `cmd/worker/`, hypervisor/k8s usage from API

**Interfaces:**
- Consumes: generated client or unstructured/controller-runtime client for `virtfoundry.io`
- Produces: `store.Repository` implementation; REST ids = CR `metadata.uid`

- [ ] **Step 1: Branch from updated main (after docs merge) or from current main**

```bash
cd /Users/matheusthurler/Documents/github/virtfoundry/core
git fetch origin
git checkout -b feat/crd-store origin/main
# cherry-pick or rely on merged docs; do not commit unrelated ROADMAP/TODO dirt
```

- [ ] **Step 2: Failing test — SaveTenant/GetTenant**

```go
func TestKubernetesStore_TenantRoundTrip(t *testing.T) {
	// envtest or fake client with Tenant scheme
	repo := NewKubernetes(fakeClient)
	tenant := &platform.Tenant{Name: "Acme", Slug: "acme", Namespace: "virtfoundry-tenant-acme", State: "Enabled"}
	repo.SaveTenant(tenant)
	got, ok := repo.GetTenantBySlug("acme")
	if !ok || got.Slug != "acme" || got.ID == "" {
		t.Fatalf("expected tenant with uid id, got %#v ok=%v", got, ok)
	}
}
```

- [ ] **Step 3: Implement mapping + store methods used by login + tenants first**

Order: User+Secret (login), Tenant, VPC, Network, Instance, … until `Repository` compiles.

- [ ] **Step 4: Remove MySQL path**

`go.mod`: drop `github.com/go-sql-driver/mysql`. `go test ./...` green.

- [ ] **Step 5: Contract HTTP test**

Login + create tenant returns JSON with `id`, `name`, `slug` keys unchanged.

- [ ] **Step 6: Commit locally**

```bash
git commit -m "feat(store): persist platform state in virtfoundry.io CRDs"
```

---

### Task 9: Helm-charts drop MySQL/worker

**Files:**
- Modify: `helm-charts/charts/virtfoundry/values.yaml`, `Chart.yaml`, `templates/rbac.yaml`, `templates/api.yaml`, `templates/configmap.yaml`
- Delete: `templates/mysql.yaml`, `templates/worker.yaml`
- Modify: docs under `helm-charts/docs/guide/installation.md`, `quickstart.md`

**Interfaces:**
- Consumes: operator chart as prerequisite
- Produces: `helm template` with zero MySQL/StatefulSet/worker

- [ ] **Step 1: Branch**

```bash
cd /Users/matheusthurler/Documents/github/virtfoundry/helm-charts
git checkout -b feat/drop-mysql-add-operator-prereq
```

- [ ] **Step 2: Remove mysql/worker; expand API ClusterRole**

Verbs `get/list/watch/create/update/patch/delete` on `virtfoundry.io` resources; Secrets create/get; Events list.

- [ ] **Step 3: Verify**

```bash
helm template virtfoundry ./charts/virtfoundry -f charts/virtfoundry/values.yaml | tee /tmp/vf.yaml
grep -i mysql /tmp/vf.yaml && exit 1 || true
grep -i worker /tmp/vf.yaml && exit 1 || true
grep virtfoundry.io /tmp/vf.yaml
```

Expected: no mysql/worker; RBAC includes API group.

- [ ] **Step 4: Commit locally (no PR yet)**

```bash
git commit -m "feat(chart)!: drop MySQL and worker; require operator CRDs"
```

---

### Task 10: Argo homelab overlay (local edits; PR last)

**Files:**
- Create: `argo-homelab/apps/virtfoundry-operator.yaml`
- Modify: `argo-homelab/apps/virtfoundry.yaml` — remove mysql ignoreDifferences
- Modify: `argo-homelab/platform/virtfoundry/values-homelab.yaml` — drop mysql; pin operator digest when available

- [ ] **Step 1: Draft Application wave 5**

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: virtfoundry-operator
  namespace: argocd
  annotations:
    argocd.argoproj.io/sync-wave: "5"
spec:
  project: default
  source:
    repoURL: https://github.com/virtfoundry/operator.git
    targetRevision: main
    path: charts/virtfoundry-operator
  destination:
    server: https://kubernetes.default.svc
    namespace: virtfoundry-system
  syncPolicy:
    automated:
      prune: false
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
      - ServerSideApply=true
```

- [ ] **Step 2: Document greenfield PVC delete for MySQL**

One-time: delete `virtfoundry-mysql` PVC after cutover so disk does not linger (`prune: false`).

- [ ] **Step 3: Hold PR until operator + core + helm local gates pass**

---

### Task 11: Docs PR on core (this branch) — only after maintainer says so

**Files:**
- Already: `docs/superpowers/specs/2026-09-01-crd-operator-design.md`
- This plan: `docs/superpowers/plans/2026-09-01-crd-operator.md`
- Modify: spec Status line → `approved`

- [ ] **Step 1: Mark spec approved**

Change header `Status: draft` → `Status: approved`.

- [ ] **Step 2: Local verification**

```bash
cd /Users/matheusthurler/Documents/github/virtfoundry/core
test -f docs/superpowers/specs/2026-09-01-crd-operator-design.md
test -f docs/superpowers/plans/2026-09-01-crd-operator.md
git log --oneline docs/crd-operator-design ^main
```

- [ ] **Step 3: When user approves — push + `gh pr create`**

Title: `docs: CRD operator design and implementation plan`

Body: summary of MySQL→CRD cutover; test plan = review docs only.

---

## Local verification checklist (gate before any PR)

| Gate | Command / check |
|------|-----------------|
| Operator unit/envtest | `cd operator && make test` |
| Operator build | `cd operator && make build` |
| Tenant kind smoke | Task 4 commands |
| Core tests | `cd core && go test ./...` |
| Helm template | Task 9 grep checks |
| No secret in CR samples | `grep -R password_hash config/samples` empty |

## PR order (after local approval)

1. Docs PR on `core` (`docs/crd-operator-design`)
2. Create GitHub `virtfoundry/operator` + push `feat/crd-v1alpha1` / merge path
3. `helm-charts` feat PR
4. `core` `feat/crd-store` PR
5. `argo-homelab` cutover PR (greenfield announced)

---

## Spec coverage self-check

| Spec section | Task(s) |
|--------------|---------|
| §3 locked decisions | Global Constraints + Tasks 1–10 |
| §4.0 API layers / keep REST | Task 8 (facade), non-goal elsewhere |
| §5 CRD catalog + `vf-*` | Tasks 2, 5 |
| §6 operator controllers | Tasks 3, 6 |
| §7 core store | Task 8 |
| §8 Helm + Argo | Tasks 7, 9, 10 |
| §9 testing | Tasks 3–4, 6, 8–9 |
| §10 git / no early PR | Task 11 + Execution notes |
| Secrets / IAM | Tasks 5–6, 8 |
| Greenfield / no MySQL exporter | Tasks 9–10 |

## Placeholder / consistency scan

- No TBD/TODO left in steps
- ShortNames consistently `vf-*`
- Finalizer string `virtfoundry.io/finalizer` matches controller code
- Module path `github.com/virtfoundry/operator` consistent
