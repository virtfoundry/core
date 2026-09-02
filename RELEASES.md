# Release Process

VirtFoundry follows [Semantic Versioning](https://semver.org/). Cross-repo versioning rules: [helm-charts versioning](https://github.com/virtfoundry/helm-charts/blob/main/docs/project/versioning.md).

## Release cadence

**Feature-based** while on **0.x**. Cut a release when a coherent set of changes is ready (store/API, operator, charts, docs). Prefer shipping **core + helm-charts + operator** on the same `vX.Y.Z`.

## Versioning (0.x)

| Change | Bump | Example |
|--------|------|---------|
| Bug / doc fix | PATCH | `0.7.0` → `0.7.1` |
| Feature or chart profile change | MINOR | `0.7.0` → `0.8.0` |
| Breaking API/chart (still 0.x) | MINOR + CHANGELOG note | `0.7.0` → `0.8.0` |
| First stable contract | MAJOR | `0.x` → `1.0.0` |

## Artifacts

Each `vX.Y.Z` release should publish:

| Artifact | Source | Tag / name |
|----------|--------|------------|
| API image | `virtfoundry/core` CI | `ghcr.io/virtfoundry/core:X.Y.Z` |
| UI image | `virtfoundry/core` CI | `ghcr.io/virtfoundry/ui:X.Y.Z` |
| Operator image | `virtfoundry/operator` CI | `ghcr.io/virtfoundry/operator:X.Y.Z` |
| Helm charts | `virtfoundry/helm-charts` chart-releaser | `virtfoundry` + `virtfoundry-operator` `X.Y.Z` |
| GitHub Releases | all three repos | tag `vX.Y.Z` |

## Checklist (every release)

1. Update `CHANGELOG.md` (core, helm-charts, operator as needed)
2. Bump **every** pin to the same `X.Y.Z`:
   - core: `ui/package.json`, `ui/package-lock.json`, `docs/PRODUCT.md`
   - helm-charts: both `Chart.yaml`, `values.yaml` image tags, install docs (`quickstart`, `installation`, `kind`, `index`, README)
   - operator: chart `Chart.yaml` + `values.yaml`
3. Merge to `main`
4. Tag and push `vX.Y.Z` on **core**, **helm-charts**, and **operator**
5. Confirm CI: GHCR SemVer tags + Helm index lists `X.Y.Z`
6. Create/refresh GitHub Releases notes

## Supported versions

| Line | Supported |
|------|-----------|
| Latest `v0.Y.Z` tag | yes |
| `main` | yes (pre-release) |
| Older 0.x minors | best-effort; security fixes prefer latest |

## Homelab / GitOps

Homelab Argo overlays pin **image digests** (not floating tags). CI on `core` `main`/tags writes digests into `argo-homelab` when configured.
