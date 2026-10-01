# VKS API (Phase 4)

Canonical transport for VirtFoundry Kubernetes Service is **gRPC**
`virtfoundry.vks.v1alpha1.ClusterService` on cmux `:8080` (same port as REST).

REST `/api/v1/vks/clusters*` is a **thin transitional shim** for the console UI
(and curl smoke). Prefer gRPC for new clients; UI will migrate later.

## Auth

Metadata (never `?token=`):

- `authorization: Bearer <jwt|vfd_live_...>`
- `x-tenant-id` — required for root; forbidden for non-root

Permissions: `vks:read`, `vks:write`, `vks:kubeconfig`.

## grpcurl (in-cluster / port-forward to core service)

```bash
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  localhost:8080 \
  virtfoundry.vks.v1alpha1.ClusterService/ListClusters
```

## Spec

`docs/superpowers/specs/2026-10-01-vks-phase4-grpc-design.md`
