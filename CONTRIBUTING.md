# Contributing to VirtFoundry

Thank you for helping grow VirtFoundry. This project lives under the [virtfoundry](https://github.com/virtfoundry) organization.

## Before you start

- Read the [Wiki Home](https://github.com/virtfoundry/core/wiki) for architecture and deployment setup
- Search [existing issues](https://github.com/virtfoundry/core/issues) before opening a duplicate

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

Cluster deploy and testing: [helm-charts](https://github.com/virtfoundry/helm-charts) (`helm install` or `make lint`).

## Branch workflow

**Do not commit directly to `main`.** Every feature or fix uses its own branch:

1. Branch from `main`: `feat/<name>`, `fix/<name>`, or `chore/<name>`
2. Implement + local tests (`go test ./...`, UI build if touched)
3. Deploy and validate on a **Kubernetes cluster** before opening a PR when behavior changes
4. Open PR → maintainer reviews and tests on a cluster → **merge only after approval**
5. After merge: **delete the feature branch** (remote + local). Org repos keep GitHub “Automatically delete head branches” enabled

Cross-repo changes: use the same branch name in `virtfoundry` and `helm-charts` when both are needed.

## Pull request process

1. Fork [virtfoundry/core](https://github.com/virtfoundry/core) (or the relevant repo) and create a feature branch
2. Keep changes focused; match existing code style
3. Run `go test ./...` and UI build when touching those areas (see [docs/CI.md](docs/CI.md))
4. Update docs when behavior or deploy steps change
5. Open a PR with a clear summary and test plan — required CI checks must be green before merge

## Repositories

| Repo | When to contribute here |
|------|-------------------------|
| [core](https://github.com/virtfoundry/core) | API, UI, local dev config |
| [operator](https://github.com/virtfoundry/operator) | CRDs and controllers |
| [helm-charts](https://github.com/virtfoundry/helm-charts) | Helm charts, docs site, deploy scripts |
| [terraform-provider-virtfoundry](https://github.com/virtfoundry/terraform-provider-virtfoundry) | Terraform provider |

Roles: [CONTRIBUTOR_LADDER.md](CONTRIBUTOR_LADDER.md) · Releases: [RELEASES.md](RELEASES.md)

## Code of conduct

This project follows the [CNCF Code of Conduct](CODE_OF_CONDUCT.md). Be respectful and constructive.

## Security

Report vulnerabilities privately — see [SECURITY.md](SECURITY.md).

## Questions

Use [GitHub Discussions](https://github.com/virtfoundry/core/discussions) for questions and ideas (prefer Issues for bugs and concrete features).
