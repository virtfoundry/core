# VirtFoundry Project Governance

VirtFoundry uses a **maintainer council** model (see [CNCF project-template GOVERNANCE-maintainer](https://github.com/cncf/project-template/blob/main/GOVERNANCE-maintainer.md)).

The project builds a Kubernetes-native multi-tenant private cloud (API + UI + `virtfoundry.io` CRDs/operator) for teams leaving Proxmox or avoiding raw KubeVirt YAML.

- [Values](#values)
- [Maintainers](#maintainers)
- [Becoming a Maintainer](#becoming-a-maintainer)
- [Decision process](#decision-process)
- [Official repositories](#official-repositories)
- [CNCF resources](#cncf-resources)
- [Code of Conduct](#code-of-conduct)
- [Security Response Team](#security-response-team)
- [Voting](#voting)
- [Modifying this charter](#modifying-this-charter)

## Values

- **Openness** — Discussion and decisions happen in public Issues, PRs, and Discussions when possible.
- **Fairness** — Contributions are judged on merit.
- **Community over product or company** — Sustaining the community outranks any single employer’s goals. Contributors participate as individuals.
- **Vendor neutrality** — No single organization controls roadmap, maintainer selection, or releases.
- **Inclusivity** — Welcoming and respectful environment.
- **Participation** — Responsibility is earned through sustained contribution ([CONTRIBUTOR_LADDER.md](CONTRIBUTOR_LADDER.md)).

## Maintainers

Maintainers have write access to VirtFoundry repositories and may merge their own or others’ patches. The current list is in [MAINTAINERS.md](MAINTAINERS.md).

The collective Maintainers form the **Maintainer Council**, the governing body for the project.

Lead maintainer (**Matheus Thurler**, CI&T) retains final merge authority on `main` and org-wide settings while the council is small. As the council grows, decisions shift toward lazy consensus and recorded votes (see [Voting](#voting)).

## Becoming a Maintainer

Demonstrate:

- Commitment for **3+ months**: discussions, contributions, reviews
- Reviews of **5+** non-trivial PRs
- **5+** non-trivial PRs merged (code and/or docs)
- Ability to collaborate; understanding of CI, review, and release processes
- Understanding of the relevant codebase (API/UI, operator, charts, or provider)

Process: nomination → public Issue/PR → existing maintainers agree → update [MAINTAINERS.md](MAINTAINERS.md).

## Decision process

1. **Day-to-day** — PRs to `main`; at least one maintainer review; cluster validation for infra-sensitive changes.
2. **Releases** — SemVer; ship app, operator, and charts together when possible — see [RELEASES.md](RELEASES.md).
3. **Large / breaking changes** — GitHub Issue or Discussion before implementation.
4. **Roadmap** — [ROADMAP.md](ROADMAP.md); community input welcome; lead maintainer / council decides.
5. **CNCF** — [docs/CNCF-CHECKLIST.md](docs/CNCF-CHECKLIST.md) tracks Sandbox readiness; no CNCF membership claims until accepted.

## Official repositories

| Repository | Role |
|------------|------|
| [core](https://github.com/virtfoundry/core) | REST API, UI, adoption layer over CRDs |
| [operator](https://github.com/virtfoundry/operator) | `virtfoundry.io` CRDs and Kubernetes operator |
| [helm-charts](https://github.com/virtfoundry/helm-charts) | Helm charts, docs site, deploy scripts |
| [terraform-provider-virtfoundry](https://github.com/virtfoundry/terraform-provider-virtfoundry) | Terraform provider |

Extended narrative: [helm-charts docs — Governance](https://virtfoundry.github.io/helm-charts/docs/project/governance/).

## CNCF resources

VirtFoundry aims for CNCF Sandbox readiness. Until accepted, do not represent the project as a CNCF project. Onboarding expectations follow [cncf/sandbox](https://github.com/cncf/sandbox) and the [project template](https://github.com/cncf/project-template).

## Code of Conduct

[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) — we follow the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md). Enforcement: project maintainers and/or [conduct@cncf.io](mailto:conduct@cncf.io).

## Security Response Team

Primary security contact: lead maintainer (see [SECURITY.md](SECURITY.md) and [MAINTAINERS.md](MAINTAINERS.md)). Private GitHub Security Advisories only — no public issues for vulnerabilities.

## Voting

Default: **lazy consensus** on PRs and Issues.

When a maintainer calls for a vote (breaking change, contentious roadmap, governance change):

- One maintainer, one vote (individuals, not companies)
- Simple majority of active maintainers
- Record the outcome in the Issue/PR

## Modifying this charter

Changes to this document require a PR reviewed by at least one other maintainer (or lead confirmation while the council has two maintainers) and should be announced in Discussions when material.

## Contributions

See [CONTRIBUTING.md](CONTRIBUTING.md). Contributions are Apache-2.0. **No CLA** today (DCO via GitHub may be enabled later).

## Contact

- Issues: https://github.com/virtfoundry/core/issues
- Discussions: https://github.com/virtfoundry/core/discussions
- Project board: https://github.com/orgs/virtfoundry/projects/1
