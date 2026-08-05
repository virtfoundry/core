# Changelog

All notable changes to **VirtFoundry** (API, worker, UI) are documented here.

Format based on [Keep a Changelog](https://keepachangelog.com/). Versioning: [SemVer](https://github.com/virtfoundry/helm-charts/blob/main/docs/project/versioning.md).

## [1.2.0] - 2026-08-05

### Added

- Volume attach and detach API (`POST/GET/DELETE /vms/{name}/volumes`) with KubeVirt hot-plug
- Volume delete endpoint (`DELETE /volumes/{id}`) with guard when still attached to a VM
- VM Detail **Storage** tab: list attached volumes, attach unattached volumes, detach
- Volumes page: show attached VM; deploy dropdown lists only unattached volumes
- `platform.storage.defaultClass` from Helm wired to tenant volume PVC creation
- `volume.vm_id` tracked on deploy, attach, and detach
- E2E API smoke scripts under `scripts/e2e/` (phases 1–3)

## [1.1.1] - 2026-08-04

### Fixed

- VM create with pod network: pod NIC renamed from `default` to `pod` to avoid KubeVirt duplicate network name conflict with the default VPC subnet
- Login page: full-width navbar, theme-aware light/dark hero panels, logo rendering, and pointer-event handling on desktop
- VirtFoundry logo PNG assets optimized (~3 MB → ~65 KB)

### Changed

- Login layout: split hero and sign-in panel with edge-to-edge header

## [1.1.0] - 2026-08-04

### Added

- Default VPC per tenant (`10.0.0.0/16` + `default` subnet) on bootstrap; VMs without explicit subnet use it automatically
- Public VM deploy attaches private (default VPC) + public NIC for bastion-style access on the same tenant network
- Self-service API keys: any authenticated user can create and revoke their own keys (no `users:write` required)
- Redux Toolkit for UI client state (auth, theme, sidebar, tenant selection)
- Sidebar accordion navigation (Compute, Storage, Network, Platform)
- Header user menu and settings popover (theme, language, docs, about)
- Bundled JetBrains Mono fonts; `favicon.svg` (V monogram)
- Targeted React Query invalidation from WebSocket events (reduced full-page refresh noise)

### Changed

- VM deploy UI: removed SSH NodePort exposure; access via public-network SSH or noVNC console
- IAM UI: API Keys tab for all users; Users/Roles tabs limited to tenant admins
- Dashboard: removed compute allocation chart; centered resource stat cards
- VirtFoundry logo: light/dark PNG stack for instant theme swap
- Header layout: tenant selector, notifications, settings, and user avatar grouped on the right

### Fixed

- Circular import between Redux store and `platform-api` causing blank UI on load
- Dark mode sidebar lag (scoped CSS transitions off theme-sensitive surfaces)
- Background polling no longer triggers visible full-page refresh overlays

## [1.0.0] - 2026-08-03

### Added

- IAM: users, roles, API keys (`vfd_live_...`), permission middleware
- Tenant admin bootstrap, `/iam` UI for users and roles

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

[1.2.0]: https://github.com/virtfoundry/core/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/virtfoundry/core/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/virtfoundry/core/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/virtfoundry/core/compare/v0.2.0...v1.0.0
[0.2.0]: https://github.com/virtfoundry/core/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/virtfoundry/core/releases/tag/v0.1.0
