#!/bin/bash
set -e
export PATH="$HOME/bin:$PATH"

echo "=== KubeVirt Installer (Official) ==="

# Get latest version
VERSION=$(curl -s https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirt/stable.txt)
VERSION=${VERSION:-"v1.8.4"}
echo "Version: $VERSION"

# Check cluster
kubectl get nodes

# Deploy operator
echo "Instalando operator..."
kubectl create -f "https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-operator.yaml"

# Deploy CRDs
echo "Instalando CRDs..."
kubectl create -f "https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-cr.yaml"

# Wait for deployment (namespace: kubevirt)
echo "Aguardando KubeVirt ficar pronto (namespace: kubevirt)..."
kubectl -n kubevirt wait --for=condition=Ready kv/kubevirt --timeout=600s

# Install virtctl
echo "Instalando virtctl..."
ARCH=$(uname -s | tr A-Z a-z)-$(uname -m | sed 's/x86_64/amd64/')
curl -L -o virtctl "https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/virtctl-${VERSION}-${ARCH}"
chmod +x virtctl && mkdir -p ~/bin && mv virtctl ~/bin/virtctl

echo ""
echo "=== STATUS ==="
kubectl get kubevirt -n kubevirt
kubectl get pods -n kubevirt
