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

Sensitive values (`JWT_SECRET`, `ROOT_PASSWORD`) are injected via Kubernetes Secrets as env vars on the API pod. The chart mounts the rendered config at `/etc/virtfoundry/config.yaml`; `CONFIG_PATH` is wired to that path.

To generate a local `config.yaml` that matches a Helm profile:

```bash
cd ../helm-charts
make render-local-config                                    # default values
make render-local-config VALUES=./charts/virtfoundry/values-gateway.yaml
```

See [Configuration guide](https://virtfoundry.github.io/helm-charts/docs/guide/configuration/) for the full values reference.

## Standalone Docker image (not Helm)

The published image no longer bakes `config.yaml.example` into it (the file shipped a `JWT_SECRET` placeholder that fails validation). A bare `docker run` of the image with no config mount will exit 1 with:

```
failed to load config "...": failed to read config: ...
```

This is **fail-closed by design** — `JWT_SECRET` is now mandatory, so the only supported way to run the API is via the Helm chart (or by mounting your own `config.yaml` and supplying `JWT_SECRET` / `ROOT_PASSWORD` env vars). Do not copy the old Dockerfile path verbatim on operators' machines; point them to the Helm chart instead.
