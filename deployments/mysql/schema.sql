-- Nimbus Cloud MySQL Schema
-- Compatible with CloudStack data migration

CREATE DATABASE IF NOT EXISTS nimbus_cloud;
USE nimbus_cloud;

-- Zones
CREATE TABLE zones (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    description TEXT,
    state VARCHAR(20) NOT NULL DEFAULT 'Disabled',
    network_type VARCHAR(20) NOT NULL DEFAULT 'Basic',
    dns1 VARCHAR(45),
    dns2 VARCHAR(45),
    internal_dns1 VARCHAR(45),
    internal_dns2 VARCHAR(45),
    guest_cidr VARCHAR(20),
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    removed DATETIME,
    INDEX idx_zones_name (name),
    INDEX idx_zones_state (state)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Pods
CREATE TABLE pods (
    id VARCHAR(36) PRIMARY KEY,
    zone_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    gateway VARCHAR(45) NOT NULL,
    netmask VARCHAR(45) NOT NULL,
    start_ip VARCHAR(45) NOT NULL,
    end_ip VARCHAR(45) NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'Disabled',
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    removed DATETIME,
    FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE,
    INDEX idx_pods_zone (zone_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Clusters
CREATE TABLE clusters (
    id VARCHAR(36) PRIMARY KEY,
    pod_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    hypervisor VARCHAR(50) NOT NULL DEFAULT 'KVM',
    cluster_type VARCHAR(20) NOT NULL DEFAULT 'CloudManaged',
    state VARCHAR(20) NOT NULL DEFAULT 'Disabled',
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    removed DATETIME,
    FOREIGN KEY (pod_id) REFERENCES pods(id) ON DELETE CASCADE,
    INDEX idx_clusters_pod (pod_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Hosts
CREATE TABLE hosts (
    id VARCHAR(36) PRIMARY KEY,
    cluster_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    ip_address VARCHAR(45) NOT NULL UNIQUE,
    hypervisor VARCHAR(50) NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'Down',
    resource_state VARCHAR(50) NOT NULL DEFAULT 'Disabled',
    cpu_count INT DEFAULT 0,
    cpu_cores INT DEFAULT 0,
    memory_total BIGINT DEFAULT 0,
    memory_used BIGINT DEFAULT 0,
    storage_total BIGINT DEFAULT 0,
    storage_used BIGINT DEFAULT 0,
    version VARCHAR(50),
    agent_version VARCHAR(50),
    last_ping DATETIME,
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    removed DATETIME,
    FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE CASCADE,
    INDEX idx_hosts_cluster (cluster_id),
    INDEX idx_hosts_state (state)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Storage Pools
CREATE TABLE storage_pool (
    id VARCHAR(36) PRIMARY KEY,
    zone_id VARCHAR(36) NOT NULL,
    cluster_id VARCHAR(36),
    name VARCHAR(255) NOT NULL,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    storage_type VARCHAR(20) NOT NULL,
    protocol VARCHAR(20),
    scope VARCHAR(20) NOT NULL DEFAULT 'CLUSTER',
    host_address VARCHAR(255),
    path VARCHAR(500),
    capacity BIGINT NOT NULL DEFAULT 0,
    used BIGINT NOT NULL DEFAULT 0,
    over_provision_factor DECIMAL(5,2) DEFAULT 2.00,
    state VARCHAR(30) NOT NULL DEFAULT 'Down',
    tags TEXT,
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    removed DATETIME,
    FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE,
    INDEX idx_storage_pool_zone (zone_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Domains
CREATE TABLE domain (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    parent_id VARCHAR(36),
    parent_path VARCHAR(500),
    level INT DEFAULT 0,
    state VARCHAR(20) NOT NULL DEFAULT 'Active',
    network_domain VARCHAR(255),
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    removed DATETIME,
    FOREIGN KEY (parent_id) REFERENCES domain(id) ON DELETE CASCADE,
    INDEX idx_domain_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Accounts
CREATE TABLE account (
    id VARCHAR(36) PRIMARY KEY,
    account_name VARCHAR(255) NOT NULL,
    account_type SMALLINT NOT NULL DEFAULT 1,
    domain_id VARCHAR(36) NOT NULL,
    domain_path VARCHAR(500),
    state VARCHAR(20) NOT NULL DEFAULT 'enabled',
    removed DATETIME,
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (domain_id) REFERENCES domain(id),
    INDEX idx_account_name (account_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Users
CREATE TABLE user (
    id VARCHAR(36) PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    account_id VARCHAR(36) NOT NULL,
    username VARCHAR(255) NOT NULL,
    password VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    email VARCHAR(255),
    state VARCHAR(20) NOT NULL DEFAULT 'enabled',
    api_key VARCHAR(255) UNIQUE,
    secret_key VARCHAR(255),
    time_zone VARCHAR(50),
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    removed DATETIME,
    FOREIGN KEY (account_id) REFERENCES account(id) ON DELETE CASCADE,
    UNIQUE KEY uk_user_account_username (account_id, username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Service Offerings
CREATE TABLE service_offering (
    id VARCHAR(36) PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    cpu_count INT NOT NULL,
    cpu_cores INT NOT NULL,
    memory BIGINT NOT NULL,
    rate_limit_cpu INT,
    rate_limit_ram INT,
    tags TEXT,
    offer_ha BOOLEAN DEFAULT FALSE,
    storage_type VARCHAR(20),
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_service_offering_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Templates
CREATE TABLE vm_template (
    id VARCHAR(36) PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    display_text VARCHAR(500),
    os_type VARCHAR(100),
    hypervisor VARCHAR(50),
    format VARCHAR(20),
    url VARCHAR(500),
    size BIGINT DEFAULT 0,
    physical_size BIGINT DEFAULT 0,
    state VARCHAR(30) NOT NULL DEFAULT 'NotProcessed',
    is_public BOOLEAN DEFAULT FALSE,
    is_featured BOOLEAN DEFAULT FALSE,
    is_ready BOOLEAN DEFAULT FALSE,
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    zone_id VARCHAR(36),
    INDEX idx_template_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Virtual Machines
CREATE TABLE vm_instance (
    id VARCHAR(36) PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    account_id VARCHAR(36) NOT NULL,
    domain_id VARCHAR(36) NOT NULL,
    zone_id VARCHAR(36) NOT NULL,
    template_id VARCHAR(36),
    service_offering_id VARCHAR(36),
    host_id VARCHAR(36),
    cluster_id VARCHAR(36),
    pod_id VARCHAR(36),
    state VARCHAR(30) NOT NULL DEFAULT 'Stopped',
    ha_enabled BOOLEAN DEFAULT FALSE,
    hypervisor VARCHAR(50),
    vnc_port INT,
    vnc_password VARCHAR(255),
    cpu_count INT DEFAULT 0,
    memory BIGINT DEFAULT 0,
    os_type VARCHAR(100),
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    removed DATETIME,
    last_start_time DATETIME,
    last_stop_time DATETIME,
    FOREIGN KEY (account_id) REFERENCES account(id) ON DELETE CASCADE,
    FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE,
    INDEX idx_vm_account (account_id),
    INDEX idx_vm_zone (zone_id),
    INDEX idx_vm_state (state),
    INDEX idx_vm_host (host_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Volumes
CREATE TABLE volumes (
    id VARCHAR(36) PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    account_id VARCHAR(36) NOT NULL,
    domain_id VARCHAR(36) NOT NULL,
    zone_id VARCHAR(36) NOT NULL,
    pool_id VARCHAR(36) NOT NULL,
    vm_id VARCHAR(36),
    state VARCHAR(30) NOT NULL DEFAULT 'Allocated',
    type VARCHAR(20) NOT NULL,
    size BIGINT NOT NULL,
    storage_type VARCHAR(20) NOT NULL,
    hypervisor VARCHAR(50),
    path VARCHAR(500),
    iqn VARCHAR(255),
    device_id INT,
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    removed DATETIME,
    attached_time DATETIME,
    FOREIGN KEY (account_id) REFERENCES account(id) ON DELETE CASCADE,
    FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE,
    INDEX idx_volumes_account (account_id),
    INDEX idx_volumes_vm (vm_id),
    INDEX idx_volumes_pool (pool_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Snapshots
CREATE TABLE snapshots (
    id VARCHAR(36) PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    volume_id VARCHAR(36) NOT NULL,
    account_id VARCHAR(36) NOT NULL,
    domain_id VARCHAR(36) NOT NULL,
    state VARCHAR(30) NOT NULL DEFAULT 'Creating',
    type VARCHAR(20) NOT NULL,
    location_type VARCHAR(20) DEFAULT 'Primary',
    path VARCHAR(500),
    size BIGINT DEFAULT 0,
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    removed DATETIME,
    FOREIGN KEY (volume_id) REFERENCES volumes(id) ON DELETE CASCADE,
    FOREIGN KEY (account_id) REFERENCES account(id) ON DELETE CASCADE,
    INDEX idx_snapshots_volume (volume_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Networks
CREATE TABLE networks (
    id VARCHAR(36) PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    zone_id VARCHAR(36) NOT NULL,
    account_id VARCHAR(36) NOT NULL,
    domain_id VARCHAR(36) NOT NULL,
    type VARCHAR(30) NOT NULL,
    traffic_type VARCHAR(30) NOT NULL,
    vlan INT,
    gateway VARCHAR(45),
    cidr VARCHAR(25),
    netmask VARCHAR(45),
    start_ip VARCHAR(45),
    end_ip VARCHAR(45),
    state VARCHAR(30) NOT NULL DEFAULT 'Allocated',
    is_persistent BOOLEAN DEFAULT FALSE,
    is_default BOOLEAN DEFAULT FALSE,
    network_offering_id VARCHAR(36),
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE,
    FOREIGN KEY (account_id) REFERENCES account(id) ON DELETE CASCADE,
    INDEX idx_networks_zone (zone_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- NICs
CREATE TABLE nics (
    id VARCHAR(36) PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    instance_id VARCHAR(36) NOT NULL,
    network_id VARCHAR(36) NOT NULL,
    ip_address VARCHAR(45),
    mac_address VARCHAR(17),
    netmask VARCHAR(45),
    gateway VARCHAR(45),
    is_default BOOLEAN DEFAULT FALSE,
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    removed DATETIME,
    FOREIGN KEY (instance_id) REFERENCES vm_instance(id) ON DELETE CASCADE,
    FOREIGN KEY (network_id) REFERENCES networks(id) ON DELETE CASCADE,
    INDEX idx_nics_instance (instance_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Async Jobs
CREATE TABLE async_jobs (
    id VARCHAR(36) PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    cmd VARCHAR(255) NOT NULL,
    cmd_version VARCHAR(50),
    instance_id VARCHAR(36),
    instance_type VARCHAR(50),
    account_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    status INT NOT NULL DEFAULT 0,
    result TEXT,
    error_code INT,
    error_msg TEXT,
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed DATETIME,
    INDEX idx_async_jobs_account (account_id),
    INDEX idx_async_jobs_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- API Keys
CREATE TABLE api_keys (
    api_key VARCHAR(255) PRIMARY KEY,
    secret_key VARCHAR(255) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    account_id VARCHAR(36) NOT NULL,
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    removed DATETIME,
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
    FOREIGN KEY (account_id) REFERENCES account(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Migration tracking
CREATE TABLE migration_info (
    id INT PRIMARY KEY AUTO_INCREMENT,
    version VARCHAR(50) NOT NULL,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    description TEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

INSERT INTO migration_info (version, description) VALUES ('1.0.0', 'Initial Nimbus Cloud schema');
