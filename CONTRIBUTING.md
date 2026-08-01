# Contributing to VirtForge Cloud

Thank you for helping grow VirtForge Cloud. This project lives under the [virtforge-cloud](https://github.com/virtforge-cloud) organization.

## Before you start

- Read the [Wiki Home](https://github.com/virtforge-cloud/virtforge/wiki) for architecture and homelab setup
- Search [existing issues](https://github.com/virtforge-cloud/virtforge/issues) before opening a duplicate

## Language

- **Commits**: English only, [Conventional Commits](https://www.conventionalcommits.org/)
- **Documentation** (README, wiki, code comments for contributors): English
- **Pull request descriptions**: English

### Commit examples

```
feat(compute): add VM resize endpoint
fix(ui): refresh tenant list after create
docs(wiki): document Multus bridge setup
chore(deploy): bump KubeVirt chart reference
```

## Development setup

```bash
cp config/config.yaml.example config/config.yaml   # optional
go build ./...
go test ./...
cd ui && npm install && npm run build
```

Cluster deploy and homelab testing: [virtforge-chart](https://github.com/virtforge-cloud/virtforge-chart) (`make deploy-homelab`).

## Branch workflow

**Do not commit directly to `main`.** Every feature or fix uses its own branch:

1. Branch from `main`: `feat/<name>`, `fix/<name>`, or `chore/<name>`
2. Implement + local tests (`go test ./...`, UI build if touched)
3. Deploy and validate on **homelab** before opening a PR
4. Open PR → maintainer reviews and tests on homelab → **merge only after approval**

Cross-repo changes: use the same branch name in `virtforge` and `virtforge-chart` when both are needed.

## Pull request process

1. Fork `virtforge-cloud/virtforge` and create a feature branch
2. Keep changes focused; match existing code style
3. Run `go test ./...` and UI build when touching those areas
4. Update wiki pages when behavior or deploy steps change
5. Open a PR with a clear summary and test plan

## Repositories

| Repo | When to contribute here |
|------|-------------------------|
| [virtforge](https://github.com/virtforge-cloud/virtforge) | API, worker, UI, migration CLI, local dev config |
| [virtforge-chart](https://github.com/virtforge-cloud/virtforge-chart) | Helm chart, cluster values, deploy scripts |
| [virtforge-website](https://github.com/virtforge-cloud/virtforge-website) | Marketing site and published docs |

## Code of conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). Be respectful and constructive.

## Security

Report vulnerabilities privately — see [SECURITY.md](SECURITY.md).
