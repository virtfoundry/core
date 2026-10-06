# Changelog

All notable changes to **VirtFoundry** (API, UI) are documented here.

Format based on [Keep a Changelog](https://keepachangelog.com/). Versioning: [SemVer](https://github.com/virtfoundry/helm-charts/blob/main/docs/project/versioning.md).

## [Unreleased]

## [0.10.0] - 2026-10-06

### Added

- **VKS (Kubernetes clusters)** — gRPC `ClusterService` on cmux `:8080` with a thin REST shim; `GetClusterSummary` for the guest cluster; console create form driven by the version/template catalog, cluster detail page, worker VMs on the Nodes tab, Workloads tab; control plane defaults to `LoadBalancer`.
- **Node image** — `ghcr.io/virtfoundry/` containerDisk images pass the allowlist; seeded `ubuntu-node-1-36-5` template (digest-pinned).

### Security

- Container images signed with cosign keyless on release.
- CodeQL, Scorecard, Dependabot (grouped monthly) and dependency review.
- RBAC contract test for the API's typed Kubernetes calls.

### Fixed

- Tenant namespaces get the operator ownership labels; backfill touches only those labels, never crashes the API when forbidden, and finds legacy Tenant CRs by `spec.slug`.
- Root credential Secret is recovered on server start.
- Seeded node template pinned to the working containerDisk digest.

### Changed

- CNCF readiness docs and supply-chain checklist updated.

## [0.9.0] - 2026-10-01

### Added

- **Realtime / gRPC** — Instance CR informer publishes full VM events; `WatchInstances` stream; UI consumes WS `power_state` for Start/Stop; poll backs off when websocket healthy; query invalidation beyond `vm.*`.
- **CRD-first compute** — isolated Multus networks on deploy; `ssh_key_id` → Instance `sshKeyRefs`; wizard cloud-init persisted; single actuator under `operatorReconcile`.
- **gRPC spike** — `InstanceService` behind cmux on `:8080`.
- **UI** — deploy wizard network cards/auto-pick, post-deploy IP/SSH/console CTAs, actions menu / template cards / tenant switcher, onboarding network step, OverflowMenu portal fix, clipboard `execCommand` fallback for one-time secrets (Weslei).

### Security

- SSH private PEM treated as one-time secret (names-only thereafter).
- CORS OPTIONS preflight on mux method-mismatch routes.
- HTTP timeouts, body limits, log tail caps.
- ContainerDisk image allowlist on deploy; events/console ticket hardening; autopermission fail-open closed.
- Platform-owned label stamped on Offerings (coordinates operator dedicatedCPU gate).

### Fixed

- Cold-path latency on `/api-keys` / snapshots (N+1 hydration, scoped lists, JWT user by username).
- Unique Network CR names per VPC; refuse delete of default VPC subnet.
- Honest List/Sync under `operatorReconcile`; CI digest write-back auth.

## [0.8.0] - 2026-09-28

### Added

- **UI day-2 polish** — deploy wizard em steps (compute → disco → rede → acesso → review), feedback de progresso no deploy, pré-checagens, Activity/Retry em Error, filtros por estado, confirm delete com digitar nome.
- **Operação na lista/detalhe** — ⌘K (command palette), clone (“Deploy like this”), bulk start/stop/delete, tags/pin/favorites (local), split view, gallery de templates, SSH one-liner, rename de display name sem Stopped, error catalog, onboarding checklist no dashboard, banner de impersonate (root), export CSV, tema “seguir sistema”.
- **Cloud-init editor local** na VM (validação YAML leve + secrets mascarados; não aplica no guest sem API).

### Fixed

- Select de chave SSH no wizard não mostra `()` quando `fingerprint` vem vazio.

## [0.7.3] - 2026-09-25

### Fixed

- **VM deploy shows Error / 0 vCPU after operator 0.7.2** — Instance CRs written by the API for pod-network (or CR-first without Multus `spec.nics`) now set `virtfoundry.io/allow-pod-network=true`, matching the operator opt-in. `ListVMs` / `SyncAllVMStates` enrich CPU/memory/template from Offering/Template refs so the UI no longer shows `0 vCPU / 0 MiB` when the Instance status has no sizing fields (background sync no longer overwrites the list cache without enrichment).

## [0.7.2] - 2026-09-25

Security release: cumulative product-security audit fixes since 0.7.1 (SSH defaults, console tickets, JWT/root secrets, CORS/WS Origin, ISO allowlist, CDI importer egress, non-root images, http(s) template→ISO).

### Security (BREAKING)

- **Do not default Linux guest SSH password to `ubuntu`** (issue [#97](https://github.com/virtfoundry/core/issues/97))
  - `BuildLinuxUserData` no longer invents `password: ubuntu` with `ssh_pwauth: true` and NOPASSWD sudo when no SSH keys are provided. Linux cloud-init requires an SSH public key **or** an explicit one-time password; otherwise create fails with a clear `400`.
  - Password SSH (`ssh_pwauth`) is enabled only when `cloud_init_password` is supplied on deploy (or an explicit `vm.default_password` / `VIRTFOUNDRY_VM_DEFAULT_PASSWORD` is configured for seed templates). SSH-key-only deploys keep `ssh_pwauth: false` and `lock_passwd: true`.
  - Seeded `ubuntu-2204` templates no longer bake a published default password. On startup, catalog seed strips historical `password: ubuntu` user-data from that template.
  - UI: SSH key is required for Linux VM create. API: optional `cloud_init_password` for lab/automation paths that cannot use keys.
  - Deploys that supply an SSH key or one-time password use the hypervisor cloud-init path (Instance CR does not yet carry `sshKeyRefs`).

### Security

- **UI/API images run non-root** (coordinates [helm-charts#42](https://github.com/virtfoundry/helm-charts/issues/42))
  - UI nginx listens on **8080** (was 80) with an unprivileged main config and `USER nginx`. Helm Service stays `80→8080`.
  - API image runs as UID `65532` (`nonroot`). Chart pod `securityContext` must match.
- **CDI importer egress NetworkPolicy in each tenant namespace** (follow-up to [#95](https://github.com/virtfoundry/core/issues/95) / [helm-charts#49](https://github.com/virtfoundry/helm-charts/issues/49))
  - `EnsureTenantNamespace` creates/updates `virtfoundry-cdi-importer-egress`, selecting CDI pods labeled `cdi.kubevirt.io=importer`. DNS to kube-dns/CoreDNS is allowed; egress to `0.0.0.0/0` and `::/0` excludes private / link-local / CGNAT ranges so DNS rebinding of an allowlisted ISO host cannot reach cluster-internal or metadata addresses over the pod network.
  - Existing tenants pick up the policy on API bootstrap (`EnsureTenant` / `BootstrapDefaultSecurityGroups`) without recreating the tenant record.
  - Docs: [VM-TEMPLATES.md](docs/VM-TEMPLATES.md#cdi-importer-egress-networkpolicy). Chart docs and operator notes live in helm-charts.

### Security (BREAKING)

- **VNC console requires `vms:console` and no longer accepts a JWT in the URL** (issue [#94](https://github.com/virtfoundry/core/issues/94))
  - `/ws/console` is mounted behind `RequirePermission(vms:console)`. A tenant viewer holding only `vms:read` can no longer open an interactive console; `tenant.operator` and `tenant.admin` still can. Cross-tenant access is unchanged — the VM is resolved inside the caller's own tenant.
  - New `POST /api/v1/vms/{name}/console-ticket` returns a single-use ticket that expires in 30 seconds and is bound to one VM and tenant. The browser opens `wss://…/ws/console?ticket=…`, so a long-lived JWT never reaches access logs, reverse-proxy logs or browser history. The endpoint is authorized on `vms:console` alone, so a custom role can grant console access without `vms:write`.
  - Credentials are read from headers only (`Authorization`, `X-API-Key`). The previous `?token=` fallback is gone from the REST API and from `/ws/console`. It remains on `/ws/events`, which has no ticket handshake and where a browser cannot set headers.
  - Clients that relied on `?token=` for REST calls or for the console must move to the `Authorization` header or to the ticket handshake.

- **Refuse to start with default `JWT_SECRET` or `ROOT_PASSWORD`** (issue [#93](https://github.com/virtfoundry/core/issues/93))
  - `JWT_SECRET` must be provided via env / Kubernetes Secret: empty, shorter than 32 characters, or equal to the historical defaults (`change-me-in-production`, `dev-secret-change-in-prod`) cause the server to exit non-zero with a clear error.
  - `ROOT_PASSWORD` is enforced only when set via env / Kubernetes Secret: at least 12 characters and not the historical default `virtfoundry` (case-insensitive).
  - An **empty** `ROOT_PASSWORD` is accepted. On a brand-new install with no root user in the store, the server generates a strong one-time password, bootstraps the root user with it, and logs it once at startup as a `WARN` — capture it from the boot log; it is not persisted in plaintext. On subsequent boots with the same empty env, the existing root hash is kept (no rotation).
  - Dev-only escape hatch: `VF_ALLOW_INSECURE_DEFAULTS=1` opts out of both checks (prints a `WARN` at startup). Never use this in production. Not exposed via the YAML config file.
  - `docker/Dockerfile` no longer bakes `config/config.yaml.example` into the image; the Helm chart (`virtfoundry/helm-charts`) must mount or render the config (tracked separately).
  - MINOR bump for 0.x per [RELEASES.md](./RELEASES.md).

### Security

- **Restrict CORS and pin console WebSocket Origin** (issue [#98](https://github.com/virtfoundry/core/issues/98))
  - API CORS no longer emits `Access-Control-Allow-Origin: *`. Responses reflect only the request host (same-origin UI proxy) or an origin listed in `security.allowed_origins` / `VIRTFOUNDRY_ALLOWED_ORIGINS`. Cross-origin preflight from a disallowed origin is `403`.
  - `/ws/console` overrides the KubeVirt upgrader's allow-all `CheckOrigin` with the same allowlist already used by `/ws/events` (ticket auth from [#94](https://github.com/virtfoundry/core/issues/94) / [#111](https://github.com/virtfoundry/core/pull/111) made this safe).
  - UI nginx (`docker/nginx-ui.conf`) adds `Content-Security-Policy: frame-ancestors 'self'`, `X-Content-Type-Options: nosniff`, and `Referrer-Policy: strict-origin-when-cross-origin` without changing `/api/` or `/ws/` proxy behaviour.

- **Allowlist ISO HTTP import URLs** (issue [#95](https://github.com/virtfoundry/core/issues/95))
  - A tenant-supplied ISO URL used to become `spec.source.http.url` verbatim, so `vms:write` was enough to make the in-cluster CDI importer fetch cloud metadata (`169.254.169.254`), in-cluster services (`https://kubernetes.default.svc`), the node, or any RFC1918 host.
  - ISO URLs now require `https` on port 443, must not embed credentials, and may not target loopback, link-local, private/shared/reserved ranges (including their NAT64 re-encoding) or internal names (`*.svc`, `*.local`, `*.internal`, `*.localdomain`, `*.home.arpa`, single labels). Rejected URLs answer `400` and nothing is persisted.
  - The host must also be on an allowlist: `security.iso_import.allowed_hosts` (or `VIRTFOUNDRY_ISO_ALLOWED_HOSTS`), defaulting to the public ISO mirrors and object storage endpoints documented in [docs/VM-TEMPLATES.md](docs/VM-TEMPLATES.md#allowed-iso-urls). `security.iso_import.disable_http_import` (or `VIRTFOUNDRY_ISO_DISABLE_HTTP_IMPORT=1`) refuses URL imports entirely; `iso_volume_id` still works.
  - Defense in depth: `CreateHTTPImportDataVolume` re-checks the target, so no caller can reach CDI with a private or in-cluster URL.

### Changed

- `cmd/server` exits with a clear error on YAML load failure (was: silent fallback to defaults).
- `config/config.yaml.example` documents the env requirement instead of carrying a placeholder `jwt_secret`.
- `config/README.md` local-dev example uses `openssl rand -base64` to mint `JWT_SECRET` / `ROOT_PASSWORD`.

## [0.7.1] - 2026-09-04

### Fixed

- VM create image list hides the platform Windows ISO until the tenant uploads an ISO (CDI import ready)

### Changed

- Full Apache-2.0 `LICENSE` text in the repository
- UI and docs advertise `0.7.1`

### Added

- Draft [`docs/CNCF-SANDBOX-APPLICATION.md`](docs/CNCF-SANDBOX-APPLICATION.md)

## [0.7.0] - 2026-09-02

First tagged release of the CRD-store / operator line (no separate `v0.6.0` tag was ever published).

### Added

- Kubernetes `Repository` store (`virtfoundry.io` CRDs) as the only persistence backend
- [virtfoundry/operator](https://github.com/virtfoundry/operator): `v1alpha1` CRDs, Tenant namespace reconciler, Instance status sync from KubeVirt
- VM list performance improvements (CR fast path, reduced writes on list)

### Removed

- MySQL store (`mysql.go`, migrations, `go-sql-driver/mysql`)
- `cmd/worker` and `cmd/migrate` (CloudStack MySQL import)
- Worker binary from API container image

### Changed

- **Breaking (0.x):** MySQL store, embedded MySQL chart resources, and `cmd/worker` removed
- Store backends: **kubernetes** (production) or **memory** (local dev/tests) only
- Helm chart defaults to CRD store; install **virtfoundry-operator** before API/UI
- Docs: [Platform prerequisites](https://virtfoundry.github.io/helm-charts/docs/guide/prerequisites/) with KubeVirt, Multus, CDI, Longhorn, MetalLB links
- UI and docs advertise `0.7.0`

### Fixed

- Console auth tenant isolation via `resolveVMAccess`
- Homelab validated: API + operator + UI without MySQL/worker pods

## [0.5.0] - 2026-08-16

Pre-1.0 release line (project is **not** SemVer 1.0). Premature `v1.0.0`–`v1.5.0` tags/releases were deleted in the 2026-09-27 version cleanup; their shipped features are recorded here under `0.5.0`. UI/Helm advertise `0.5.0`.

### Added (absorbed from deleted premature 1.x line)

- IAM: users, roles, API keys (`vfd_live_...`), permission middleware, tenant admin bootstrap, `/iam` UI
- Default VPC per tenant; self-service API keys; volumes attach/detach; service offerings CRUD; template catalog + ISO import; root delete-tenant; dedicated CPU offerings; Features guide
- UI: Redux client state, accordion nav, login redesign, version from `package.json`

### Fixed (absorbed)

- Volume delete 409 when attached; VM pod NIC naming; CPU overcommit scheduling; IFNAMSIZ-safe public bridge default; CI attestation fix for containerd 2.x

## [0.2.0] - 2026-08-02

### Added

- VM templates: CRUD API, per-tenant defaults, UI Templates page
- ISO / CDI deploy path for Windows (boot PVC + install ISO)
- Public IP deploy: pool allocation, cloud-init static addressing by MAC
- Security groups: multi-rule editor, default tenant SG bootstrap
- SSH key injection on deploy; public IP requires SG + optional key

### Changed

- Public Multus NAD: macvlan → bridge CNI without IPAM (guest-routable IPs)
- `applyVMInfo`: prefer public NIC IP over pod masquerade address
- VM deploy: dual network (pod + public) when `allowPodNetwork` enabled

### Fixed

- Public VM SSH: cloud-init network-data with correct gateway and IP pool sync
- IP release on VM delete (`ReleaseIPAddressByAddress`)
- UI/API alignment for public network, security groups, and template deploy

## [0.1.0] - 2026-08-01

### Added

- Initial open-source release: multi-tenant IaaS API, worker, React UI
- KubeVirt VM lifecycle, Multus networking, NetworkPolicy security groups
- MySQL persistence, JWT auth, Gateway-compatible deployment

[0.7.3]: https://github.com/virtfoundry/core/compare/v0.7.2...v0.7.3
[0.7.2]: https://github.com/virtfoundry/core/compare/v0.7.1...v0.7.2
[0.7.1]: https://github.com/virtfoundry/core/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/virtfoundry/core/compare/v0.5.0...v0.7.0
[0.5.0]: https://github.com/virtfoundry/core/compare/v0.2.0...v0.5.0
[0.2.0]: https://github.com/virtfoundry/core/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/virtfoundry/core/releases/tag/v0.1.0
