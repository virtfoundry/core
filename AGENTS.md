# VirtForge Cloud — Agent Guide

Cursor rules (optional, local): `.cursor/rules/`. This project is open source under [virtforge-cloud](https://github.com/virtforge-cloud).

## Rules

| File | Scope |
|------|-------|
| `virtforge-project.mdc` | Project context, stack, multi-tenant model, OSS conventions |
| `go-backend.mdc` | `**/*.go` — handler/service/store layers |
| `react-ui.mdc` | `ui/**/*.{tsx,ts}` — TanStack Query, query-keys, realtime |

## Language & documentation

- **All documentation** (README, wiki, architecture guides, comments meant for contributors): **English**
- **Git commits**: **English**, [Conventional Commits](https://www.conventionalcommits.org/) (`feat`, `fix`, `docs`, `chore`, etc.)
- **Extended docs**: prefer the [GitHub Wiki](https://github.com/virtforge-cloud/virtforge/wiki) for architecture, runbooks, and roadmap detail; keep README focused on quick start

## Useful commands

```bash
# Build Go
go build ./...

# Build UI
cd ui && npm run build

# Local API (optional config)
cp config/config.yaml.example config/config.yaml
ROOT_PASSWORD=virtforge go run ./cmd/server
```

Docs: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) · [`config/README.md`](config/README.md) · [Wiki](https://github.com/virtforge-cloud/virtforge/wiki)

## Preferences

- VNC console opens in a new tab (`/console?name=&namespace=`); Cirros uses VGA (~720×400); Ubuntu/Fedora use `video: virtio`

## Do not

- Couple handlers directly to `store.Memory` or MySQL — use `store.Repository`
- Hardcode query keys in the UI — use `lib/query-keys.ts`
- Add K8s manifests or Helm values here — they belong in [virtforge-chart](https://github.com/virtforge-cloud/virtforge-chart)
- Reference employer-specific projects, paths, or secrets in public docs or commits
