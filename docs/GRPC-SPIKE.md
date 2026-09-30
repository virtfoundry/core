# gRPC spike (core#135)

> **Spike only.** Not a production API contract. REST remains canonical for UI and Terraform.

## What this is

Experimental `InstanceService` on package `virtfoundry.iaas.v1alpha1`:

| RPC | Behavior in this spike |
|-----|-------------------------|
| `ListInstances` | Thin adapter over `internal/service/compute` (`ListVMs`) |
| `GetInstance` | Thin adapter over `GetVM` |
| `WatchInstances` | Initial `MODIFIED` snapshot, then hub-backed stream of `vm.created` / `vm.updated` / `vm.deleted` (same hub as `/ws/events`) until client disconnect |

Transport: **grpc-go behind [cmux](https://github.com/soheilhy/cmux) on the same ClusterIP port as REST (`:8080`)**. No separate Ingress path for gRPC in this spike. No `:9090` / `GRPCRoute` unless cmux proves unusable later.

## Auth (fail-closed)

Metadata only — **never** `?token=`:

- `authorization: Bearer <jwt|vfd_live_...>`
- `x-tenant-id` — required for root (impersonation); forbidden for non-root

Missing/invalid credentials or unresolved tenant → `Unauthenticated`.

Permission (parity with REST `AutoPermission` for `GET /vms`): non-root needs `vms:read`; root bypasses.

## Explicitly out of scope

- gRPC in **operator**
- Killing or replacing REST
- VNC / console over gRPC (stays WS + ticket)
- Migrating Terraform provider to gRPC
- Helm Ingress / Gateway API gRPC exposure

## Enable / disable

| Env | Effect |
|-----|--------|
| *(unset)* / any value except `0` | cmux dual-stack (default) |
| `VIRTFOUNDRY_GRPC=0` | HTTP-only (fail-soft if you need to bisect) |

If cmux exits with error, the process logs and **falls back to HTTP-only** on `:8080`.

## Proto / codegen

```bash
./scripts/generate-proto.sh   # needs protoc + protoc-gen-go + protoc-gen-go-grpc
```

Generated Go lives under `api/gen/` and is committed.

## Try List (grpcurl)

```bash
# After login, with JWT + tenant:
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -H "x-tenant-id: $TENANT_ID" \
  virtfoundry.homelab:8080 \
  virtfoundry.iaas.v1alpha1.InstanceService/ListInstances
```

ClusterIP / in-cluster clients are the intended consumers; do not put gRPC on the public Ingress for this spike.

## Watch stream mapping

| Hub event (`/ws/events`) | `WatchInstances` `event_type` |
|--------------------------|-------------------------------|
| (initial ListVMs) | `MODIFIED` snapshot |
| `vm.created` | `ADDED` |
| `vm.updated` | `MODIFIED` |
| `vm.deleted` | `DELETED` |

ADDED/MODIFIED try to enrich via `GetVM`; hub payload alone carries `id`/`name`/`state`. Non-`vm.*` hub events are ignored.

## TODO (remaining)

- [ ] Root `all_tenants` scope on Watch (WS supports it; gRPC pins to resolved tenant)
- [ ] Method-map interceptor for future RPCs beyond `vms:read` (today handlers call `requireVMsRead`)
- [ ] Helm note only (no Ingress) — chart change only if docs need a flag
- [ ] Decide production port strategy (`:8080` cmux vs `GRPCRoute`)
- [ ] SDK / client stubs for in-cluster operators (not TF yet)

> Note: epic [#135](https://github.com/virtfoundry/core/issues/135) is closed after the cmux spike (#162). Hub Watch is a follow-up on that track.
