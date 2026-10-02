# gRPC as default transport — migration plan

**Date:** 2026-10-01  
**Status:** Planning (not implementing in the console-refine wave)  
**Repos:** `virtfoundry/core` (API surface + UI client), later `terraform-provider-virtfoundry`, docs  
**Related:** `docs/GRPC-SPIKE.md` (InstanceService spike — **superseded as policy**), `docs/VKS-API.md`, `docs/superpowers/specs/2026-10-01-vks-phase4-grpc-design.md`

## Problem

Today the **product contract** for VKS is already gRPC-canonical (`ClusterService`), but the **default client path** is still REST (`platformFetch` → `/api/v1/...`). IaaS `InstanceService` is still labeled a spike; Terraform and the React UI speak REST only. That locks a dual stack forever and delays Watch/streaming benefits.

Goal: **gRPC (or Connect) becomes the default client transport**; REST shrinks to transitional/compat until removed.

## Non-goals (this planning doc)

- Implementing the migration in the same PRs as VKS console refine.
- Removing REST in one big bang.
- Putting raw HTTP/2 gRPC on public Ingress without a browser-compatible layer.
- Operator ↔ core gRPC.

## Current state

| Consumer | Transport today | Server |
|----------|-----------------|--------|
| UI (`core/ui`) | REST JSON | mux on cmux `:8080` |
| Terraform provider | REST JSON | same |
| grpcurl / future CLI | gRPC | `InstanceService` + `ClusterService` on cmux |
| Browser WebSocket | `/ws/*` + tickets | same port (HTTP) |

Auth gRPC (already): `authorization: Bearer`, `x-tenant-id` fail-closed — keep.

## Target state

```
Browser UI ──► Connect-Web / gRPC-Web ──► Gateway ──► core :8080 (cmux)
TF / CLI   ──► grpc-go (native)      ──►              core :8080 (cmux)
Legacy     ──► REST shim (deprecated) ──►              same handlers → domain
```

- **Domain services stay one:** `internal/service/*` — no business logic in transports.
- **Canonical API = protobuf services** under `api/proto/virtfoundry/{iaas,vks}/v1alpha1`.
- REST becomes `Deprecated` in docs, then removed after TF+UI cutover.

## Approach options

| # | Approach | Pros | Cons |
|---|----------|------|------|
| A | **Connect (connect-go + connect-web)** over HTTP/1.1+JSON or proto | Browser-friendly; one codegen story; works through many proxies | New stack next to grpc-go (can dual-register) |
| B | Classic **gRPC-Web** + Envoy/grpcwebproxy | Pure gRPC server stays | Extra hop; Gateway complexity on homelab |
| C | Native gRPC only for TF/CLI; UI stays REST forever | Least UI work | Never “default gRPC” for the console |

**Recommendation: A (Connect)** as the UI/default path; keep **native gRPC** for TF/CLI/grpcurl on the same port via cmux (Connect can share handlers via connect adapters, or run Connect alongside grpc-go). Spike a thin PoC on `ClusterService.ListClusters` before migrating IaaS surface.

If Connect PoC fails behind Traefik/Gateway, fall back to **B** for browser only.

## Phased plan

### Phase 0 — Policy + inventory (docs, 1 PR)

- Rewrite `docs/GRPC-SPIKE.md` → `docs/GRPC.md`: InstanceService promoted from spike → **canonical alongside ClusterService**; REST = shim.
- Inventory every UI `platformFetch` route and map to existing/missing RPCs.
- Inventory TF client methods same way.
- Mark REST routes in OpenAPI/docs as transitional.

**Exit:** published inventory table (path → RPC or “needs proto”).

### Phase 1 — Browser transport PoC (core)

- Add Connect (or gRPC-Web) for **`ClusterService` only** (smallest surface).
- UI: one module `lib/grpc/vks-client.ts` calling `ListClusters` / `GetCluster`; feature flag `VITE_API_TRANSPORT=rest|grpc` (default `rest` until green).
- Homelab: confirm Gateway/HTTPRoute passes Connect/gRPC-Web (headers, HTTP/2 or h1 fallback).
- Auth: same Bearer + tenant metadata/headers.

**Exit:** flag-on List/Get VKS works on `virtfoundry.homelab`; flag-off unchanged.

### Phase 2 — VKS UI full cutover

- All VKS pages use gRPC/Connect (including `GetClusterSummary`, Watch if ready).
- REST `/api/v1/vks/*` remains for curl/smoke; no new REST features.

**Exit:** `VITE_API_TRANSPORT=grpc` default for VKS routes.

### Phase 3 — IaaS proto completion

- Expand `virtfoundry.iaas.v1alpha1` beyond List/Get/Watch Instances: networks, templates, offerings, SSH keys, IAM as needed by UI create wizards (or batch “PlatformService” carefully — prefer resource-shaped services).
- Wire Connect + grpc-go for each as it lands.
- Promote InstanceService docs off “spike”.

**Exit:** DeployVMWizard catalogs callable via gRPC/Connect.

### Phase 4 — UI default = gRPC

- Flip default `VITE_API_TRANSPORT=grpc` for all migrated routes.
- REST only for: health (optional), WS ticket minting (can stay HTTP), anything not yet ported.
- Remove dead `platformFetch` call sites as they migrate.

**Exit:** UI production build defaults to gRPC; CI smoke both transports until Phase 5.

### Phase 5 — Terraform provider

- Add grpc client (native) using same protos; feature flag or major version bump.
- Migrate resources starting with `virtfoundry_vks_cluster`, then VM.
- Document Registry upgrade notes.

**Exit:** TF examples use gRPC endpoint; REST client deprecated.

### Phase 6 — REST sunset

- Announce deprecation window (e.g. one minor on 0.x).
- Delete REST handlers + AutoPermission path tables that only served UI.
- Keep `/health` (and maybe a tiny legacy login) if needed for probes.

**Exit:** no `/api/v1` resource CRUD (or stub 410 Gone).

## Gateway / homelab notes

- Today UI hits `http://virtfoundry.homelab` → HTTPRoute → UI pod proxies API, or API path via nginx in UI chart — **verify** whether browser talks to API through UI nginx or direct. Connect/gRPC-Web must be allowed on that path (path prefix `/virtfoundry.vks.v1alpha1.ClusterService/` or Connect paths).
- cmux already demuxes HTTP1/HTTP2 on `:8080`; Connect-JSON can ride HTTP/1.1 through Traefik more easily than h2c-only gRPC.

## Auth & IAM

- Reuse interceptor pattern from `internal/api/grpc` (Bearer + tenant + permission).
- Map every RPC to existing permission keys (`vks:read`, `vms:read`, …) — no parallel ACL.
- Login may stay REST POST initially; later `AuthService.Login` RPC.

## Risks

| Risk | Mitigation |
|------|------------|
| Traefik/Gateway breaks gRPC-Web | PoC Phase 1 before IaaS expansion; Connect-JSON fallback |
| Dual codegen (Go + TS) drift | Single `buf`/`protoc` pipeline; CI check generated dirty |
| Watch + browser | Prefer Connect server-stream or keep WS hub until Watch proven |
| Large UI rewrite | Route-by-route flag; VKS first |

## Relation to VKS console refine

Console refine (catalogs, Nodes, Summary, TF REST resource) **ships on REST/shim** as planned. Summary RPC is added to `ClusterService` so Phase 2 of **this** migration picks it up without a second API. Do not block refine on Connect.

## Suggested first implementation PR (later)

1. Docs: `GRPC.md` + inventory.  
2. Connect PoC behind flag for `ListClusters`/`GetCluster` only.

## Open decisions (resolve at Phase 0–1 kickoff)

1. Connect vs gRPC-Web (recommend Connect).  
2. Whether UI nginx proxies gRPC or browser talks to API Service DNS on LAN.  
3. TF major bump vs soft flag for gRPC endpoint.
