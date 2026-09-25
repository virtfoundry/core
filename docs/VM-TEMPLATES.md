# VM Templates

> **Published guide:** [Images and templates](https://virtfoundry.github.io/helm-charts/docs/guide/features/templates/) on the VirtFoundry docs site.
> Prefer that page for end users; this file is kept for in-repo links and contributor context.

VirtFoundry VM templates define the OS image used when deploying a virtual machine. Templates can be **container disks** (cloud-init capable Linux images) or **ISO imports** (typically Windows).

## Platform vs tenant templates

| Scope | `tenant_id` | Who manages | Examples |
|-------|-------------|-------------|----------|
| **Platform** | empty | Seeded at startup; read-only in UI | `cirros`, `ubuntu-2204`, `windows-server-2022` |
| **Tenant** | tenant UUID | Created per tenant on bootstrap or via UI/API | `fedora-39` (default tenant seed) |

Tenants see **both** platform and their own templates when listing or deploying VMs. Platform templates are not duplicated per tenant — tenant bootstrap only adds images that are not already in the platform catalog.

## Container disk images

KubeVirt runs container disks as ephemeral root volumes. Use images with cloud-init support for Linux.

**Recommended sources:**

- [quay.io/containerdisks](https://quay.io/organization/containerdisks) — maintained OS images (Ubuntu, Fedora, CentOS, …)
- [KubeVirt demos](https://github.com/kubevirt/kubevirt/tree/main/containerimages) — e.g. `quay.io/kubevirt/cirros-container-disk-demo`, `quay.io/kubevirt/fedora-container-disk-demo`

**Examples:**

| Name | Image |
|------|-------|
| Cirros (demo) | `quay.io/kubevirt/cirros-container-disk-demo` |
| Ubuntu 22.04 | `quay.io/containerdisks/ubuntu:22.04` |
| Fedora 39 | `quay.io/kubevirt/fedora-container-disk-demo` |

## Register via UI

1. Select a tenant (root users: use the tenant switcher).
2. Open **Images & Templates** (`/templates`).
3. Click **Register image**.
4. Choose **Container disk** or **ISO (PVC)**.
5. For container disks: set name, display name, image URL, and optional extra `#cloud-config` user-data.

Platform templates appear with a **platform** badge and cannot be edited or deleted.

## Register via API

```bash
# List templates (platform + tenant)
curl -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  https://api.example.com/api/v1/vm-templates

# Register a container disk
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ubuntu-2404",
    "display_name": "Ubuntu 24.04",
    "image": "quay.io/containerdisks/ubuntu:24.04",
    "source_type": "container",
    "os_type": "linux"
  }' \
  https://api.example.com/api/v1/vm-templates
```

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/vm-templates` | List platform + tenant templates |
| POST | `/api/v1/vm-templates` | Create tenant template |
| PATCH | `/api/v1/vm-templates/{id}` | Update tenant template |
| DELETE | `/api/v1/vm-templates/{id}` | Delete tenant template |

## ISO import flow

ISO templates are used for Windows (or other OS installs from ISO). VirtFoundry uses **CDI** (Containerized Data Importer) to download the ISO into a DataVolume/PVC in the tenant namespace.

**Requirements:**

- CDI installed on the cluster (included in the [VirtFoundry Helm chart](https://github.com/virtfoundry/helm-charts))
- HTTPS URL to the ISO file, on a host the admin allows (see [Allowed ISO URLs](#allowed-iso-urls))
- StorageClass with ReadWriteOnce support (default: `local-path`)

**Flow:**

1. Create template with `source_type: "iso"` and `image` set to the ISO URL.
2. API sets `import_state: "importing"` and `state: "Inactive"`.
3. Background job creates a CDI HTTP import DataVolume in the tenant namespace.
4. On success: `import_state: "ready"`, `state: "Active"`, ISO linked as a volume.
5. On failure: `import_state: "failed"`, error stored in description.

**Optional fields for ISO templates:**

| Field | Default | Description |
|-------|---------|-------------|
| `iso_size_gi` | 8 | DataVolume size for the ISO |
| `boot_disk_size_gi` | 32 | Blank boot disk created at VM deploy |
| `storage_class` | cluster default | StorageClass for DataVolumes |

The UI polls the templates list every 5 seconds while any template is importing, showing a spinner badge on the card.

**Windows Server eval example:**

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "windows-server-2022",
    "display_name": "Windows Server 2022 Eval",
    "image": "https://go.microsoft.com/fwlink/?linkid=2195280",
    "source_type": "iso",
    "iso_size_gi": 8,
    "boot_disk_size_gi": 32
  }' \
  https://api.example.com/api/v1/vm-templates
```

VMs cannot be deployed from an ISO template until `import_state` is `ready`.

## Allowed ISO URLs

CDI downloads the ISO from **inside the cluster**, so the URL a tenant submits is fetched by a cluster pod. VirtFoundry therefore validates it before creating the DataVolume and answers `400` when it is not acceptable.

Always refused, regardless of configuration:

| Refused | Examples |
|---------|----------|
| Any scheme other than `https`, or a port other than 443 | `http://…`, `file:///…`, `https://host:9200/…` |
| URLs with embedded credentials | `https://user:pass@host/iso` |
| Loopback and localhost | `127.0.0.1`, `[::1]`, `localhost`, `*.localhost` |
| Link-local (cloud metadata) | `169.254.169.254`, `fe80::/10` |
| Private and shared address space | `10/8`, `172.16/12`, `192.168/16`, `fc00::/7`, `100.64/10` |
| Reserved ranges, incl. NAT64 re-encoding of the above | `0.0.0.0/8`, `240.0.0.0/4`, `64:ff9b::/96` |
| In-cluster, node and LAN names | `*.svc`, `*.local` (incl. `*.svc.cluster.local`), `*.internal`, `*.localdomain`, `*.home.arpa`, single labels such as `kubernetes` |

On top of that, the host must be on the admin **allowlist**. With no configuration, the built-in list covers the public sources used by this guide: Microsoft evaluation downloads (`go.microsoft.com`, `*.prss.microsoft.com`), Ubuntu / Debian / Fedora / Rocky / AlmaLinux install media, and object storage for pre-signed URLs (`s3.amazonaws.com`, `*.s3.amazonaws.com`, `*.blob.core.windows.net`, `storage.googleapis.com`, `*.r2.cloudflarestorage.com`).

To publish ISOs from your own mirror, list it explicitly — configuring `allowed_hosts` **replaces** the built-in list:

```yaml
security:
  iso_import:
    allowed_hosts:
      - "iso.mylab.example.com"      # exact host
      - "*.blob.core.windows.net"    # subdomains, not the apex
    disable_http_import: false       # true refuses every URL import
```

Equivalent environment variables, for a read-only ConfigMap:

```bash
VIRTFOUNDRY_ISO_ALLOWED_HOSTS="iso.mylab.example.com,*.blob.core.windows.net"
VIRTFOUNDRY_ISO_DISABLE_HTTP_IMPORT=1
```

The effective allowlist is logged at startup (`iso import allowlist`). With `disable_http_import: true`, tenants can still register an ISO template from an existing volume (`iso_volume_id`), which is the path to use for ISOs uploaded to a PVC.

### CDI importer egress NetworkPolicy

The allowlist is **name-based**. CDI resolves the hostname itself, so DNS rebinding of an allowlisted host (or a redirect that lands on a private address) can still make the importer fetch cluster-internal or metadata targets if the network path exists.

Defense in depth: on every tenant namespace ensure, VirtFoundry creates/updates `virtfoundry-cdi-importer-egress` — an **Egress** NetworkPolicy in the **tenant** namespace (where CDI importer pods run), not in the chart release namespace. It selects pods labeled `cdi.kubevirt.io=importer` and:

- Allows DNS to `kube-system` pods labeled `k8s-app=kube-dns` (UDP/TCP 53)
- Allows egress to `0.0.0.0/0` except RFC1918, link-local, and CGNAT (`10/8`, `172.16/12`, `192.168/16`, `169.254/16`, `100.64/10`)
- Allows egress to `::/0` except ULA / link-local / loopback (`fc00::/7`, `fe80::/10`, `::1/128`) for dual-stack clusters

Private ISO mirrors on RFC1918 need an extra allow rule on that NetworkPolicy (or a second policy) — see the [helm-charts templates guide](https://virtfoundry.github.io/helm-charts/docs/guide/features/templates/#cdi-importer-egress). The CNI must enforce NetworkPolicy for this to take effect. DNS rebinding to a *public* malicious IP remains an app-layer concern; keep the allowlist tight.

## Deploying VMs with templates

On the **VMs** page, pick a template from the dropdown (fedora from tenant catalog, ubuntu/cirros from platform). Link to `/templates` is provided to register more images.

Service offerings (CPU/memory) are listed separately via `/api/v1/service-offerings`.

Full feature guide: [Images and templates](https://virtfoundry.github.io/helm-charts/docs/guide/features/templates/).
