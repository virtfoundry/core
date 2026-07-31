#!/usr/bin/env bash
# Install CDI (Containerized Data Importer) for ISO/DataVolume imports on homelab.
set -euo pipefail

KUBE_CONTEXT="${KUBE_CONTEXT:-homelab}"

echo "==> Context: $KUBE_CONTEXT"
kubectl config use-context "$KUBE_CONTEXT"

if kubectl get cdi cdi -n cdi &>/dev/null; then
  echo "==> CDI already installed"
  kubectl -n cdi get pods
  exit 0
fi

VERSION="${CDI_VERSION:-$(curl -fsSL https://api.github.com/repos/kubevirt/containerized-data-importer/releases/latest | grep '"tag_name"' | head -1 | cut -d'"' -f4)}"
echo "==> Installing CDI $VERSION"

kubectl apply -f "https://github.com/kubevirt/containerized-data-importer/releases/download/${VERSION}/cdi-operator.yaml"
kubectl apply -f "https://github.com/kubevirt/containerized-data-importer/releases/download/${VERSION}/cdi-cr.yaml"

echo "==> Waiting for CDI..."
kubectl -n cdi wait --for=condition=Available cdi/cdi --timeout=300s
kubectl -n cdi rollout status deployment/cdi-deployment --timeout=300s
kubectl -n cdi get pods
echo "==> CDI ready"
