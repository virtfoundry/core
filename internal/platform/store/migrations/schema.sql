-- VirtForge Cloud platform schema (MySQL 8+)
-- Import-friendly: external_uuid + import_source nullable unique per entity.

CREATE DATABASE IF NOT EXISTS virtforge CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE virtforge;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version     VARCHAR(64) PRIMARY KEY,
    applied_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

CREATE TABLE IF NOT EXISTS users (
    id              CHAR(36) PRIMARY KEY,
    username        VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    role            VARCHAR(32) NOT NULL,
    tenant_id       CHAR(36) NULL,
    email           VARCHAR(255) NULL,
    created_at      DATETIME(3) NOT NULL
);

CREATE TABLE IF NOT EXISTS tenants (
    id              CHAR(36) PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    slug            VARCHAR(64) NOT NULL UNIQUE,
    namespace       VARCHAR(255) NOT NULL,
    state           VARCHAR(32) NOT NULL DEFAULT 'Enabled',
    external_uuid   VARCHAR(64) NULL,
    import_source   VARCHAR(32) NULL,
    created_at      DATETIME(3) NOT NULL,
    UNIQUE KEY uk_tenant_external (import_source, external_uuid)
);

CREATE TABLE IF NOT EXISTS vpcs (
    id              CHAR(36) PRIMARY KEY,
    tenant_id       CHAR(36) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    cidr            VARCHAR(64) NOT NULL,
    namespace       VARCHAR(255) NOT NULL,
    state           VARCHAR(32) NOT NULL DEFAULT 'Enabled',
    external_uuid   VARCHAR(64) NULL,
    import_source   VARCHAR(32) NULL,
    created_at      DATETIME(3) NOT NULL,
    INDEX idx_vpc_tenant (tenant_id),
    UNIQUE KEY uk_vpc_external (import_source, external_uuid)
);

CREATE TABLE IF NOT EXISTS security_groups (
    id              CHAR(36) PRIMARY KEY,
    tenant_id       CHAR(36) NOT NULL,
    vpc_id          CHAR(36) NULL,
    name            VARCHAR(255) NOT NULL,
    description     TEXT NULL,
    rules_json      JSON NOT NULL,
    external_uuid   VARCHAR(64) NULL,
    import_source   VARCHAR(32) NULL,
    created_at      DATETIME(3) NOT NULL,
    INDEX idx_sg_tenant (tenant_id),
    UNIQUE KEY uk_sg_tenant_name (tenant_id, name),
    UNIQUE KEY uk_sg_external (import_source, external_uuid)
);

CREATE TABLE IF NOT EXISTS networks (
    id              CHAR(36) PRIMARY KEY,
    tenant_id       CHAR(36) NULL,
    vpc_id          CHAR(36) NULL,
    name            VARCHAR(255) NOT NULL,
    network_type    VARCHAR(32) NOT NULL DEFAULT 'isolated',
    cidr            VARCHAR(64) NOT NULL,
    gateway         VARCHAR(64) NULL,
    nad_namespace   VARCHAR(255) NULL,
    nad_name        VARCHAR(255) NULL,
    state           VARCHAR(32) NOT NULL DEFAULT 'Enabled',
    external_uuid   VARCHAR(64) NULL,
    import_source   VARCHAR(32) NULL,
    created_at      DATETIME(3) NOT NULL,
    INDEX idx_net_tenant (tenant_id),
    INDEX idx_net_type (network_type),
    UNIQUE KEY uk_net_external (import_source, external_uuid)
);

CREATE TABLE IF NOT EXISTS service_offerings (
    id              CHAR(36) PRIMARY KEY,
    name            VARCHAR(255) NOT NULL UNIQUE,
    display_name    VARCHAR(255) NOT NULL,
    cpu             INT NOT NULL,
    memory_mi       BIGINT NOT NULL,
    storage_tags    VARCHAR(255) NULL,
    state           VARCHAR(32) NOT NULL DEFAULT 'Active',
    external_uuid   VARCHAR(64) NULL,
    import_source   VARCHAR(32) NULL,
    created_at      DATETIME(3) NOT NULL,
    UNIQUE KEY uk_offering_external (import_source, external_uuid)
);

CREATE TABLE IF NOT EXISTS vm_templates (
    id              CHAR(36) PRIMARY KEY,
    tenant_id       CHAR(36) NOT NULL DEFAULT '',
    name            VARCHAR(255) NOT NULL,
    display_name    VARCHAR(255) NOT NULL,
    description     TEXT NULL,
    image           VARCHAR(512) NOT NULL,
    source_type     VARCHAR(32) NOT NULL DEFAULT 'container',
    os_type         VARCHAR(64) NULL,
    cloud_init_user_data TEXT NULL,
    iso_volume_id       CHAR(36) NULL,
    iso_size_gi         INT NOT NULL DEFAULT 8,
    boot_disk_size_gi   INT NOT NULL DEFAULT 32,
    storage_class       VARCHAR(64) NULL,
    import_state        VARCHAR(32) NULL,
    hypervisor      VARCHAR(64) NOT NULL DEFAULT 'KubeVirt',
    state           VARCHAR(32) NOT NULL DEFAULT 'Active',
    external_uuid   VARCHAR(64) NULL,
    import_source   VARCHAR(32) NULL,
    created_at      DATETIME(3) NOT NULL,
    UNIQUE KEY uk_template_tenant_name (tenant_id, name),
    UNIQUE KEY uk_template_external (import_source, external_uuid)
);

CREATE TABLE IF NOT EXISTS vms (
    id                  CHAR(36) PRIMARY KEY,
    tenant_id           CHAR(36) NOT NULL,
    vpc_id              CHAR(36) NULL,
    name                VARCHAR(255) NOT NULL,
    display_name        VARCHAR(255) NULL,
    namespace           VARCHAR(255) NOT NULL,
    state               VARCHAR(32) NOT NULL DEFAULT 'Creating',
    error_message       TEXT NULL,
    cpu                 INT NOT NULL DEFAULT 1,
    memory_mi           BIGINT NOT NULL DEFAULT 1024,
    image               VARCHAR(512) NULL,
    template            VARCHAR(255) NULL,
    ip                  VARCHAR(64) NULL,
    hypervisor          VARCHAR(64) NULL,
    zone                VARCHAR(64) NULL,
    host_name           VARCHAR(255) NULL,
    service_offering_id CHAR(36) NULL,
    external_uuid       VARCHAR(64) NULL,
    import_source       VARCHAR(32) NULL,
    created_at          DATETIME(3) NOT NULL,
    updated_at          DATETIME(3) NULL,
    UNIQUE KEY uk_vm_tenant_name (tenant_id, name),
    UNIQUE KEY uk_vm_external (import_source, external_uuid),
    INDEX idx_vm_tenant (tenant_id)
);

CREATE TABLE IF NOT EXISTS vm_nics (
    id              CHAR(36) PRIMARY KEY,
    vm_id           CHAR(36) NOT NULL,
    name            VARCHAR(64) NOT NULL,
    ip              VARCHAR(64) NULL,
    mac             VARCHAR(64) NULL,
    nic_type        VARCHAR(32) NULL,
    network_id      CHAR(36) NULL,
    nad_namespace   VARCHAR(255) NULL,
    nad_name        VARCHAR(255) NULL,
    INDEX idx_nic_vm (vm_id),
    FOREIGN KEY (vm_id) REFERENCES vms(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS volumes (
    id              CHAR(36) PRIMARY KEY,
    tenant_id       CHAR(36) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    size_gi         INT NOT NULL,
    namespace       VARCHAR(255) NOT NULL,
    pvc_name        VARCHAR(255) NOT NULL,
    state           VARCHAR(32) NOT NULL DEFAULT 'Allocated',
    vm_id           CHAR(36) NULL,
    external_uuid   VARCHAR(64) NULL,
    import_source   VARCHAR(32) NULL,
    created_at      DATETIME(3) NOT NULL,
    INDEX idx_vol_tenant (tenant_id),
    UNIQUE KEY uk_vol_external (import_source, external_uuid)
);

CREATE TABLE IF NOT EXISTS snapshots (
    id              CHAR(36) PRIMARY KEY,
    tenant_id       CHAR(36) NOT NULL,
    volume_id       CHAR(36) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    namespace       VARCHAR(255) NOT NULL,
    snapshot_uid    VARCHAR(255) NULL,
    state           VARCHAR(32) NOT NULL DEFAULT 'Creating',
    external_uuid   VARCHAR(64) NULL,
    import_source   VARCHAR(32) NULL,
    created_at      DATETIME(3) NOT NULL,
    INDEX idx_snap_tenant (tenant_id),
    UNIQUE KEY uk_snap_external (import_source, external_uuid)
);

CREATE TABLE IF NOT EXISTS vm_snapshots (
    id              CHAR(36) PRIMARY KEY,
    tenant_id       CHAR(36) NOT NULL,
    vm_id           CHAR(36) NOT NULL,
    vm_name         VARCHAR(255) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    namespace       VARCHAR(255) NOT NULL,
    snapshot_uid    VARCHAR(255) NULL,
    phase           VARCHAR(32) NOT NULL DEFAULT 'Pending',
    external_uuid   VARCHAR(64) NULL,
    import_source   VARCHAR(32) NULL,
    created_at      DATETIME(3) NOT NULL,
    INDEX idx_vmsnap_tenant (tenant_id),
    UNIQUE KEY uk_vmsnap_external (import_source, external_uuid)
);

CREATE TABLE IF NOT EXISTS async_jobs (
    id              CHAR(36) PRIMARY KEY,
    tenant_id       CHAR(36) NOT NULL,
    job_type        VARCHAR(64) NOT NULL,
    status          VARCHAR(32) NOT NULL DEFAULT 'pending',
    payload         TEXT NULL,
    result          TEXT NULL,
    error_msg       TEXT NULL,
    created_at      DATETIME(3) NOT NULL,
    updated_at      DATETIME(3) NOT NULL,
    INDEX idx_job_status (status),
    INDEX idx_job_tenant (tenant_id)
);

CREATE TABLE IF NOT EXISTS ssh_key_pairs (
    id              CHAR(36) PRIMARY KEY,
    tenant_id       CHAR(36) NOT NULL,
    name            VARCHAR(128) NOT NULL,
    public_key      TEXT NOT NULL,
    fingerprint     VARCHAR(128) NOT NULL,
    created_at      DATETIME(3) NOT NULL,
    INDEX idx_sshkey_tenant (tenant_id),
    UNIQUE KEY uk_sshkey_tenant_name (tenant_id, name)
);

-- Default catalog (idempotent seeds applied by app on first boot)

CREATE TABLE IF NOT EXISTS audit_events (
    id               CHAR(36) PRIMARY KEY,
    actor_user_id    CHAR(36) NOT NULL,
    actor_role       VARCHAR(32) NOT NULL,
    target_tenant_id CHAR(36) NOT NULL,
    action           VARCHAR(64) NOT NULL,
    method           VARCHAR(16) NOT NULL,
    path             VARCHAR(512) NOT NULL,
    resource_type    VARCHAR(64) NULL,
    resource_id      VARCHAR(128) NULL,
    created_at       DATETIME(3) NOT NULL,
    INDEX idx_audit_tenant (target_tenant_id),
    INDEX idx_audit_actor (actor_user_id),
    INDEX idx_audit_created (created_at)
);

CREATE TABLE IF NOT EXISTS ip_addresses (
    id          CHAR(36) PRIMARY KEY,
    network_id  CHAR(36) NOT NULL,
    address     VARCHAR(64) NOT NULL,
    status      VARCHAR(32) NOT NULL DEFAULT 'available',
    vm_nic_id   CHAR(36) NULL,
    created_at  DATETIME(3) NOT NULL,
    UNIQUE KEY uk_ip_network_addr (network_id, address),
    INDEX idx_ip_status (network_id, status)
);
