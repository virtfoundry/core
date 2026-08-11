# Governance

VirtFoundry is an open-source project under the [Apache License 2.0](LICENSE).

## Project lead

The [virtfoundry](https://github.com/virtfoundry) GitHub organization maintains the official repositories:

| Repository | Role |
|------------|------|
| [core](https://github.com/virtfoundry/core) | API, worker, UI |
| [helm-charts](https://github.com/virtfoundry/helm-charts) | Helm chart, deploy tooling, docs site |
| [terraform-provider-virtfoundry](https://github.com/virtfoundry/terraform-provider-virtfoundry) | Terraform provider |

| Role | Responsibility |
|------|----------------|
| **Maintainer (BDFL)** | Merges, releases, roadmap, org settings |
| **Contributors** | Changes via pull requests; no automatic governance rights |

Extended narrative (sponsorship, trademarks): [helm-charts docs — Governance](https://virtfoundry.github.io/helm-charts/docs/project/governance/).

## Decision process

1. **Day-to-day** — PRs to `main`, maintainer review; cluster validation for infra-sensitive changes  
2. **Releases** — SemVer; keep app + chart versions aligned when shipping together  
3. **Large / breaking changes** — GitHub Issue or Discussion before implementation  
4. **Roadmap** — [ROADMAP.md](ROADMAP.md); community input welcome, maintainer decides  

## Contributions

See [CONTRIBUTING.md](CONTRIBUTING.md). Contributions are Apache-2.0. **No CLA** today.

## Code of conduct & security

- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)  
- [SECURITY.md](SECURITY.md) — private vulnerability reports only  

## Contact

- Issues: https://github.com/virtfoundry/core/issues  
- Discussions: https://github.com/virtfoundry/core/discussions (when enabled)  
