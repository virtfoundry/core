# VM Templates

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
- HTTP(S) URL to the ISO file
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

## Deploying VMs with templates

On the **VMs** page, pick a template from the dropdown (fedora from tenant catalog, ubuntu/cirros from platform). Link to `/templates` is provided to register more images.

Service offerings (CPU/memory) are listed separately via `/api/v1/service-offerings`.
