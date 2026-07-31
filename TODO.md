# Nimbus Cloud

## Plataforma K8s-native
- [x] JWT auth + root bootstrap
- [x] Tenants, VPCs, Networks, SGs, VMs, Volumes, Snapshots
- [x] VM Snapshots (KubeVirt VirtualMachineSnapshot)
- [x] API `/api/v1`, UI, manifests K8s, worker
- [x] Realtime WebSocket + VM detail page
- [x] **Fase 0–3 MySQL persistence** (ver abaixo)

## MySQL — Fases implementadas (0–3)

### Fase 0 — Repository interface
- [x] `store.Repository` interface
- [x] `Memory` + `MySQL` implementam
- [x] API e worker usam o mesmo store via `store.Open()`

### Fase 1 — MySQL dedicado no namespace
- [x] StatefulSet `nimbus-mysql` em `nimbus-system`
- [x] Schema próprio `schema-nimbus.sql` (import-friendly: `external_uuid`, `import_source`)
- [x] `internal/platform/store/mysql.go` + migrations em boot
- [x] Config `database.dsn` no ConfigMap

### Fase 2 — Worker async + reconciliação
- [x] Worker lê `async_jobs` do MySQL (`ListPendingJobs`)
- [x] Jobs `deploy_vm`, `reconcile`
- [x] `ReconcileAll`: adopt orphans KubeVirt → DB, marca `Destroyed` se sumiu

### Fase 3 — Catálogo + Multus
- [x] Tabelas `service_offerings`, `vm_templates` + seed default
- [x] API `GET /service-offerings`, `GET /vm-templates`
- [x] Deploy VM com `service_offering_id`, `template_id`, `network_ids`
- [x] Multus NAD por network + NICs persistidos em `vm_nics`

## Fases guardadas (4–5) — implementar depois

### Fase 4 — Migração CloudStack → MySQL Nimbus
- [ ] `cmd/migrate` gravar em MySQL (hoje ainda usa Memory)
- [ ] Mapear offerings/templates CloudStack → catálogo Nimbus
- [ ] Import idempotente via `external_uuid` + `import_source=cloudstack`

### Fase 5 — Deploy + rede avançada
- [ ] Job `migrate` no cluster pós-deploy
- [ ] Ingress TLS
- [ ] Rede public/private + Load Balancer
- [ ] UI: seletor de offering/template na criação de VM

## Homelab
- [x] KubeVirt Snapshot CRD (feature gate)
- [x] Multus CNI + bridge nimbus-br0 (`scripts/setup-homelab-multus.sh`)
- [ ] Ingress TLS
