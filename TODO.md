# VirtForge Cloud — Roadmap

## Core platform (done)

- [x] JWT auth + root bootstrap
- [x] Multi-tenant IaaS: tenants, VPCs, networks, security groups, VMs, volumes, snapshots
- [x] VM snapshots (KubeVirt VirtualMachineSnapshot)
- [x] API `/api/v1`, React UI, async worker
- [x] Realtime WebSocket + VM detail page + noVNC console
- [x] MySQL persistence (`store.Repository` — Memory + MySQL, shared by API and worker)
- [x] Async jobs: `deploy_vm`, `reconcile` (orphan adoption + destroyed detection)
- [x] VM catalog: service offerings, templates, Multus NICs

## Next

### CloudStack migration
- [ ] `cmd/migrate` write to MySQL (still uses Memory today)
- [ ] Map CloudStack offerings/templates → VirtForge catalog
- [ ] Idempotent import via `external_uuid` + `import_source=cloudstack`

### Production hardening
- [ ] Post-deploy migration job in cluster
- [ ] Ingress TLS
- [ ] Public/private networks + load balancer
- [ ] UI: offering/template selector on VM create

Homelab deploy (optional): `make -C ../virtforge-chart deploy-homelab`
