# VirtFoundry — Roadmap

## Core platform (done)

- [x] JWT auth + root bootstrap
- [x] Multi-tenant IaaS: tenants, VPCs, networks, security groups, VMs, volumes, snapshots
- [x] VM snapshots (KubeVirt VirtualMachineSnapshot)
- [x] API `/api/v1`, React UI, async worker
- [x] Realtime WebSocket + VM detail page + noVNC console
- [x] MySQL persistence (`store.Repository` — Memory + MySQL, shared by API and worker)
- [x] Async jobs: `deploy_vm`, `reconcile` (orphan adoption + destroyed detection)
- [x] VM catalog: service offerings, templates (platform + tenant), Multus NICs
- [x] Templates UI (`/templates`) with container disk and ISO import
- [x] VM deploy uses `/vm-templates` and `/service-offerings` API catalog

## Next

### CloudStack migration
- [ ] `cmd/migrate` write to MySQL (still uses Memory today)
- [ ] Map CloudStack offerings/templates → VirtFoundry catalog
- [ ] Idempotent import via `external_uuid` + `import_source=cloudstack`

### Production hardening
- [ ] Post-deploy migration job in cluster
- [ ] Ingress TLS
- [ ] Public/private networks + load balancer
