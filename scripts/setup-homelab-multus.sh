#!/usr/bin/env bash
# Instala Multus CNI e prepara bridge nimbus-br0 nos nodes (homelab Linux + KVM).
set -euo pipefail

KUBE_CONTEXT="${KUBE_CONTEXT:-homelab}"
MULTUS_MANIFEST="${MULTUS_MANIFEST:-https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset-thick.yml}"

echo "==> Context: $KUBE_CONTEXT"
kubectl config use-context "$KUBE_CONTEXT"

echo "==> Multus CNI"
if kubectl get ds -n kube-system kube-multus-ds &>/dev/null; then
  echo "Multus já instalado"
else
  kubectl apply -f "$MULTUS_MANIFEST"
  kubectl -n kube-system rollout status daemonset/kube-multus-ds --timeout=300s
fi

echo "==> Bridge nimbus-br0 nos nodes"
kubectl apply -f - <<'EOF'
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: nimbus-bridge-setup
  namespace: kube-system
  labels:
    app.kubernetes.io/name: nimbus-bridge-setup
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: nimbus-bridge-setup
  template:
    metadata:
      labels:
        app.kubernetes.io/name: nimbus-bridge-setup
    spec:
      hostNetwork: true
      hostPID: true
      tolerations:
        - operator: Exists
      initContainers:
        - name: setup-bridge
          image: alpine:3.21
          securityContext:
            privileged: true
          command:
            - sh
            - -c
            - |
              set -e
              ip link show nimbus-br0 >/dev/null 2>&1 || ip link add nimbus-br0 type bridge
              ip link set nimbus-br0 up
              echo "nimbus-br0 ready on $(hostname)"
      containers:
        - name: pause
          image: registry.k8s.io/pause:3.10
          resources:
            requests:
              cpu: 1m
              memory: 8Mi
EOF

kubectl -n kube-system rollout status daemonset/nimbus-bridge-setup --timeout=120s
echo "=== Multus + nimbus-br0 prontos ==="
