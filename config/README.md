# Application config

VirtFoundry reads a YAML config file at startup (`CONFIG_PATH`, default `config/config.yaml`).

## Local development (this repo)

```bash
cp config/config.yaml.example config/config.yaml   # optional — defaults work with memory store
JWT_SECRET="$(openssl rand -base64 32)" \
ROOT_PASSWORD="$(openssl rand -base64 18)" \
go run ./cmd/server
```

The API refuses to start when `JWT_SECRET` or `ROOT_PASSWORD` are missing, too short, or a known default. For local development only, `VF_ALLOW_INSECURE_DEFAULTS=1` opts out of these checks (the API will print a loud warning and fall back to the historical defaults — never use this flag in production).

If `ROOT_PASSWORD` is unset in a brand-new install (no root user in the store), the API generates a one-time strong password, bootstraps the root user with it, and logs it once. Save it from the boot log before it scrolls away.

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

See [Configuration guide](https://virtfoundry.github.io/helm-charts/docs/guide/configuration/) for the full values reference.
