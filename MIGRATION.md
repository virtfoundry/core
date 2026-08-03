# Moved to VirtFoundry

**VirtForge Cloud** has been rebranded to **[VirtFoundry](https://github.com/virtfoundry)**.

| Old | New |
|-----|-----|
| [virtforge-cloud/virtforge](https://github.com/virtforge-cloud/virtforge) | **[virtfoundry/core](https://github.com/virtfoundry/core)** |
| [virtforge-cloud/virtforge-chart](https://github.com/virtforge-cloud/virtforge-chart) | **[virtfoundry/helm-charts](https://github.com/virtfoundry/helm-charts)** |

## Install (1.0.0+)

```bash
helm repo add virtfoundry https://virtfoundry.github.io/helm-charts
helm install virtfoundry virtfoundry/virtfoundry --version 1.0.0 \
  -n virtfoundry-system --create-namespace
```

Documentation: https://virtfoundry.github.io/helm-charts/docs/

---

This repository is a **read-only mirror** during migration. New issues and releases: use the VirtFoundry org.
