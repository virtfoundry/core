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
make help          # list all targets (UI-focused)
make test          # run UI tests (vitest)
make build         # build UI production bundle
```

Direct commands (if you prefer not to use the Makefile):

```bash
go build ./...
go test ./...
cd ui && npm ci && npm test && npm run build
```

The root `Makefile` covers the UI workflow (`make test`, `make build`,
`make ci`, `make clean`). Backend commands (`go build`, `go test`) are
run directly until a Go runner is added.

Cluster deploy and testing: [helm-charts](https://github.com/virtfoundry/helm-charts) (`helm install` or `make lint`).

## Branch workflow

**Do not commit directly to `main`.** Every feature or fix uses its own branch:

1. Branch from `main`: `feat/<name>`, `fix/<name>`, or `chore/<name>`
2. Implement + local tests (`make test`, `make build` if UI touched; `go test ./...` if backend touched)
3. Deploy and validate on a **Kubernetes cluster** before opening a PR when behavior changes
4. Open PR → maintainer reviews and tests on a cluster → **merge only after approval**
5. After merge: **delete the feature branch** (remote + local). Org repos keep GitHub “Automatically delete head branches” enabled

Cross-repo changes: use the same branch name in `virtfoundry` and `helm-charts` when both are needed.

## Pull request process

1. Fork [virtfoundry/core](https://github.com/virtfoundry/core) (or the relevant repo) and create a feature branch
2. Keep changes focused; match existing code style
3. Run `make test` / `make build` for UI changes and `go test ./...` for backend changes (see [docs/CI.md](docs/CI.md))
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

## Kubernetes API access (RBAC contract)

The API's access to the Kubernetes API is declared in [docs/rbac-contract.yaml](docs/rbac-contract.yaml). `go test ./internal/platform/k8s/` fails if the code makes a typed client-go call that is not listed there, or if an entry is no longer used. When you add such a call:

1. Add the `group/resource/verb` to the contract.
2. Grant it in the `virtfoundry-api` ClusterRole in [helm-charts](https://github.com/virtfoundry/helm-charts) (its `chart-lint` workflow checks the chart against this contract).
3. If the call tolerates `Forbidden` and you do not want to widen the role, list it under `bestEffortVerbs` instead.

Unit tests use fake clients and cannot see RBAC, so this is the only check before a deploy.

## Code of conduct

This project follows the [CNCF Code of Conduct](CODE_OF_CONDUCT.md). Be respectful and constructive.

## Security

Report vulnerabilities privately — see [SECURITY.md](SECURITY.md).

## Questions

Use [GitHub Discussions](https://github.com/virtfoundry/core/discussions) for questions and ideas (prefer Issues for bugs and concrete features).
