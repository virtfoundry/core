# Application config

VirtForge reads a YAML config file at startup (`CONFIG_PATH`, default `config/config.yaml`).

## Local development (this repo)

```bash
cp config/config.yaml.example config/config.yaml   # optional — defaults work with memory store
ROOT_PASSWORD=virtforge go run ./cmd/server
```

| File | Tracked | Purpose |
|------|---------|---------|
| `config.yaml.example` | yes | Minimal template for `go run` / Docker build |
| `config.yaml` | no (gitignored) | Your local overrides |

Secrets for local dev can also come from env: `JWT_SECRET`, `ROOT_PASSWORD`.

## Kubernetes (virtforge-chart)

Cluster runtime config is **not** maintained here. The Helm chart renders a ConfigMap from `values.yaml`:

```
virtforge-chart/charts/virtforge/values.yaml  →  ConfigMap  →  /etc/virtforge/config.yaml
```

Sensitive values (`JWT_SECRET`, `ROOT_PASSWORD`) are injected via Kubernetes Secrets as env vars on the API pod.

To generate a local `config.yaml` that matches a Helm profile:

```bash
cd ../virtforge-chart
make render-local-config                                    # default values
make render-local-config VALUES=./charts/virtforge/values-homelab.yaml
```

See [virtforge-chart/docs/CONFIGURATION.md](https://github.com/virtforge-cloud/virtforge-chart/blob/main/docs/CONFIGURATION.md) for the full values reference.
