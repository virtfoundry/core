# VirtFoundry — traction & CNCF checklist

Living checklist to grow adoption fast and stay Sandbox-ready.  
**Canonical repo:** [virtfoundry/core](https://github.com/virtfoundry/core). Chart/docs: [helm-charts](https://github.com/virtfoundry/helm-charts).

**Positioning:** private cloud multi-tenant on Kubernetes for people leaving **Proxmox** (or avoiding “raw KubeVirt”) who want tenant / VPC / VM / volume / IAM via API + UI.

---

## Phase 0 — House in order (weeks 1–4) — START HERE

Goal: project looks intentional to strangers in under 5 minutes.

| # | Item | Owner | Status |
|---|------|-------|--------|
| 0.1 | Apache-2.0 `LICENSE` + `NOTICE` on all official repos | — | ✅ full Apache-2.0 text (GitHub-detectable) + NOTICE |
| 0.2 | `GOVERNANCE.md` + `MAINTAINERS.md` (root; Company column) | — | ✅ ([project-template](https://github.com/cncf/project-template) shape) |
| 0.3 | `CONTRIBUTING.md` + Conventional Commits | — | ✅ |
| 0.4 | `CODE_OF_CONDUCT.md` (CNCF CoC) on all repos | | ✅ |
| 0.5 | `SECURITY.md` on all repos | | ✅ |
| 0.5b | `RELEASES.md` + `CONTRIBUTOR_LADDER.md` + `CODEOWNERS` | | ✅ (recommended template files) |
| 0.6 | `ROADMAP.md` (public milestones) | | ✅ |
| 0.7 | `ADOPTERS.md` (template + first entry) | | ✅ |
| 0.8 | [Why VirtFoundry](WHY.md) (vs Proxmox / KubeVirt / Harvester) | | ✅ |
| 0.9 | README badges + links to Why / Roadmap / Security | | ✅ |
| 0.10 | Quickstart path under 30 min documented (kind or homelab) | | ✅ ([guide](https://virtfoundry.github.io/helm-charts/docs/guide/quickstart/)) |
| 0.11 | SemVer releases + CHANGELOG kept current | — | ✅ (keep discipline) |
| 0.12 | Enable GitHub Discussions on `core` | | ✅ |
| 0.13 | 5+ issues labeled `good first issue` | | ✅ (#27–#31) |

**Exit criteria:** cold visitor understands what it is, how to install, how to contribute, and how VirtFoundry differs from Proxmox.

---

## Phase 1 — Traction + Sandbox-shaped (months 1–3)

Goal: demos, adopters, discoverability — *not* the CNCF application yet.

| # | Item | Status |
|---|------|--------|
| 1.1 | 10-min demo video (VM + volume + snapshot + UI) | ⬜ |
| 1.2 | Blog/post: “Leaving Proxmox for K8s-native private cloud” | ⬜ |
| 1.3 | Comparison page kept honest (what Proxmox still wins) | ✅ (WHY.md “When Proxmox still wins”) |
| 1.4 | 2–3 adopters listed (homelab OK; company optional) | ✅ (Matheus + Rodrigo + Weslei; see [ADOPTERS.md](../ADOPTERS.md)) |
| 1.5 | Slack or Discord + link from README | ⬜ |
| 1.6 | Talk/meetup (KubeVirt / CNCF BR / local) | ⬜ |
| 1.7 | CI green on PR (Go test + UI build + helm lint) | ✅ ([docs/CI.md](CI.md); enforce ruleset on `main`) |
| 1.8 | `virtfoundry.dev` or GitHub Pages as single front door | ✅ (Pages canonical; see website.md) |
| 1.9 | TAG Runtime / KubeVirt community intro (async) | ⬜ |

**Exit criteria:** someone outside the maintainer can install from docs and open a useful PR.

---

## Phase 1.5 — Supply chain and repo hygiene

Reviewers look at how the project is built and shipped. Tracked in [#200](https://github.com/virtfoundry/core/issues/200).

| # | Item | Status |
|---|------|--------|
| 1.5.1 | Dependabot (grouped, monthly), CodeQL, OpenSSF Scorecard and dependency review (AGPL/GPL/BUSL deny list) on every official repo | ✅ merged in all six repos; `dependency-review` is non-blocking until 1.5.9 |
| 1.5.2 | SPDX SBOM attached to each release | ✅ merged (core, operator, helm-charts, terraform-provider); attaches from the next release |
| 1.5.3 | Community health files and full Apache-2.0 `LICENSE` on `vks` and `vks-image-factory` | ✅ merged |
| 1.5.4 | `protect-main` ruleset on every official repo, with force-push and deletion protection | ✅ all six repos |
| 1.5.5 | Secret scanning, push protection, private vulnerability reporting, Dependabot security updates | ✅ all six repos |
| 1.5.6 | Signed images and release artifacts (cosign keyless; terraform releases are GPG-signed) | 🟡 operator already signs; core (API, UI) in core#221, vks in vks#21 |
| 1.5.7 | RBAC contract check: the API's typed Kubernetes calls vs the `virtfoundry-api` roles | 🟡 core side merged (core#220); chart side in helm-charts#100 |
| 1.5.8 | Resolve the stale security PRs | ✅ core#103 was already on `main` (closed as superseded); terraform-provider#15 landed as #40 with authorship preserved |
| 1.5.9 | Enable the dependency graph in each repository's settings, then drop `continue-on-error` from `dependency-review` | ⬜ |
---

## Phase 2 — CNCF Sandbox application prep (when Phase 1 exits)

| # | Item | Status |
|---|------|--------|
| 2.1 | Sandbox proposal draft (problem, differentiation, alignment) | ✅ [CNCF-SANDBOX-APPLICATION.md](CNCF-SANDBOX-APPLICATION.md) |
| 2.1b | Repo age (soft signal): the cncf/sandbox README states no minimum, but reviewers have flagged projects under six months | 🟡 (`core` since 2026-08-03; soft target ~2027-02) |
| 2.2 | Adopters statement + logos (if any) | 🟡 (statement in ADOPTERS.md; need non-maintainer adopters) |
| 2.3 | Multiple contributors with merged PRs | 🟡 (Rodrigo + Weslei maintainers; grow more external contributors) |
| 2.3b | Maintainer spread (the README says employer diversity is considered, not required) | 🟡 3 maintainers, 2 employers (Matheus + Weslei @ CI&T; Rodrigo @ SYS MANAGER); see employer-consent item in #200 |
| 2.4 | Security contact + advisory process practiced once | ⬜ |
| 2.5 | Signatory + contact emails filled for Contribution Agreement | 🟡 (placeholders in draft) |
| 2.6 | Optional Day 0 GTR / TAG Runtime intro | ⬜ |
| 2.7 | Submit via [cncf/sandbox](https://github.com/cncf/sandbox) issue form | ⬜ **blocked by Phase 1 exit and [#200](https://github.com/virtfoundry/core/issues/200)**, not by repo age alone |

**Do not apply early** without 0+1 exit criteria and the open items in [#200](https://github.com/virtfoundry/core/issues/200).

How review works (cncf/sandbox README, checked 2026-10-05): the TOC reviews applications about every two months, first in first out. Outcomes are Approved, Declined, Need-Info, Postponed and Returning. A Postponed result costs a cycle, so apply when ready rather than as early as possible.

### Known risks, stated plainly

- **Positioning.** The README says operators that enable another OSS project should join it as a subproject, and that only reusable projects (not reference architectures) are accepted. The answer to "why not a KubeVirt subproject" carries the application. `vks` (overlap with Kamaji and Cluster API) is experimental and kept out of the application scope.
- **Concentration.** About 93% of commits in `core` come from one person, and every listed adopter is a maintainer's homelab.
- **Employer diversity.** It is considered, not required, and GitHub org membership does not count. Two of the three maintainers share an employer. Confirm no employer claims rights, because the Contribution Agreement transfers trademark and domain.
- **Governance.** The lead maintainer's final merge authority sits awkwardly next to the vendor-neutrality claim; GOVERNANCE.md already says it moves toward council decisions as the council grows.

---

## Phase 3 — After Sandbox (later)

| # | Item | Status |
|---|------|--------|
| 3.1 | Grow maintainers beyond BDFL | ✅ (lead + Weslei + Rodrigo; see MAINTAINERS.md) |
| 3.2 | Multi-cluster CI (kind + bare metal or cloud) | ⬜ |
| 3.3 | Incubation criteria tracking | ⬜ |

---

## This week (execution order)

1. ~~Land Phase 0 docs~~  
2. ~~Open GitHub milestones + Project + checklist issues~~  
3. ~~Enable Discussions (#25) + seed good first issues (#26)~~  
4. ~~Document **quickstart under 30 min** (#24)~~  
5. Record or script a **demo** ([#32](https://github.com/virtfoundry/core/issues/32))  
6. Ask 2 friends/homelabs to try install and file issues (non-maintainer ADOPTERS)  
7. ~~Homelab E2E suite green (CR store)~~ — done 2026-09  
8. ~~Full Apache-2.0 LICENSE text + Sandbox application draft~~  
9. Work the open items in [#200](https://github.com/virtfoundry/core/issues/200); open the cncf/sandbox issue when Phase 1 exits and those are done (soft target ~2027-02, no hard minimum age)  

## Related docs

- [GOVERNANCE.md (core)](https://github.com/virtfoundry/core/blob/main/GOVERNANCE.md) — how decisions are made  
- [CNCF-CHECKLIST.md](docs/CNCF-CHECKLIST.md) — traction & Sandbox readiness  
- [Installation](https://virtfoundry.github.io/helm-charts/docs/guide/installation/)  
- [Topologies](https://virtfoundry.github.io/helm-charts/docs/guide/topologies/)  

## CNCF open-source alignment (2026)

VirtFoundry follows common [CNCF](https://www.cncf.io/) project conventions:

| Requirement | Location |
|-------------|----------|
| Apache-2.0 + NOTICE | All official repos (full LICENSE text; see [Sandbox draft](CNCF-SANDBOX-APPLICATION.md)) |
| GOVERNANCE + MAINTAINERS | [GOVERNANCE.md](../GOVERNANCE.md), [MAINTAINERS.md](../MAINTAINERS.md) |
| CONTRIBUTING + Conventional Commits | [CONTRIBUTING.md](../CONTRIBUTING.md) |
| CODE_OF_CONDUCT (Contributor Covenant 2.1) | All repos |
| SECURITY.md + private advisory | [SECURITY.md](../SECURITY.md) |
| ADOPTERS + ROADMAP | [ADOPTERS.md](../ADOPTERS.md), [ROADMAP.md](../ROADMAP.md) |
| Operator / CRD-first control plane | [operator](https://github.com/virtfoundry/operator), [CRD design spec](docs/superpowers/specs/2026-09-01-crd-operator-design.md) |
| Platform store | Kubernetes CRDs (`store.driver=kubernetes`) only — no MySQL/Vitess/worker |
