# Application config

VirtFoundry reads a YAML config file at startup (`CONFIG_PATH`, default `config/config.yaml`).

## Local development (this repo)

```bash
cp config/config.yaml.example config/config.yaml   # optional — defaults work with memory store
ROOT_PASSWORD=virtfoundry go run ./cmd/server
```

| File | Tracked | Purpose |
|------|---------|---------|
| `config.yaml.example` | yes | Minimal template for `go run` / Docker build |
| `config.yaml` | no (gitignored) | Your local overrides |

Secrets for local dev can also come from env: `JWT_SECRET`, `ROOT_PASSWORD`.

## Kubernetes (helm-charts)

Cluster runtime config is **not** maintained here. The Helm chart renders a ConfigMap from `values.yaml`:

```
helm-charts/charts/virtfoundry/values.yaml  →  ConfigMap  →  /etc/virtfoundry/config.yaml
```

Sensitive values (`JWT_SECRET`, `ROOT_PASSWORD`) are injected via Kubernetes Secrets as env vars on the API pod.

To generate a local `config.yaml` that matches a Helm profile:

```bash
cd ../helm-charts
make render-local-config                                    # default values
make render-local-config VALUES=./charts/virtfoundry/values-gateway.yaml
```

See [helm-charts/docs/CONFIGURATION.md](https://github.com/virtfoundry/helm-charts/blob/main/docs/CONFIGURATION.md) for the full values reference.
