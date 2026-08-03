# IAM plan — users, API keys, permissions

Prerequisite for **Terraform provider**, automation, and multi-user tenants.

**Rule:** [.cursor/rules/auth-iam.mdc](../.cursor/rules/auth-iam.mdc)  
**Blocks:** [TERRAFORM-PROVIDER-PLAN.md](./TERRAFORM-PROVIDER-PLAN.md) until Phase 2 (API keys) minimum.

---

## Goal

Each **tenant** has multiple **users**. Each user has:

- Login credentials (username + password) **or**
- One or more **API keys** (long-lived, revocable, scoped)

Every API call is authorized by **permissions**, not a single coarse role. Code stays centralized: one permission matrix, middleware enforcement, audit trail.

```
                    ┌─────────────┐
  Browser login ──► │ JWT (short) │
  Terraform/CI  ──► │ API key     │──► middleware ──► permission check ──► handler
                    └─────────────┘
```

---

## Current state (v0.2.x)

| Piece | Status |
|-------|--------|
| Roles `root`, `tenant_admin`, `user` in `platform.User` | Model only |
| Bootstrap `root` from `ROOT_PASSWORD` | Works |
| `{slug}-admin` on tenant create | Works |
| JWT Bearer auth | Works |
| `RequireRoot` for `/tenants` | Works |
| Tenant users CRUD | **Done** (`/users`, `/roles`) |
| API keys | **Done** (`/api-keys`, Bearer `vfd_live_...`) |
| Per-route permissions | **Done** (`AutoPermission` middleware) |
| `RoleUser` enforcement | **Done** via `role_id` + permission matrix |

---

## Target model

### Actors

| Actor | Scope | Notes |
|-------|-------|-------|
| **root** | All tenants | Platform operator; `X-Tenant-ID` to impersonate tenant context |
| **tenant user** | Single `tenant_id` | Normal IaaS user |
| **API key** | Same as owning user | Optional subset of user permissions (`scopes`) |

### Entities (new / extended)

```text
users              (existing — extend)
  id, tenant_id, username, password_hash, role_id, email, state, created_at

roles              (new — per tenant + system templates)
  id, tenant_id (null = system), name, description, is_system

role_permissions   (new)
  role_id, permission   -- e.g. "vms:create"

api_keys           (new)
  id, user_id, name, prefix, secret_hash, scopes_json, expires_at,
  last_used_at, revoked_at, created_at

user_invites       (optional phase 4)
  email, tenant_id, role_id, token_hash, expires_at
```

**API key format:** `vfd_live_<prefix>_<random>` — store only bcrypt/SHA256 of secret; show full key once on create.

### Permission strings (v1)

Namespaced `resource:action`. Start small; extend with new API resources.

| Permission | Routes / behavior |
|------------|-------------------|
| `tenants:read` | `GET /tenants` (root) |
| `tenants:write` | `POST /tenants` (root) |
| `users:read` | `GET /users` |
| `users:write` | `POST/PATCH/DELETE /users`, API keys |
| `vpcs:read` | `GET /vpcs`, cidr-plan |
| `vpcs:write` | `POST/PATCH/DELETE /vpcs` |
| `networks:read` | `GET /networks` |
| `networks:write` | `POST/PATCH/DELETE /networks` |
| `security_groups:read` | `GET /security-groups` |
| `security_groups:write` | `POST/PATCH/DELETE /security-groups` |
| `volumes:read` | `GET /volumes`, snapshots list |
| `volumes:write` | `POST /volumes`, snapshots create |
| `vms:read` | `GET /vms`, templates list, vm-snapshots list |
| `vms:write` | deploy, start, stop, delete, patch |
| `vms:console` | WebSocket console (optional gate) |
| `ssh_keys:read` | `GET /ssh-keys` |
| `ssh_keys:write` | create, register, delete |

### Built-in roles (templates)

| Role | Permissions | Who gets it |
|------|-------------|-------------|
| **platform.root** | `*` | Bootstrap root user |
| **tenant.admin** | All except `tenants:*` | `{slug}-admin` on tenant create |
| **tenant.operator** | read/write compute, network, storage, ssh; no `users:write` | DevOps |
| **tenant.viewer** | `*:read` only | Read-only |
| **tenant.custom** | Assigned per role record | Future |

Custom roles: tenant admin creates role + attaches permission list.

---

## API (new endpoints)

### Users (tenant-scoped; root with `X-Tenant-ID`)

| Method | Path | Permission |
|--------|------|------------|
| GET | `/api/v1/users` | `users:read` |
| POST | `/api/v1/users` | `users:write` |
| PATCH | `/api/v1/users/{id}` | `users:write` |
| DELETE | `/api/v1/users/{id}` | `users:write` |

Body: `username`, `password`, `role_id` or `role_name`, `email`.

### Roles (tenant-scoped)

| Method | Path | Permission |
|--------|------|------------|
| GET | `/api/v1/roles` | `users:read` |
| POST | `/api/v1/roles` | `users:write` |
| PATCH | `/api/v1/roles/{id}` | `users:write` |
| DELETE | `/api/v1/roles/{id}` | `users:write` (not system roles) |

### API keys (own keys or admin for tenant)

| Method | Path | Permission |
|--------|------|------------|
| GET | `/api/v1/api-keys` | `users:read` (own) / all if admin |
| POST | `/api/v1/api-keys` | `users:write` — returns secret **once** |
| DELETE | `/api/v1/api-keys/{id}` | revoke |

Body: `name`, `expires_in_days`, optional `scopes[]` (must be ⊆ user permissions).

### Auth changes

| Method | Path | Notes |
|--------|------|-------|
| POST | `/api/v1/auth/login` | unchanged (JWT) |
| POST | `/api/v1/auth/token` | optional: exchange API key → short JWT |

**Authentication header:**

- JWT: `Authorization: Bearer <jwt>`
- API key: `Authorization: Bearer vfd_live_...` (detect by prefix) **or** `X-API-Key: vfd_live_...`

Middleware resolves to `Actor` context: `{ UserID, TenantID, Permissions []string, AuthMethod }`.

---

## Code layout (keep control centralized)

```text
internal/auth/
  auth.go           # JWT (existing)
  apikey.go         # hash, validate, prefix parse
  permissions.go    # constants + AllPermissions slice
  authorize.go      # HasPermission(actor, perm), wildcard * support

internal/service/identity/
  service.go        # extend: users, roles, keys
  roles.go          # built-in role seeds
  apikeys.go

internal/api/middleware/
  jwt.go            # extend: API key path
  authorize.go      # RequirePermission("vms:write")

internal/platform/models.go
  RoleRecord, APIKey, ...

internal/platform/store/
  migrations for users/roles/api_keys
```

**Rule:** handlers call `middleware.RequirePermission` — never `if claims.Role == ...` for tenant resources (except root bypass).

---

## Phased implementation

### Phase 1 — Roles & permissions foundation

- [x] `permissions.go` + `authorize.go` middleware
- [x] DB tables: `roles`, `role_permissions`; migrate `users.role` → `users.role_id`
- [x] Seed system roles (`platform.root`, `tenant.admin`, `tenant.operator`, `tenant.viewer`)
- [x] Wire **existing routes** to permission checks (`AutoPermission` middleware)
- [x] Unit tests: denied without permission, admin passes

**Exit criteria:** tenant admin still works; viewer gets 403 on `POST /vms`.

### Phase 2 — Tenant users CRUD

- [x] `GET/POST/PATCH/DELETE /users`
- [x] `GET/POST/PATCH/DELETE /roles` (custom roles)
- [ ] Root lists users across tenants (optional `GET /tenants/{id}/users`)
- [ ] Audit log: user create/delete

**Exit criteria:** second user in tenant with `tenant.viewer` role.

### Phase 3 — API keys

- [x] `api_keys` table + CRUD endpoints
- [x] Auth middleware accepts API key → load user + permissions
- [x] Optional scope restriction on key
- [ ] `last_used_at` update (async/throttled)

**Exit criteria:** `curl -H "Authorization: Bearer vfd_live_..." /api/v1/vms` works with correct 403/200.

### Phase 4 — UI & polish

- [x] UI: Users, Roles, API Keys (tenant admin) — `/iam`
- [x] Copy key once modal; revoke button
- [ ] Password change for self (`PATCH /auth/me/password`)

### Phase 5 — Terraform provider unblocked

- [ ] Provider `api_key` attribute documented
- [ ] Resume [TERRAFORM-PROVIDER-PLAN.md](./TERRAFORM-PROVIDER-PLAN.md) Phase 0

---

## Security notes

- API key secret shown **once** on create (like GitHub PAT)
- Store `secret_hash` only (bcrypt or HMAC-SHA256 with server pepper)
- Keys inherit tenant boundary — cannot escalate to another tenant
- Root API keys: discouraged; document platform break-glass only
- Rate-limit `POST /auth/login` and failed API key attempts (later)

---

## Open decisions

| Topic | Recommendation | Status |
|-------|----------------|--------|
| Key in Bearer vs header | Bearer with `vf_` prefix detection | TBD |
| JWT for API key | Optional short JWT exchange | Defer |
| Casbin vs static matrix | Static matrix v1; Casbin if roles explode | **Static v1** |
| Memory store | Parity for dev/tests | Required |

---

## Tracking

| Phase | Status | Target |
|-------|--------|--------|
| 1 — Permissions middleware | **done** | |
| 2 — Users & roles API | **done** | |
| 3 — API keys | **done** | |
| 4 — UI | **done** | |
| 5 — Terraform provider | **unblocked** | resume TF plan Phase 0 |
