# Terraform provider plan — `terraform-provider-virtfoundry`

Track provider work against the VirtFoundry REST API. Update this file when API resources change.

**Rule:** see [.cursor/rules/terraform-provider.mdc](../.cursor/rules/terraform-provider.mdc) — major API additions require a provider update before release.

!!! warning "Blocked on IAM"
    Do **not** start provider auth for production until [AUTH-IAM-PLAN.md](./AUTH-IAM-PLAN.md) Phase 3 (API keys) ships. Terraform needs long-lived `api_key` credentials with scoped permissions.

## Goal

First-party [Terraform](https://www.terraform.io/) provider so users manage VirtFoundry like any cloud:

```hcl
provider "virtfoundry" {
  endpoint  = "https://virtfoundry.example.com"
  api_key   = var.virtfoundry_api_key   # preferred after IAM Phase 3
  tenant_id = virtfoundry_tenant.demo.id
  # username/password — dev only; JWT expires
}

resource "virtfoundry_vm" "web" {
  name               = "web-01"
  template_id        = virtfoundry_vm_template.ubuntu.id
  service_offering_id = "small"
  security_group_ids = [virtfoundry_security_group.ssh.id]
}
```

The provider is a **thin client** over `/api/v1` — no direct KubeVirt/K8s calls from Terraform.

---

## Architecture

```
Terraform CLI
    → terraform-provider-virtfoundry (Plugin Framework)
        → VirtFoundry API (JWT + X-Tenant-ID)
            → worker → KubeVirt / Multus / PVC
```

| Layer | Responsibility |
|-------|----------------|
| Provider block | `endpoint`, credentials, default `tenant_id`, TLS |
| Resources | CRUD mapped to API; poll async jobs where needed |
| Data sources | Read-only catalogs (`service_offerings`, templates list) |
| Import | Support `terraform import` using API IDs |

**Out of scope v1:** WebSocket console, live VM logs, CloudStack `cmd/migrate`.

---

## API → Terraform mapping (current API v0.2.x)

Status legend: `—` not started · `plan` designed · `dev` in progress · `done` shipped · `n/a` not planned

| API route | Terraform | Status | Notes |
|-----------|-----------|--------|-------|
| `POST /auth/login` | Provider config | — | JWT cached in provider meta |
| `GET/POST /tenants` | `virtfoundry_tenant` | — | Root credentials only |
| `GET/POST/PATCH/DELETE /vpcs` | `virtfoundry_vpc` | — | |
| `GET/POST/PATCH/DELETE /networks` | `virtfoundry_network` | — | `vpc_id`, CIDR |
| `GET/POST/PATCH/DELETE /security-groups` | `virtfoundry_security_group` | — | Rules inline or `dynamic` block |
| `GET/POST /volumes` | `virtfoundry_volume` | — | Size, name; storage class gap in API |
| `GET/POST /snapshots` | `virtfoundry_volume_snapshot` | — | Volume snapshot |
| `GET/POST/PATCH/DELETE /vm-templates` | `virtfoundry_vm_template` | — | container + iso `source_type` |
| `GET/POST/PATCH/DELETE /vms` (+ start/stop/delete) | `virtfoundry_vm` | — | `desired_state`, networks, SGs, SSH key |
| `GET/POST /vm-snapshots` (+ restore/delete) | `virtfoundry_vm_snapshot` | — | Optional separate resource or VM nested |
| `GET/POST/DELETE /ssh-keys` | `virtfoundry_ssh_key` | — | Register public key material |
| `GET /service-offerings` | `virtfoundry_service_offerings` (data) | — | Read-only seed |
| `GET /vpcs/cidr-plan`, `/networks/cidr-plan` | data source (optional) | n/a | Can compute client-side |
| `POST /vms/{name}/ssh` | attribute on `virtfoundry_vm` | — | `expose_ssh`, SG linkage |
| WebSocket `/ws/console` | — | n/a | Use UI or virtctl |

---

## Phased roadmap

### Phase 0 — Repository bootstrap

- [ ] Create `virtfoundry/terraform-provider-virtfoundry` repo
- [ ] Scaffold with `terraform-plugin-framework` (Go)
- [ ] Provider block + login + health check
- [ ] CI: `go test`, `golangci-lint`, acceptance tests (optional VirtFoundry in Kind)
- [ ] Release: GitHub Releases + Terraform Registry publish workflow

**Exit criteria:** `terraform init` + `terraform plan` with empty config succeeds against live API.

### Phase 1 — Identity & tenancy

- [ ] `virtfoundry_tenant` (root)
- [ ] Provider `tenant_id` default + per-resource override
- [ ] Document root vs tenant-scoped tokens

### Phase 2 — Networking

- [ ] `virtfoundry_vpc`
- [ ] `virtfoundry_network`
- [ ] `virtfoundry_security_group` (ingress/egress rules)

**Exit criteria:** Terraform applies VPC + network + SG; VM can attach later.

### Phase 3 — Compute

- [ ] `virtfoundry_vm_template` (container disk first)
- [ ] `virtfoundry_vm` (deploy, start/stop via desired state)
- [ ] `virtfoundry_ssh_key` + VM SSH exposure attributes
- [ ] ISO template path after CDI fields stable

**Exit criteria:** `examples/minimal` — one Ubuntu VM with SSH SG from Terraform only.

### Phase 4 — Storage

- [ ] `virtfoundry_volume`
- [ ] `virtfoundry_volume_snapshot`
- [ ] `virtfoundry_vm_snapshot`

### Phase 5 — Docs & Registry

- [ ] Provider docs on GitHub Pages or Registry docs
- [ ] Link from virtfoundry-chart installation guide
- [ ] Example module: `examples/full-stack` (tenant + net + vm)

---

## Provider schema (draft)

```hcl
provider "virtfoundry" {
  endpoint            = string           # required, e.g. https://virtfoundry.example.com
  username            = string           # required
  password            = string           # required, sensitive
  tenant_id           = optional(string)  # default tenant for resources
  insecure            = optional(bool)    # skip TLS verify (dev only)
  jwt_expire_seconds  = optional(number)   # default from API
}
```

Environment variable fallbacks: `VIRTFORGE_ENDPOINT`, `VIRTFORGE_USERNAME`, `VIRTFORGE_PASSWORD`, `VIRTFORGE_TENANT_ID`.

---

## Sync workflow (ongoing)

When merging a **major API feature** in `virtfoundry`:

1. Add row or update **Status** in the mapping table above
2. Open matching issue/PR in `terraform-provider-virtfoundry`
3. Bump provider MINOR when resource ships; document min API version
4. Add acceptance test hitting real API (or httptest mock from OpenAPI fixture)

---

## Open questions

| Topic | Options | Decision |
|-------|---------|----------|
| Repo location | Same monorepo vs separate repo | **Separate repo** (HashiCorp Registry convention) |
| Async VM deploy | Poll job API vs poll VM state | TBD — inspect worker job exposure |
| VM start/stop | Separate resources vs `desired_state` | Prefer **`desired_state`** on `virtfoundry_vm` |
| Public IP on VM | Field on VM vs network attachment | Map `network_ids` + platform public pool config |

---

## References

- API routes: [cmd/server/main.go](../cmd/server/main.go)
- Architecture: [ARCHITECTURE.md](./ARCHITECTURE.md)
- Plugin Framework: https://developer.hashicorp.com/terraform/plugin/framework
- Registry publishing: https://developer.hashicorp.com/terraform/registry/providers/publishing

**Last updated:** 2026-08-03 (plan created; provider repo not scaffolded yet)
