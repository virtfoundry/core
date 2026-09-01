# Governance

VirtFoundry is an open-source project under the [Apache License 2.0](LICENSE), aligned with [CNCF](https://www.cncf.io/) open-source project practices (public governance, maintainers file, security disclosure, Contributor Covenant).

## Maintainers

See [MAINTAINERS.md](MAINTAINERS.md) for the canonical list.

| Name | GitHub | Role |
|------|--------|------|
| Matheus Thurler | [@Matheus-Thurler](https://github.com/Matheus-Thurler) | Lead maintainer — releases, roadmap, org settings, security contact |
| Rodrigo Gonçalves | [@RodrigoGoncalves-dev](https://github.com/RodrigoGoncalves-dev) | Maintainer — review, homelab validation, operator/infra work |

Matheus retains final merge authority on `main` and org-wide decisions. Rodrigo is a full maintainer for review, cluster testing, and day-to-day technical direction.

## Official repositories

| Repository | Role |
|------------|------|
| [core](https://github.com/virtfoundry/core) | REST API, UI, adoption layer over CRDs |
| [operator](https://github.com/virtfoundry/operator) | `virtfoundry.io` CRDs and Kubernetes operator |
| [helm-charts](https://github.com/virtfoundry/helm-charts) | Helm charts, docs site, deploy scripts |
| [terraform-provider-virtfoundry](https://github.com/virtfoundry/terraform-provider-virtfoundry) | Terraform provider |

Extended narrative (sponsorship, trademarks): [helm-charts docs — Governance](https://virtfoundry.github.io/helm-charts/docs/project/governance/).

## Decision process

1. **Day-to-day** — PRs to `main`; at least one maintainer review; cluster validation for infra-sensitive changes (Rodrigo or lead maintainer on homelab).
2. **Releases** — SemVer; keep app, operator, and chart versions documented when shipping together.
3. **Large / breaking changes** — GitHub Issue or Discussion before implementation.
4. **Roadmap** — [ROADMAP.md](ROADMAP.md); community input welcome; lead maintainer decides.
5. **CNCF** — [docs/CNCF-CHECKLIST.md](docs/CNCF-CHECKLIST.md) tracks Sandbox readiness; no CNCF membership claims until accepted.

## Contributions

See [CONTRIBUTING.md](CONTRIBUTING.md). Contributions are Apache-2.0. **No CLA** today.

## Code of conduct & security

- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) — enforced by maintainers listed above.
- [SECURITY.md](SECURITY.md) — private vulnerability reports only; lead maintainer is security contact.

## Contact

- Issues: https://github.com/virtfoundry/core/issues
- Discussions: https://github.com/virtfoundry/core/discussions
- Project board: https://github.com/orgs/virtfoundry/projects/1
