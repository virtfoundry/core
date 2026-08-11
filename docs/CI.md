# Continuous integration

Required checks on pull requests to `main` (GitHub rulesets / branch protection).

| Repository | Workflow | Required jobs | What it covers |
|------------|----------|---------------|----------------|
| [core](https://github.com/virtfoundry/core) | `CI` | `go`, `ui` | `go build` / `go test ./...`, UI `npm ci` + `npm run build` |
| [core](https://github.com/virtfoundry/core) | `Build and push images` | `build` | Docker build of API + UI images (push only on `main` / tags) |
| [helm-charts](https://github.com/virtfoundry/helm-charts) | `Chart lint` | `lint` | `helm lint`, `helm template`, MkDocs `--strict` |
| [terraform-provider-virtfoundry](https://github.com/virtfoundry/terraform-provider-virtfoundry) | `CI` | `test` | `gofmt`, `go vet`, `go test`, `go build` |

## Local equivalents

```bash
# core
go test ./...
cd ui && npm ci && npm run build

# helm-charts
helm lint ./charts/virtfoundry
helm template virtfoundry ./charts/virtfoundry
pip install -r requirements-docs.txt && mkdocs build --strict

# terraform-provider
gofmt -l .   # must be empty
go test ./...
```

Do **not** merge with failing required checks. Docs-only PRs still run the full matrix so `main` stays green for newcomers.

## Enforce on GitHub

Enable a repository ruleset named `protect-main` on each repo’s `main` branch requiring the job names above (`go`/`ui`/`build`, `lint`, `test`). Example for core (Settings → Rules → New ruleset, or API):

```json
{
  "name": "protect-main",
  "target": "branch",
  "enforcement": "active",
  "conditions": { "ref_name": { "include": ["refs/heads/main"], "exclude": [] } },
  "rules": [{
    "type": "required_status_checks",
    "parameters": {
      "strict_required_status_checks_policy": true,
      "required_status_checks": [
        { "context": "go" },
        { "context": "ui" },
        { "context": "build" }
      ]
    }
  }]
}
```
