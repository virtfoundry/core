# Security Policy

## Supported versions

| Version | Supported |
|---------|-----------|
| `main` branch | yes |
| tagged releases | yes |

## Reporting a vulnerability

**Do not open public GitHub issues for security vulnerabilities.**

Report via a **private GitHub security advisory** on the affected repository, or contact maintainers listed in [MAINTAINERS.md](MAINTAINERS.md). Primary security contact: **Matheus Thurler** ([@Matheus-Thurler](https://github.com/Matheus-Thurler)).

Include:

- Description of the issue and impact
- Steps to reproduce
- Affected component (API, UI, operator, CRDs, Helm chart, Terraform provider)
- Suggested fix (if any)

We aim to acknowledge reports within **7 days** and publish fixes via tagged releases when applicable.

## Secure deployment notes

- Change default `ROOT_PASSWORD` and `JWT_SECRET` before any non-lab deployment
- Platform state lives in `virtfoundry.io` CRs; **never** put password or API-key hashes in CR `spec` — use Kubernetes Secrets only
- Restrict API ingress and enable TLS at the ingress controller
- Run API and operator ServiceAccounts with least-privilege RBAC (review Helm chart / operator `config/rbac`)
- Pin container images by digest in production GitOps overlays
