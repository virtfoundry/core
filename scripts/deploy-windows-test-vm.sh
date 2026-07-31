#!/usr/bin/env bash
# Deploy a Windows Server 2022 eval VM for IOPS testing with pod-network internet (masquerade).
#
# First boot: install Windows via Nimbus console VNC (~30-45 min). After install, use datadisk for IOPS.
# Internet: pod masquerade (no Multus network) — DHCP inside guest via KubeVirt.
#
# Requires CDI (scripts/setup-homelab-cdi.sh). Blank disks use CDI DataVolumes because local-path
# direct PVC provisioning often times out on this homelab.
#
# Usage:
#   KUBE_CONTEXT=homelab TENANT_NS=nimbus-tenant-acme ./scripts/deploy-windows-test-vm.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
KUBE_CONTEXT="${KUBE_CONTEXT:-homelab}"
TENANT_NS="${TENANT_NS:-nimbus-tenant-acme}"
VM_NAME="${VM_NAME:-win-iops-test}"
BOOT_SIZE="${BOOT_SIZE:-32Gi}"
DATA_SIZE="${DATA_SIZE:-16Gi}"
ISO_SIZE="${ISO_SIZE:-8Gi}"
CPU="${CPU:-4}"
MEMORY="${MEMORY:-16Gi}"
STORAGE_CLASS="${STORAGE_CLASS:-local-path}"
ISO_URL="${ISO_URL:-https://go.microsoft.com/fwlink/?linkid=2195280}"
VIRTIO_IMAGE="${VIRTIO_IMAGE:-quay.io/kubevirt/virtio-container-disk:v1.8.4}"
ISO_WAIT="${ISO_WAIT:-3600}"

BOOT_DV="${VM_NAME}-boot"
DATA_DV="${VM_NAME}-data"
ISO_DV="${VM_NAME}-iso"

echo "==> Context: $KUBE_CONTEXT | namespace: $TENANT_NS | VM: $VM_NAME"
kubectl config use-context "$KUBE_CONTEXT"

if ! kubectl get ns "$TENANT_NS" &>/dev/null; then
  echo "ERROR: namespace $TENANT_NS not found"
  exit 1
fi

if ! kubectl get cdi cdi -n cdi &>/dev/null; then
  echo "==> CDI not found — installing"
  KUBE_CONTEXT="$KUBE_CONTEXT" "$ROOT/scripts/setup-homelab-cdi.sh"
fi

if ! kubectl -n kube-system get pod -l app=multus -o jsonpath='{.items[0].status.phase}' 2>/dev/null | grep -q Running; then
  echo "WARN: Multus not Running — restarting daemonset"
  kubectl -n kube-system delete pod -l app=multus --force --grace-period=0 2>/dev/null || true
  kubectl -n kube-system rollout status daemonset/kube-multus-ds --timeout=120s || true
fi

dv_apply() {
  local name=$1 yaml=$2
  if kubectl -n "$TENANT_NS" get "datavolume/$name" &>/dev/null; then
    phase="$(kubectl -n "$TENANT_NS" get "datavolume/$name" -o jsonpath='{.status.phase}')"
    if [ "$phase" = "Succeeded" ]; then
      echo "==> DataVolume $name already imported"
      return 0
    fi
  fi
  kubectl apply -f - <<EOF
apiVersion: cdi.kubevirt.io/v1beta1
kind: DataVolume
metadata:
  name: ${name}
  namespace: ${TENANT_NS}
  labels:
    app.kubernetes.io/managed-by: nimbus-iaas
    nimbus.io/vm: ${VM_NAME}
  annotations:
    cdi.kubevirt.io/storage.bind.immediate.requested: "true"
spec:
${yaml}
EOF
}

echo "==> Boot disk DataVolume ($BOOT_SIZE)"
dv_apply "$BOOT_DV" "$(cat <<EOF
  source:
    blank: {}
  pvc:
    accessModes: [ReadWriteOnce]
    storageClassName: ${STORAGE_CLASS}
    resources:
      requests:
        storage: ${BOOT_SIZE}
EOF
)"

echo "==> Data disk for IOPS tests ($DATA_SIZE)"
dv_apply "$DATA_DV" "$(cat <<EOF
  source:
    blank: {}
  pvc:
    accessModes: [ReadWriteOnce]
    storageClassName: ${STORAGE_CLASS}
    resources:
      requests:
        storage: ${DATA_SIZE}
EOF
)"

echo "==> Windows ISO DataVolume (download may take several minutes)"
dv_apply "$ISO_DV" "$(cat <<EOF
  source:
    http:
      url: "${ISO_URL}"
  pvc:
    accessModes: [ReadWriteOnce]
    storageClassName: ${STORAGE_CLASS}
    resources:
      requests:
        storage: ${ISO_SIZE}
EOF
)"

echo "==> Waiting for PVCs to bind"
for dv in "$BOOT_DV" "$DATA_DV" "$ISO_DV"; do
  kubectl -n "$TENANT_NS" wait "pvc/$dv" --for=jsonpath='{.status.phase}'=Bound --timeout=300s
done

echo "==> Waiting for ISO import (timeout ${ISO_WAIT}s)"
kubectl -n "$TENANT_NS" wait "datavolume/${ISO_DV}" --for=condition=Ready --timeout="${ISO_WAIT}s"

# Blank DVs may leave orphan importer pods; safe to remove once PVCs are Bound.
kubectl -n "$TENANT_NS" delete pod -l cdi.kubevirt.io/dataVolume --force --grace-period=0 2>/dev/null || true

echo "==> VirtualMachine"
kubectl apply -f - <<EOF
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: ${VM_NAME}
  namespace: ${TENANT_NS}
  labels:
    app.kubernetes.io/managed-by: nimbus-iaas
    nimbus.io/os: windows
spec:
  runStrategy: Always
  template:
    metadata:
      labels:
        kubevirt.io/domain: ${VM_NAME}
        nimbus.io/vm: ${VM_NAME}
        nimbus.io/log-source: velas
    spec:
      domain:
        machine:
          type: q35
        firmware:
          bootloader:
            efi:
              secureBoot: false
        devices:
          disks:
            - name: bootdisk
              disk:
                bus: sata
            - name: datadisk
              disk:
                bus: sata
            - name: installiso
              bootOrder: 1
              disk:
                bus: sata
            - name: virtiodrivers
              cdrom:
                bus: sata
          interfaces:
            - name: default
              masquerade: {}
              model: e1000
        features:
          hyperv: {}
        resources:
          requests:
            memory: ${MEMORY}
            cpu: "${CPU}"
      networks:
        - name: default
          pod: {}
      volumes:
        - name: bootdisk
          persistentVolumeClaim:
            claimName: ${BOOT_DV}
        - name: datadisk
          persistentVolumeClaim:
            claimName: ${DATA_DV}
        - name: installiso
          persistentVolumeClaim:
            claimName: ${ISO_DV}
        - name: virtiodrivers
          containerDisk:
            image: ${VIRTIO_IMAGE}
EOF

echo "==> Waiting for VMI to start..."
kubectl -n "$TENANT_NS" wait vmi/"${VM_NAME}" --for=jsonpath='{.status.phase}'=Running --timeout=600s || true

NODE_IP="$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')"
echo ""
echo "=== Windows test VM deployed ==="
echo "VM:        $VM_NAME"
echo "Namespace: $TENANT_NS"
echo "Console:   http://${NODE_IP}:30880/console?name=${VM_NAME}&namespace=${TENANT_NS}"
echo "UI login:  root / nimbus  (select tenant acme in header)"
echo ""
echo "First boot: Windows installer from ISO. Install on bootdisk; datadisk stays empty for IOPS."
echo "After install: load VirtIO drivers from virtio CD, then diskspd/CrystalDiskMark on datadisk."
echo "Internet: masquerade pod network — guest DHCP after Windows setup."
echo ""
kubectl -n "$TENANT_NS" get vm,vmi,pvc 2>/dev/null | grep -E "${VM_NAME}|NAME" || kubectl -n "$TENANT_NS" get vm,vmi
