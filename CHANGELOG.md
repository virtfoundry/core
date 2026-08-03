# Changelog

All notable changes to **VirtFoundry** (API, worker, UI) are documented here.

Format based on [Keep a Changelog](https://keepachangelog.com/). Versioning: [SemVer](https://github.com/virtfoundry/helm-charts/blob/main/docs/project/versioning.md).

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

[0.2.0]: https://github.com/virtfoundry/core/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/virtfoundry/core/releases/tag/v0.1.0
