# Changelog

All notable changes to **VirtFoundry** (API, worker, UI) are documented here.

Format based on [Keep a Changelog](https://keepachangelog.com/). Versioning: [SemVer](https://github.com/virtfoundry/helm-charts/blob/main/docs/project/versioning.md).

## [Unreleased]

### Added

- Deploy flag `dedicated_cpu` for Guaranteed QoS (`requests.cpu = limits.cpu = cores`) when overcommit is not desired
- Service offering field `dedicated_cpu` plus seeded `small-dedicated` / `medium-dedicated` / `large-dedicated`
- UI deploy checkbox and Offerings admin toggle for dedicated CPU

### Fixed

- VM deploy CPU scheduling: set guest `domain.cpu.cores` and omit CPU request by default so KubeVirt `cpuAllocationRatio` can overcommit (fixes `Insufficient cpu` when VMs are idle)

## [1.4.1] - 2026-08-05

### Fixed

- Volume delete while attached to a VM returns **409 Conflict** instead of HTTP 500
- Volume delete for missing ID returns **404 Not Found** via typed API errors
- Login and settings UI show app version from `ui/package.json` at build time (was hardcoded v1.1.1)

## [1.4.0] - 2026-08-05

### Added

- `docs/VM-TEMPLATES.md`: container disks, ISO import (CDI), platform vs tenant scope, UI and API registration
- Templates UI polls every 5s during ISO import with spinner/failed badges on template cards

### Changed

- Template seed deduplication: platform catalog keeps global templates (cirros, ubuntu-2204, windows); tenant bootstrap only adds fedora-39 and skips names already in platform catalog
- Updated `docs/ARCHITECTURE.md`, `TODO.md`, and README docs section for current templates API and UI

## [1.3.0] - 2026-08-05

### Added

- Service offerings CRUD API for root users (`POST/PATCH/DELETE /service-offerings`) with validation and soft-delete (state → Inactive)
- Admin UI at `/offerings` to list, create, edit, and deactivate offerings
- VM resize persists `service_offering_id` via `PATCH /vms/{name}`; VM detail overview shows offering name

## [1.2.0] - 2026-08-05

### Added

- Volume attach and detach API (`POST/GET/DELETE /vms/{name}/volumes`) with KubeVirt hot-plug
- Volume delete endpoint (`DELETE /volumes/{id}`) with guard when still attached to a VM
- VM Detail **Storage** tab: list attached volumes, attach unattached volumes, detach
- Volumes page: show attached VM; deploy dropdown lists only unattached volumes
- `platform.storage.defaultClass` from Helm wired to tenant volume PVC creation
- `volume.vm_id` tracked on deploy, attach, and detach

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

[1.4.1]: https://github.com/virtfoundry/core/compare/v1.4.0...v1.4.1
[1.4.0]: https://github.com/virtfoundry/core/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/virtfoundry/core/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/virtfoundry/core/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/virtfoundry/core/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/virtfoundry/core/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/virtfoundry/core/compare/v0.2.0...v1.0.0
[0.2.0]: https://github.com/virtfoundry/core/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/virtfoundry/core/releases/tag/v0.1.0
