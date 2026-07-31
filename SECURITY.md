# Security Policy

## Supported versions

| Version | Supported |
|---------|-----------|
| main branch | yes |
| tagged releases | yes |

## Reporting a vulnerability

**Do not open public GitHub issues for security vulnerabilities.**

Please report security issues by emailing the repository owner via GitHub (private security advisory) or by contacting maintainers listed on the [virtforge-cloud organization profile](https://github.com/virtforge-cloud).

Include:

- Description of the issue and impact
- Steps to reproduce
- Affected component (API, UI, worker, charts)
- Suggested fix (if any)

We aim to acknowledge reports within 7 days.

## Secure deployment notes

- Change default `ROOT_PASSWORD` and `JWT_SECRET` before any non-lab deployment
- Use external MySQL with strong credentials when `mysql.enabled=false` in Helm
- Restrict API ingress and enable TLS at the ingress controller
- Run worker and API service accounts with least-privilege RBAC (review `deployments/k8s/base/rbac.yaml`)
