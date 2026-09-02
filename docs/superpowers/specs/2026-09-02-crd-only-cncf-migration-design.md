# VirtFoundry CRD-only + CNCF Sandbox migration — design spec

Date: 2026-09-02  
Status: **draft** (approved direction; implementation phased, **not 1.0**)  
Supersedes: [2026-09-01-crd-operator-design.md](./2026-09-01-crd-operator-design.md) §3 “REST stays forever” and §4.0 adoption-layer table — REST becomes an **internal UI gateway**, not the public control-plane API.  
Repos: `virtfoundry/operator`, `virtfoundry/core`, `virtfoundry/helm-charts`, `virtfoundry/terraform-provider-virtfoundry`, `Matheus-Thurler/argo-homelab`

**Baseline shipped:** `0.6.0` — CRD store default, MySQL/worker removed, operator Tenant + Instance status, homelab green.

---

## 1. Goal

Make **`virtfoundry.io` CRDs + operator** the only public control-plane contract — the shape CNCF Sandbox reviewers expect from a Kubernetes-native project — while keeping the **current UI pixel-perfect** (same screens, flows, console, snapshots).

**Success (Sandbox-shaped, pre-1.0):**

| Audience | Primary interface | Secondary |
|----------|-------------------|-----------|
| Platform / GitOps | `kubectl`, Argo CD, Terraform (`virtfoundry` provider → K8s API) | — |
| Tenant admin / demo | Web UI (unchanged UX) | — |
| CNCF reviewer | Install in &lt;30 min, `kubectl get vf-*`, GitOps YAML in docs | 10-min demo video |

**Not success yet:** declaring **1.0** or removing the UI gateway before every flow is validated on homelab + kind.

---

## 2. Non-goals

- **1.0** stability contract (explicitly deferred).
- Removing the web UI or degrading current UX.
- Replacing `terraform-provider-virtfoundry` with generic `hashicorp/kubernetes` manifests.
- OIDC-only auth in the first migration slices (JWT + API keys stay for UI gateway).
- Bundling KubeVirt/Multus/CDI inside the VirtFoundry chart (stay cluster prerequisites).
- MySQL revival or dual-write.

---

## 3. CNCF narrative (Sandbox proposal language)

> **Desired state** lives in `virtfoundry.io/v1alpha1` CustomResources. The **operator** reconciles Tenant namespaces, Multus NADs, PVCs, and KubeVirt VirtualMachines. Humans use the **web UI**; automation uses **kubectl**, **Argo CD**, or the **VirtFoundry Terraform provider** (Kubernetes API backend). There is no external platform database.

Alignment checklist:

| CNCF signal | VirtFoundry answer |
|-------------|-------------------|
| Kubernetes-native extension | CRD group `virtfoundry.io`, kubebuilder operator |
| Clear separation of concerns | Operator = actuator; CRs = desired state |
| GitOps-friendly | All entities `kubectl apply` / Argo Application |
| No control-plane DB | etcd only (+ Secrets for credential hashes) |
| Documented install | One quickstart &lt;30 min (kind profile) |
| Multiple clients | UI, TF provider, kubectl — same CRs underneath |

**UI gateway is not a weakness** if documented as an optional ergonomic layer (like many projects ship `kubectl` + dashboard). Public API surface for integrators is **CRDs**, not `/api/v1`.

---

## 4. Target architecture (radical CRD-first)

```
                    ┌─────────────────────────────────────────┐
                    │           kube-apiserver (etcd)          │
                    │     virtfoundry.io/v1alpha1 CRDs         │
                    └─────────────────┬───────────────────────┘
                                      │
         ┌────────────────────────────┼────────────────────────────┐
         │                            │                            │
         ▼                            ▼                            ▼
  kubectl / Argo CD          terraform-provider-virtfoundry    virtfoundry-operator
  (GitOps, samples)          (client-go, typed CRUD)           (reconcile → KubeVirt,
                                                                    Multus, CDI, NP, PVC)
         │                            │
         │                            │
         └────────────┬───────────────┘
                      │
                      ▼
            virtfoundry-ui-gateway  (rename mentally: cmd/server)
            • JWT / session / tenant scope
            • REST /api/v1 → CR read/write (no hypervisor calls)
            • Console WS, logs, imperative VM actions (subresources)
            • Informer cache → fast lists (no etcd write on list)
                      │
                      ▼
                 React UI (unchanged)
                 platform-api.ts → /api/v1
```

### 4.1 Component roles

| Component | Role | Public? |
|-----------|------|---------|
| **CRDs** | Source of truth, OpenAPI schema, CEL validation | **Yes** |
| **Operator** | All infra side effects | **Yes** (binary + chart) |
| **UI gateway** (`core` API) | Auth + REST facade + console/imperative | Internal (UI only) |
| **UI** | React SPA | **Yes** (product) |
| **TF provider** | Automation | **Yes** (Registry) |

### 4.2 What disappears from `core` over time

| Remove from API path | When | Moved to |
|----------------------|------|----------|
| Hypervisor driver (`internal/infra/hypervisor`) | 0.7 | `operator` controllers |
| Direct KubeVirt client in API handlers | 0.7 | Operator + status on Instance CR |
| REST as documented integration API | 0.8 | CRD docs + TF provider |
| Duplicate business logic in services | 0.7–0.8 | Operator validation webhooks (optional) |

### 4.3 UI preservation contract

**Zero React route/screen changes** during 0.7–0.8 unless a bugfix requires it.

- `ui/src/lib/platform-api.ts` keeps calling `/api/v1/*`.
- Gateway implements the **same JSON shapes** (`PlatformVM`, `Tenant`, …) by mapping CR `spec`/`status` + `metadata.uid`.
- Performance: gateway uses **shared informers** per resource type; list endpoints never call `SaveVM` or sync loops.
- Console: gateway proxies KubeVirt subresource (already tenant-scoped); no UI change.

Optional later (0.9+): SSE/WebSocket push from gateway informers → UI drops polling; still `/api/v1`.

---

## 5. Terraform provider — dois modos de auth (permanentes)

Keep **`terraform-provider-virtfoundry`**. CRDs são a fonte de verdade; o **transporte e a auth** dependem de **quem** roda o Terraform.

**Problema:** `kubeconfig` com cluster-admin dá acesso total — inaceitável para automação **multi-tenant** com API keys escopadas.

**Solução:** dois personas, dois modos — **não** é bridge temporário.

```
┌─────────────────────────────────────────────────────────────────┐
│ Persona A — Platform / GitOps (SRE, root, módulo tenant)        │
│ Auth: kubeconfig + RBAC (ClusterRole ou SA de plataforma)       │
│ Transporte: client-go → kube-apiserver → CRDs                   │
│ Escopo: criar Tenant, catálogo, qualquer namespace              │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ Persona B — Tenant automation (CI do cliente, tenant_admin)     │
│ Auth: api_key (ou JWT) — OU kubeconfig de SA só do tenant       │
│ Transporte B1: HTTP → UI gateway → CRs (tenant_id + scopes)     │
│ Transporte B2: client-go → SA com Role no NS do tenant          │
│ Escopo: só virtfoundry-tenant-<slug>; sem cross-tenant          │
└─────────────────────────────────────────────────────────────────┘
```

### 5.1 Provider configure

**Platform (GitOps, módulos `tenant` / bootstrap):**

```hcl
provider "virtfoundry" {
  config_path = "~/.kube/config"
  context     = "homelab"
  # SA deve ter ClusterRole virtfoundry-platform-admin (template no chart)
}
```

**Tenant (CI, usuário com API key — sem cluster-admin):**

```hcl
provider "virtfoundry" {
  endpoint = "https://virtfoundry.homelab"
  api_key  = var.virtfoundry_api_key   # vfd_live_... scopes + tenant embutidos
  # gateway valida key, aplica tenant_id, escreve só CRs permitidos
}
```

**Tenant avançado (100% K8s RBAC, sem gateway):**

```hcl
provider "virtfoundry" {
  config_path    = var.tenant_kubeconfig   # SA só em virtfoundry-tenant-acme
  config_context = "tenant-acme-automation"
}
```

Documentar os três no README do provider; Sandbox demo usa **platform** no install script e **api_key** no exemplo `examples/vm` tenant.

### 5.2 O que cada modo faz no backend

| Modo | Quem valida permissão | Onde state vive | Multi-tenant |
|------|------------------------|-----------------|--------------|
| `config_path` (platform SA) | Kubernetes RBAC | etcd (CRDs) | N/A — operador de plataforma |
| `config_path` (tenant SA) | Role no namespace do tenant | etcd (CRDs) | Sim — SA não vê outros NS |
| `api_key` + `endpoint` | Gateway (scopes + tenant_id do key) | etcd via gateway escrevendo CRs | Sim — igual UI/REST hoje |

O gateway **não** é “REST legado” para tenant: é **fronteira de segurança** que traduz API key → operações CRD escopadas (mesma lógica `AuthenticateAPIKey` + `FilterScopes` que já existe no core).

### 5.3 Implementação no provider (0.7+)

| Recurso TF | Platform `kubeconfig` | Tenant `api_key` | Tenant SA `kubeconfig` |
|------------|-------------------------|------------------|-------------------------|
| `virtfoundry_tenant` | ✅ create | ❌ (sem permissão) | ❌ |
| `virtfoundry_instance` | ✅ any NS | ✅ só tenant do key | ✅ só NS do SA |
| `virtfoundry_vpc`, `network`, … | ✅ | ✅ escopado | ✅ NS do SA |
| Data sources (offerings, templates) | ✅ cluster get | ✅ read-only via gateway | ✅ get cluster-scoped |

Provider escolhe client no `Configure()`:

- `endpoint` set → HTTP client (tenant path)
- `config_path` set → `client-go` (platform ou tenant SA — RBAC do cluster decide)

### 5.4 RBAC templates (Helm / docs)

**Platform** — `ClusterRole virtfoundry-platform-admin`:

```yaml
rules:
  - apiGroups: ["virtfoundry.io"]
    resources: ["tenants", "offerings", "users", "templates"]
    verbs: ["*"]
  - apiGroups: ["virtfoundry.io"]
    resources: ["instances", "disks", "vpcs", "networks", "securitygroups", ...]
    verbs: ["*"]   # all namespaces
  - apiGroups: [""]
    resources: ["namespaces", "secrets"]
    verbs: ["get", "list", "create", "update", "patch", "delete"]
```

**Tenant automation SA** — `Role` em `virtfoundry-tenant-<slug>`:

```yaml
rules:
  - apiGroups: ["virtfoundry.io"]
    resources: ["instances", "disks", "vpcs", "networks", "securitygroups", "sshkeys", "instancesnapshots", "disksnapshots", "apikeys", "roles"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: ["virtfoundry.io"]
    resources: ["offerings", "templates"]
    verbs: ["get", "list", "watch"]   # catálogo cluster-scoped, read-only
```

Módulo Terraform `modules/tenant-credentials` (0.8): cria SA + Role + Secret token; output kubeconfig para Persona B2.

### 5.5 API keys vs kubeconfig tenant — quando usar

| Critério | API key (gateway) | Tenant SA kubeconfig |
|----------|-------------------|----------------------|
| UX para tenant | Melhor — URL + key, sem K8s | Exige entregar kubeconfig/SA |
| CNCF / GitOps puro | Camada extra (gateway) | Mais nativo (só RBAC) |
| Rotação | Revoke key na UI | Rotacionar Secret do SA |
| Scopes granulares | `scopes[]` no APIKey CR | Só o que o Role permite |
| Sandbox narrative | “CRDs + operator”; gateway = UI/automation edge | “CRDs + RBAC”; máximo K8s |

**Recomendação:** manter **ambos** permanentemente. Documentar API key como default tenant; SA kubeconfig como “GitOps-native tenant”. **Não** remover `endpoint`/`api_key` no caminho a 1.0.

### 5.6 O que muda vs spec anterior

- ~~“REST deprecated, só kubeconfig”~~ → **incorreto para multi-tenant**
- Público integrator API = **CRDs**; auth de tenant = **API key (gateway)** ou **tenant SA RBAC**
- Platform Terraform = **kubeconfig** com role explícita, nunca cluster-admin em exemplos

---

## 6. Operator completion (blocking CRD-only)

**0.6.0 gap:** Tenant + Instance **status** only. API still triggers some VM lifecycle.

| Controller | Priority | 0.7 milestone |
|------------|----------|---------------|
| Instance create/update/delete → KubeVirt VM | **P0** | API stops calling hypervisor |
| Network → Multus NAD | **P0** | VPC/network GitOps works |
| VPC → VPC namespace | **P0** | |
| Disk → PVC | **P0** | |
| DiskSnapshot → VolumeSnapshot | **P1** | |
| InstanceSnapshot → VMSnapshot | **P1** | homelab validated |
| SecurityGroup → NetworkPolicy | **P1** | |
| Template ISO → CDI DataVolume | **P1** | |
| IPAddress allocation | **P2** | |
| User/APIKey → Secret | **P2** | API bootstrap only until then |

**Validation gate (0.7 exit):** deploy `fedora-primary` equivalent from **only** an `Instance` CR + `kubectl`; UI shows it without API-side KubeVirt calls.

---

## 7. Performance model

| Path | Latency drivers | 0.6.0 | Target |
|------|-----------------|-------|--------|
| UI list VMs | Gateway → store → etcd | Informer + CR fast path | Gateway informer only |
| UI create VM | REST → CR → operator → KubeVirt | API writes CR | Same; API thin |
| TF apply | REST × N resources | 2 hops | 1 hop (K8s API) |
| GitOps sync | Argo → CR → operator | Works for Tenant | All kinds |
| Operator reconcile | workqueue depth | Low | Scales with HA replicas |

**Footprint (homelab target):**

| Pod | Replicas | Notes |
|-----|----------|-------|
| virtfoundry-operator | 1 (HA optional) | ~128–256Mi |
| virtfoundry-api (gateway) | 1 | Shrinks when hypervisor code removed |
| virtfoundry-ui | 1 | Static nginx |
| MySQL / worker | **0** | Gone since 0.6.0 |

---

## 8. Quick install — Sandbox demo path

**Goal:** cold reviewer → UI login + running VM in **≤30 minutes** on kind (no VLAN).

### 8.1 One-command target (0.7)

```bash
# Meta entry (script or Make target in helm-charts)
curl -fsSL https://virtfoundry.github.io/helm-charts/scripts/install-kind.sh | bash
# OR
make -C helm-charts install-kind PROFILE=quickstart
```

Script order (idempotent):

1. kind cluster + local-path (or bundled Longhorn optional profile)
2. KubeVirt + CDI + Multus (`scripts/setup/*.sh`)
3. `helm install virtfoundry-operator …`
4. `helm install virtfoundry …` (secrets from env or flags)
5. Wait + print URL (`http://localhost:8080`) + `root` password
6. Optional: `kubectl apply -f samples/quickstart/instance.yaml`

### 8.2 Docs structure (Sandbox packet)

| Doc | Purpose |
|-----|---------|
| [quickstart.md](https://virtfoundry.github.io/helm-charts/docs/guide/quickstart/) | UI path |
| [kind.md](https://virtfoundry.github.io/helm-charts/docs/guide/kind.md) | Laptop |
| **NEW:** `guide/gitops-quickstart.md` | `kubectl apply` tenant + VM in 5 CRs |
| **NEW:** `samples/gitops/` in operator repo | Copy-paste YAML |
| CNCF proposal appendix | Architecture diagram + install timing screenshot |

### 8.3 Demo script (10 min video — Phase 1 CNCF)

1. Run kind install script (2 min)
2. Login UI → dashboard (1 min)
3. `kubectl get vf-instance,vf-tenant -A` (30 s)
4. Deploy VM from UI → Running → console (4 min)
5. VM snapshot → `kubectl get vmsnapshot` (2 min)
6. One `terraform apply` from `examples/vm` with kubeconfig (30 s teaser)

---

## 9. Release phases (test as we go — no 1.0 yet)

| Version | Theme | Exit test (homelab + kind) |
|---------|-------|----------------------------|
| **0.6.0** ✅ | CRD store default, operator bootstrap | No MySQL; UI + 2 VMs + snapshots |
| **0.7.0** | Operator owns VM/network/disk create; API = gateway; TF K8s mode beta | VM from Instance CR only; TF creates tenant+VM via kubeconfig |
| **0.7.x** | Remaining controllers P1; gateway informers all lists | GitOps quickstart doc runnable |
| **0.8.0** | REST **tenant** path stays (`api_key`); platform TF uses kubeconfig + RBAC templates; `samples/` complete | Tenant CI uses api_key; platform uses SA — neither uses cluster-admin in docs |
| **0.8.x** | Validating webhooks (CEL + optional admission); operator CI + digest write-back | Argo homelab fully digest-pinned |
| **0.9.0** | Sandbox proposal draft; demo video; 2+ adopters | Phase 2 CNCF checklist green |
| **1.0.0** | **Explicit later** — stability promise, REST removal optional | TBD after Sandbox feedback |

**Rule:** no version bump without homelab Argo sync + kind quickstart smoke.

---

## 10. Testing gates (each PR slice)

| Layer | Command |
|-------|---------|
| Operator | `make test`, envtest reconcile |
| Core gateway | Go integration tests against envtest CRDs |
| UI | Existing vitest + E2E skill flows |
| Helm | `make lint` |
| Homelab | Argo sync; `kubectl get pods -n virtfoundry-system` |
| Kind quickstart | Script from clean kind cluster |
| Terraform | `terraform apply` + `destroy` in `examples/vm` (kubeconfig mode) |

---

## 11. Documentation updates (parallel track)

- Profile README: “Canonical API: `virtfoundry.io` CRDs”
- Update [2026-09-01 spec](./2026-09-01-crd-operator-design.md) header → **partially superseded**
- ROADMAP: add 0.7–0.9 CRD-only milestones
- CNCF-CHECKLIST Phase 2: link this spec + gitops quickstart
- Terraform provider README: kubeconfig first, REST deprecated

---

## 12. Risks and mitigations

| Risk | Mitigation |
|------|------------|
| UI regression | Freeze `platform-api.ts` contract; E2E on homelab |
| TF users on REST | Tenant keeps `api_key`; platform gets RBAC templates; never document cluster-admin for tenants |
| Operator bug blocks all creates | Status conditions + Events; keep gateway read-only fallback |
| Sandbox “not enough contributors” | good first issues on operator controllers |
| CRD v1alpha1 breaking changes | Stay pre-1.0; document in CHANGELOG per minor |

---

## 13. Immediate next steps (0.7.0 kickoff)

1. **Operator:** Instance controller (create/delete KubeVirt VM from spec).
2. **Core:** Remove hypervisor calls from deploy/start/stop handlers → patch Instance CR spec/status only.
3. **TF provider:** Add `config_path`; implement `virtfoundry_instance` via Instance CR.
4. **helm-charts:** `scripts/install-kind.sh` + `samples/quickstart/`.
5. **Docs:** `guide/gitops-quickstart.md` with 5 YAML files.

---

## Appendix A — CRD public API examples (GitOps)

```yaml
apiVersion: virtfoundry.io/v1alpha1
kind: Tenant
metadata:
  name: acme
spec:
  displayName: Acme Corp
  slug: acme
---
apiVersion: virtfoundry.io/v1alpha1
kind: Instance
metadata:
  name: web-01
  namespace: virtfoundry-tenant-acme
spec:
  displayName: web-01
  offeringRef: small
  templateRef: fedora-40
  dedicatedCPU: false
```

```bash
kubectl get vf-tenant,vf-instance -A
kubectl describe vf-instance web-01 -n virtfoundry-tenant-acme
```

## Appendix B — UI gateway rename (optional, 0.8)

Chart label: `app.kubernetes.io/component=ui-gateway` (Deployment name can stay `virtfoundry-api` for compatibility). Docs call it **UI gateway** to avoid implying a public REST product.
