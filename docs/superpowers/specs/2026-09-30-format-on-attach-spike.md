# Spike: format-on-attach for runtime volume attach (core#18)

**Status:** spike / design only — not shipping API or UI in this PR.  
**Date:** 2026-09-30

## Problem

Runtime attach (`POST /vms/{name}/volumes`) hot-plugs a PVC into KubeVirt. Unlike deploy-time `data_volume_id` (cloud-init `FormatDataDisk` → mkfs + mount `/mnt/iops`), cloud-init does **not** re-run on hot-plug. The guest sees a raw block device; UI already hints users to `mkfs`/`mount` manually.

## Why not small

A correct “format on attach” needs more than a request flag:

| Concern | Notes |
|---------|--------|
| Guest execution channel | Prefer qemu-guest-agent `guest-exec` (Linux). No agent → fail closed or queue for next boot. |
| Windows | Out of scope for v1; document manual Disk Management path. |
| Safety | Must refuse format of already-mounted / non-empty / wrong serial devices. Idempotent attach retries. |
| API surface | `format`, `fs_type`, `mount_path` on attach body; default mount e.g. `/mnt/data`. |
| Operator / CRD | If Instance CR owns disks, format intent belongs on attachment status — avoid dual-write with hypervisor driver. |
| UX | Checkbox “Format and mount (Linux only)” + clear failure toast when agent missing. |

Estimated: multi-PR (API contract → guest-agent path → UI → e2e), not a single low backlog ship.

## Proposed contract (future)

```http
POST /api/v1/vms/{name}/volumes
{
  "volume_id": "<uuid>",
  "format": true,
  "fs_type": "ext4",
  "mount_path": "/mnt/data"
}
```

Semantics:

1. Hot-plug PVC as today.
2. If `format=false`/omitted → current behavior (raw device, hint in UI).
3. If `format=true` → wait for disk visible in guest (agent), `mkfs.<fs_type>` once, mount at `mount_path`, record result on attachment / events. Errors are 409/422 with machine-readable reason (`guest_agent_unavailable`, `device_busy`, …).

Deploy-time path stays cloud-init `FormatDataDisk`; do not unify implementations in v1 — share docs only.

## Incremental next steps

1. Confirm guest-agent availability on default Linux templates (homelab smoke).
2. Spike PoC: attach + `guest-exec` mkfs against one Cirros/Ubuntu VM (throwaway branch).
3. Spec API errors + OpenAPI; open implementation PR behind feature note in CHANGELOG.
4. UI checkbox only after API returns structured status.

## Out of scope here

- Code changes to attach handler / KubeVirt driver / UI
- Windows guest tooling
- Auto-mount without explicit `format: true`
