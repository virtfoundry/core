# AGENTS — virtfoundry/core

API + UI do VirtFoundry (Go REST, React SPA). Deploy no homelab via Argo CD após merge em `main`.

## Cursor Team Kit

Usar o plugin **cursor-team-kit**:

| Situação | Skill |
|----------|--------|
| Branch + PR | `new-branch-and-pr` / `review-and-ship` |
| CI | `fix-ci` + `loop-on-ci` |
| PR legível | `make-pr-easy-to-review` |
| Typecheck | `check-compiler-errors` |
| Limpar noise de AI | `deslop` |

Rules: `typescript-exhaustive-switch`, `no-inline-imports`.

## VirtFoundry

- SemVer produto **0.9.x** (alinhar CHANGELOG / RELEASES).
- Testes no **homelab** — nunca Kind.
- Preview sem commit só com pedido explícito.
- Não taguear / mergear release sem OK do maintainer.
- **CRD-first (`operatorReconcile`):** Start/Stop → só `Instance.spec.powerState`. DeployVM → um actuator (`deployVMViaOperator` / SaveVM) — nunca `CreateVM`+`SaveVM` no mesmo fluxo (core#131).

## Docs locais

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — arquitetura atual
- [docs/PRODUCT.md](docs/PRODUCT.md) — produto
- [RELEASES.md](RELEASES.md) — processo de release
- [CONTRIBUTING.md](CONTRIBUTING.md)
