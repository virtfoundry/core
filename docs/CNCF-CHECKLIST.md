# VirtFoundry — traction & CNCF checklist

Living checklist to grow adoption fast and stay Sandbox-ready.  
**Canonical repo:** [virtfoundry/core](https://github.com/virtfoundry/core). Chart/docs: [helm-charts](https://github.com/virtfoundry/helm-charts).

**Positioning:** private cloud multi-tenant on Kubernetes for people leaving **Proxmox** (or avoiding “raw KubeVirt”) who want tenant / VPC / VM / volume / IAM via API + UI.

---

## Phase 0 — House in order (weeks 1–4) — START HERE

Goal: project looks intentional to strangers in under 5 minutes.

| # | Item | Owner | Status |
|---|------|-------|--------|
| 0.1 | Apache-2.0 `LICENSE` + `NOTICE` on all official repos | — | ✅ core / helm / terraform / operator |
| 0.2 | `GOVERNANCE.md` + `MAINTAINERS.md` (root) | — | ✅ |
| 0.3 | `CONTRIBUTING.md` + Conventional Commits | — | ✅ |
| 0.4 | `CODE_OF_CONDUCT.md` on all repos | | ✅ |
| 0.5 | `SECURITY.md` on all repos | | ✅ |
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
| 1.4 | 2–3 adopters listed (homelab OK; company optional) | ✅ (Matheus + Rodrigo; see [ADOPTERS.md](../ADOPTERS.md)) |
| 1.5 | Slack or Discord + link from README | ⬜ |
| 1.6 | Talk/meetup (KubeVirt / CNCF BR / local) | ⬜ |
| 1.7 | CI green on PR (Go test + UI build + helm lint) | ✅ ([docs/CI.md](CI.md); enforce ruleset on `main`) |
| 1.8 | `virtfoundry.dev` or GitHub Pages as single front door | ✅ (Pages canonical; see website.md) |
| 1.9 | TAG Runtime / KubeVirt community intro (async) | ⬜ |

**Exit criteria:** someone outside the maintainer can install from docs and open a useful PR.

---

## Phase 2 — CNCF Sandbox application prep (when Phase 1 exits)

| # | Item | Status |
|---|------|--------|
| 2.1 | Sandbox proposal draft (problem, differentiation, alignment) | ⬜ |
| 2.2 | Adopters statement + logos (if any) | 🟡 (statement in ADOPTERS.md; logos when available) |
| 2.3 | Multiple contributors with merged PRs | ⬜ (Rodrigo Gonçalves — maintainer; grow external contributors) |
| 2.4 | Security contact + advisory process practiced once | ⬜ |
| 2.5 | Submit via CNCF TOC process | ⬜ |

**Do not apply early** without 0+1 exit criteria.

---

## Phase 3 — After Sandbox (later)

| # | Item | Status |
|---|------|--------|
| 3.1 | Grow maintainers beyond BDFL | 🟡 (lead + Rodrigo; document in MAINTAINERS.md) |
| 3.2 | Multi-cluster CI (kind + bare metal or cloud) | ⬜ |
| 3.3 | Incubation criteria tracking | ⬜ |

---

## This week (execution order)

1. ~~Land Phase 0 docs~~  
2. ~~Open GitHub milestones + Project + checklist issues~~  
3. ~~Enable Discussions (#25) + seed good first issues (#26)~~  
4. ~~Document **quickstart under 30 min** (#24)~~  
5. Record or script a **demo** ([#32](https://github.com/virtfoundry/core/issues/32))  
6. Ask 2 friends/homelabs to try install and file issues  
7. ~~Homelab E2E suite green (CR store)~~ — done 2026-09  

## Related docs

- [GOVERNANCE.md (core)](https://github.com/virtfoundry/core/blob/main/GOVERNANCE.md) — how decisions are made  
- [CNCF-CHECKLIST.md](docs/CNCF-CHECKLIST.md) — traction & Sandbox readiness  
- [Installation](https://virtfoundry.github.io/helm-charts/docs/guide/installation/)  
- [Topologies](https://virtfoundry.github.io/helm-charts/docs/guide/topologies/)  

## CNCF open-source alignment (2026)

VirtFoundry follows common [CNCF](https://www.cncf.io/) project conventions:

| Requirement | Location |
|-------------|----------|
| Apache-2.0 + NOTICE | All official repos |
| GOVERNANCE + MAINTAINERS | [GOVERNANCE.md](../GOVERNANCE.md), [MAINTAINERS.md](../MAINTAINERS.md) |
| CONTRIBUTING + Conventional Commits | [CONTRIBUTING.md](../CONTRIBUTING.md) |
| CODE_OF_CONDUCT (Contributor Covenant 2.1) | All repos |
| SECURITY.md + private advisory | [SECURITY.md](../SECURITY.md) |
| ADOPTERS + ROADMAP | [ADOPTERS.md](../ADOPTERS.md), [ROADMAP.md](../ROADMAP.md) |
| Operator / CRD-first control plane | [operator](https://github.com/virtfoundry/operator), [CRD design spec](docs/superpowers/specs/2026-09-01-crd-operator-design.md) |
| Platform store | Kubernetes CRDs (`store.driver=kubernetes`) only — no MySQL/Vitess/worker |
