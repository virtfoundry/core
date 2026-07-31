# Modularização por domínio — análise

Pergunta: faz sentido separar o Nimbus em projetos/módulos por funcionalidade (`vm`, `network`, `user`, `tenant`, …)?

## Decisão adotada

**Separar apenas conceitualmente** — domínios claros na documentação, pacotes e pastas da UI, sem repos ou microserviços separados.

Um monorepo, um deploy (`nimbus-api` + `nimbus-worker` + `nimbus-ui`), pacotes por domínio em `internal/service/` (implementado).

---

## Resposta curta

| Abordagem | Veredicto |
|-----------|-----------|
| Pastas/pacotes por domínio **no mesmo repo** | ✅ Recomendado agora |
| Serviços Go separados (`ComputeService`, `NetworkService`, …) | ✅ Recomendado agora |
| UI organizada por feature (`features/vms/`, …) | ✅ Opcional, baixo risco |
| **Repositórios ou microserviços separados** | ❌ Cedo demais |

Não é ruim pensar em domínios — é a direção certa. O que seria ruim **hoje** é fragmentar em múltiplos deploys/repos antes de estabilizar fronteiras e completar lacunas da API.

---

## Por que separar em repos seria prematuro

### 1. Acoplamento real entre domínios

Deploy de VM **precisa** de tenant (namespace), opcionalmente network (NAD), offering e template. Não dá para isolar compute sem uma API de orquestração ou eventos entre serviços.

```
DeployVM = tenant + catalog + networks + kubevirt + store + ws broadcast
```

Microserviço de VM teria que chamar (ou duplicar lógica de) tenant, network e catálogo — ou você mantém um **orquestrador** central, que é o que já existe (`PlatformService`).

### 2. Infra compartilhada

- Um JWT, um middleware, um `store.Repository`
- Um worker que reconcilia VMs **e** processa jobs de deploy
- Um cluster K8s, um namespace `nimbus-system`, um MySQL
- RBAC ClusterRole único para API + worker

Separar repos multiplica: CI, versionamento de contratos, deploy coordenado, debugging distribuído.

### 3. Escala de time

Com um time pequeno (1–3 devs), monorepo + pacotes bem nomeados entrega 80% do benefício com 20% do custo operacional.

### 4. Lacunas ainda abertas

Antes de cortar fronteiras duras, vale fechar:
- Delete/update de rede e storage
- CRUD de usuários
- UI consumindo catálogo da API
- Auth no console WS
- Migrate CloudStack → MySQL

Refatorar pacotes **depois** disso evita retrabalho em contratos entre serviços.

---

## O que recomendo fazer agora

### Backend: fatiar `PlatformService`

Manter **um binário** (`server` + `worker`), dividir lógica em serviços de domínio:

```
internal/service/
├── platform.go          # Facade fina — delega para os domínios
├── tenant/
│   └── service.go       # CreateTenant, EnsureNamespace, bootstrap admin
├── identity/
│   └── service.go       # Login, users (futuro)
├── network/
│   └── service.go       # VPC, Network, SecurityGroup
├── storage/
│   └── service.go       # Volume, VolumeSnapshot
├── compute/
│   └── service.go       # VM, VMSnapshot, catalog read
└── jobs/
    └── service.go       # async_jobs, ReconcileAll
```

**Regras:**
- Cada serviço recebe `store.Repository` + apenas os clientes K8s que precisa
- Domínios **não** importam uns aos outros circularmente — dependências unidirecionais (compute → network, compute → tenant)
- Handlers continuam finos; injetam a facade ou serviços específicos

### Store: já está modular

`store.Repository` é uma interface grande, mas pode evoluir para interfaces menores por agregado:

```go
type TenantStore interface { ... }
type VMStore interface { ... }
// ou manter Repository único no início — menos churn
```

Só vale quebrar a interface quando houver implementações diferentes ou testes que sofram com mock gigante.

### K8s: já está modular

`internal/platform/k8s/` já separa por arquivo (`tenant.go`, `vpc.go`, …). Manter assim.

### UI: feature folders (opcional)

Hoje `pages/` já mapeia 1:1 com domínios. Evolução natural:

```
ui/src/features/
├── tenants/     # page + hooks + components locais
├── vms/
├── networks/
└── shared/      # Layout, Modal, api client
```

Não precisa de monorepo npm separado — só colocation por feature.

---

## Quando faria sentido repos/serviços separados

| Sinal | Ação |
|-------|------|
| Times distintos donos de compute vs network | Considerar serviços separados |
| Escala de API exige deploy independente de compute | Extrair worker + API compute |
| Catálogo/billing vira produto B2B separado | Repo próprio |
| SLAs diferentes (console WS vs CRUD) | Split de deploy, não necessariamente repo |

Até lá: **monorepo, múltiplos pacotes, um deploy**.

---

## Mapa de domínios proposto

| Domínio | Pacote Go | Tabelas | Infra K8s |
|---------|-----------|---------|-----------|
| **Identity** | `service/identity` | `users` | — |
| **Tenant** | `service/tenant` | `tenants` | Namespace tenant |
| **Network** | `service/network` | `vpcs`, `networks`, `security_groups` | VPC NS, NAD, NetPol |
| **Storage** | `service/storage` | `volumes`, `snapshots` | PVC, VolumeSnapshot |
| **Compute** | `service/compute` | `vms`, `vm_nics`, `vm_snapshots`, catálogo | KubeVirt VM/Snapshot |
| **Jobs** | `service/jobs` | `async_jobs` | — |

**Console** fica em `api/handler` + `ui/pages/VMConsole` — é transporte sobre compute, não domínio de negócio separado.

---

## Plano de migração incremental (sem big bang)

1. **Extrair `tenant/service.go`** — menor dependência externa
2. **Extrair `network/service.go`** — VPC/network/SG
3. **Extrair `storage/service.go`**
4. **Extrair `compute/service.go`** — o mais acoplado; por último
5. **`platform.go`** vira facade que os handlers usam (API pública estável)
6. Testes por pacote com mocks do store

Cada passo: `go test ./...`, deploy homelab, sem mudar rotas HTTP.

---

## Conclusão

Separar **conceitualmente** por vm/network/user/tenant é correto e alinhado com DDD.  
Separar **fisicamente** em projetos distintos agora adiciona complexidade operacional sem resolver o acoplamento — ele só mudaria de in-process para network calls.

O caminho pragmático: **monorepo modular** → estabilizar API → só então avaliar split de deploy se a escala ou o time exigirem.
