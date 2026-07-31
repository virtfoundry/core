#!/bin/bash
# Instala KubeVirt no homelab seguindo https://kubevirt.io/quickstart_cloud/
# NÃO remove recursos existentes — apenas adiciona namespace kubevirt.
set -euo pipefail

KUBE_CONTEXT="${KUBE_CONTEXT:-homelab}"

echo "=== KubeVirt quickstart — $KUBE_CONTEXT ==="
kubectl config use-context "$KUBE_CONTEXT"

if kubectl get kubevirt.kubevirt.io/kubevirt -n kubevirt &>/dev/null; then
  echo "KubeVirt já instalado — pulando deploy"
else
  VERSION=$(curl -s https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirt/stable.txt)
  echo "Versão stable: $VERSION"
  kubectl create -f "https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-operator.yaml"
  kubectl create -f "https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-cr.yaml"
  echo "Aguardando KubeVirt..."
  kubectl -n kubevirt wait --for=condition=Available kubevirt.kubevirt.io/kubevirt --timeout=600s
fi

kubectl get kubevirt.kubevirt.io/kubevirt -n kubevirt -o=jsonpath="{.status.phase}{'\n'}"
kubectl get all -n kubevirt
kubectl describe node mgmt-01 | grep -i "devices.kubevirt.io/kvm" || true

echo ""
echo "API local apontando para homelab:"
echo "  kubectl config use-context $KUBE_CONTEXT"
echo "  cd nimbus-iaas && ROOT_PASSWORD=nimbus go run ./cmd/server"
echo ""
echo "UI: http://localhost:3000 — login root/nimbus, selecione tenant no header"
