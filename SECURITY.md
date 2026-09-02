# Security Policy

Aligned with the [CNCF project template](https://github.com/cncf/project-template/blob/main/SECURITY.md) security policy shape.

## Supported versions

| Version | Supported |
|---------|-----------|
| Latest `v0.Y.Z` release | yes |
| `main` branch | yes |
| Older 0.x tags | best-effort |

## Reporting a vulnerability

**Do not open public GitHub issues for security vulnerabilities.**

Prefer:

1. **GitHub Private Vulnerability Reporting** on the affected repo (e.g. [core security advisories](https://github.com/virtfoundry/core/security/advisories/new))
2. Contact maintainers in [MAINTAINERS.md](MAINTAINERS.md) — primary: **Matheus Thurler** ([@Matheus-Thurler](https://github.com/Matheus-Thurler))

Include:

- Description of the issue and impact
- Steps to reproduce
- Affected component (API, UI, operator, CRDs, Helm chart, Terraform provider) and versions
- Suggested fix (if any)

We aim to **acknowledge within 7 days** and provide an estimated fix timeline. Fixes ship via tagged releases and GitHub Security Advisories when applicable.

## Disclosure policy

When an issue is confirmed:

1. Develop and test a fix
2. Assign a CVE if appropriate
3. Release a patched version
4. Publish a security advisory

## Security Response Team

Lead maintainer (see [MAINTAINERS.md](MAINTAINERS.md)). Additional maintainers may assist as needed.

## Secure deployment notes

- Change default `ROOT_PASSWORD` and `JWT_SECRET` before any non-lab deployment
- Platform state lives in `virtfoundry.io` CRs; **never** put password or API-key hashes in CR `spec` — use Kubernetes Secrets only
- Restrict API ingress and enable TLS at the ingress controller
- Run API and operator ServiceAccounts with least-privilege RBAC (review Helm chart / operator `config/rbac`)
- Pin container images by digest in production GitOps overlays
