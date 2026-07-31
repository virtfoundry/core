#!/usr/bin/env bash
# Deploy Nimbus Cloud to homelab (in-cluster only — no local API/UI servers).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
KUBE_CONTEXT="${KUBE_CONTEXT:-homelab}"
PLATFORM="${PLATFORM:-linux/amd64}"
OVERLAY="${ROOT}/deployments/k8s/overlays/homelab"
IMPORT_POD="nimbus-image-import"
IMPORT_NS="nimbus-system"

echo "==> Context: $KUBE_CONTEXT"
kubectl config use-context "$KUBE_CONTEXT"

echo "==> KubeVirt feature gates (Snapshot, VideoConfig)"
kubectl patch kubevirt kubevirt -n kubevirt --type merge \
  -p '{"spec":{"configuration":{"developerConfiguration":{"featureGates":["Snapshot","VideoConfig"]}}}}' \
  2>/dev/null || echo "WARN: enable KubeVirt feature gates manually"

echo "==> Multus CNI + bridge (homelab networking)"
if [ -x "$ROOT/scripts/setup-homelab-multus.sh" ]; then
  KUBE_CONTEXT="$KUBE_CONTEXT" "$ROOT/scripts/setup-homelab-multus.sh"
else
  echo "WARN: setup-homelab-multus.sh not found — skip"
fi

echo "==> Build images ($PLATFORM)"
docker buildx build --platform "$PLATFORM" -t nimbus/iaas-api:latest -f "$ROOT/deployments/Dockerfile" "$ROOT" --load
docker buildx build --platform "$PLATFORM" -t nimbus/iaas-ui:latest -f "$ROOT/deployments/Dockerfile.ui" "$ROOT" --load

UI_STATIC="$(mktemp -d)"
trap 'rm -rf "$UI_STATIC" "$TMPDIR" 2>/dev/null || true' EXIT
UI_CID="$(docker create nimbus/iaas-ui:latest)"
docker cp "${UI_CID}:/usr/share/nginx/html/." "$UI_STATIC/"
docker rm "$UI_CID" >/dev/null

echo "==> Apply manifests"
kubectl apply -k "$OVERLAY"

echo "==> Wait for MySQL"
kubectl -n nimbus-system rollout status statefulset/nimbus-mysql --timeout=300s || true
kubectl -n nimbus-system wait --for=condition=ready pod -l app=nimbus-mysql --timeout=300s || true

echo "==> UI ConfigMaps"
kubectl delete configmap nimbus-ui-assets -n nimbus-system --ignore-not-found
kubectl create configmap nimbus-ui-static -n nimbus-system \
  --from-file="$UI_STATIC/index.html" \
  --from-file="$UI_STATIC/50x.html" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl create configmap nimbus-ui-assets -n nimbus-system \
  --from-file="$UI_STATIC/assets"

echo "==> Import images to node containerd via privileged pod"
kubectl -n "$IMPORT_NS" delete pod "$IMPORT_POD" --ignore-not-found --wait=true 2>/dev/null || true
kubectl apply -f "$OVERLAY/image-import-pod.yaml"
kubectl -n "$IMPORT_NS" wait --for=condition=Ready pod/"$IMPORT_POD" --timeout=120s
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT
docker save nimbus/iaas-api:latest -o "$TMPDIR/nimbus-api.tar"
docker save nimbus/iaas-ui:latest -o "$TMPDIR/nimbus-ui.tar"

kubectl cp "$TMPDIR/nimbus-api.tar" "${IMPORT_NS}/${IMPORT_POD}:/import/nimbus-api.tar"
kubectl cp "$TMPDIR/nimbus-ui.tar" "${IMPORT_NS}/${IMPORT_POD}:/import/nimbus-ui.tar"
kubectl -n "$IMPORT_NS" exec "$IMPORT_POD" -- sh -c \
  'cp /import/nimbus-api.tar /host/tmp/nimbus-api.tar && cp /import/nimbus-ui.tar /host/tmp/nimbus-ui.tar && \
   chroot /host ctr -n k8s.io images rm docker.io/nimbus/iaas-api:latest docker.io/nimbus/iaas-ui:latest 2>/dev/null || true && \
   chroot /host ctr -n k8s.io content rm sha256:3008bbb2d5c5d4e93168955a94eeaf5a0dec7a40535543f2a682a5cec1f4799f 2>/dev/null || true && \
   chroot /host ctr -n k8s.io images import /tmp/nimbus-api.tar && \
   chroot /host ctr -n k8s.io images import /tmp/nimbus-ui.tar && \
   chroot /host ctr -n k8s.io images label docker.io/nimbus/iaas-api:latest io.cri-containerd.image=managed && \
   chroot /host ctr -n k8s.io images label docker.io/nimbus/iaas-ui:latest io.cri-containerd.image=managed && \
   chroot /host systemctl restart containerd'
sleep 8

echo "==> Restart workloads"
kubectl -n nimbus-system rollout restart deployment/nimbus-api deployment/nimbus-ui deployment/nimbus-worker
kubectl -n nimbus-system rollout status deployment/nimbus-api --timeout=180s
kubectl -n nimbus-system rollout status deployment/nimbus-ui --timeout=180s
kubectl -n nimbus-system rollout status deployment/nimbus-worker --timeout=180s

kubectl -n "$IMPORT_NS" delete pod "$IMPORT_POD" --ignore-not-found

NODE_IP="$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')"
echo ""
echo "=== Nimbus Cloud deployed ==="
echo "UI:  http://${NODE_IP}:30880"
echo "API: http://${NODE_IP}:30880/api/v1 (via UI nginx proxy)"
echo "Login: root / nimbus"
echo ""
kubectl -n nimbus-system get pods,svc
