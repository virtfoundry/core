const API_BASE = import.meta.env.VITE_PLATFORM_API_URL || '/api/v1';

function dispatchUnauthorized() {
  void import('../store').then(({ dispatchUnauthorized: dispatch }) => dispatch());
}

export interface PlatformUser {
  id: string;
  username: string;
  role: 'root' | 'tenant_admin' | 'user';
  tenant_id?: string;
  email?: string;
}

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  namespace: string;
  state: string;
}

export interface PlatformVM {
  id: string;
  name: string;
  display_name?: string;
  namespace: string;
  state: string;
  error_message?: string;
  cpu: number;
  memory_mi: number;
  image?: string;
  template?: string;
  ip?: string;
  hypervisor?: string;
  zone?: string;
  host_name?: string;
  service_offering_id?: string;
  tenant_id?: string;
  nics?: Array<{ name: string; ip?: string; mac?: string; type?: string }>;
  created_at?: string;
  updated_at?: string;
}

export interface Volume {
  id: string;
  name: string;
  size_gi: number;
  state: string;
  namespace: string;
  pvc_name: string;
  vm_id?: string;
}

export interface VPC {
  id: string;
  name: string;
  cidr: string;
  namespace: string;
  state: string;
}

export interface Network {
  id: string;
  name: string;
  cidr: string;
  vpc_id?: string;
  network_type?: 'isolated' | 'shared';
  gateway?: string;
  nad_namespace?: string;
  nad_name?: string;
  state: string;
}

export interface CIDRBlock {
  cidr: string;
  label: string;
  available: boolean;
  reason?: string;
}

export interface VPCCIDRPlan {
  suggestions: CIDRBlock[];
  existing: string[];
}

export interface SubnetCIDRPlan {
  vpc_cidr: string;
  prefix: number;
  auto?: string;
  suggestions: CIDRBlock[];
  used: string[];
}

export interface SecurityGroup {
  id: string;
  name: string;
  description?: string;
  vpc_id?: string;
  rules: Array<{
    direction: string;
    protocol: string;
    port_from?: number;
    port_to?: number;
    cidr: string;
  }>;
}

export interface Snapshot {
  id: string;
  name: string;
  volume_id: string;
  state: string;
}

export interface VMSnapshot {
  id: string;
  name: string;
  vm_id: string;
  vm_name: string;
  phase: string;
  namespace: string;
  created_at?: string;
}

export interface SSHKeyPair {
  id: string;
  tenant_id: string;
  name: string;
  public_key: string;
  fingerprint: string;
  created_at?: string;
}

export interface VMSSHInfo {
  exposed: boolean;
  node_port?: number;
  vm_ip?: string;
}

function getToken(): string | null {
  return localStorage.getItem('jwt_token');
}

function getTenantId(): string | null {
  return localStorage.getItem('tenant_id');
}

async function platformFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken();
  const tenantId = getTenantId();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  if (tenantId) headers['X-Tenant-ID'] = tenantId;

  const res = await fetch(`${API_BASE}${path}`, { ...options, headers });
  if (res.status === 401 && !path.startsWith('/auth/login')) {
    dispatchUnauthorized();
    throw new Error('Session expired');
  }
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    const msg = body.error || body.errortext || body.errorText || res.statusText;
    throw new Error(msg);
  }
  return body as T;
}

export async function login(username: string, password: string) {
  return platformFetch<{ token: string; user: PlatformUser }>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  });
}

export async function getMe() {
  return platformFetch<{ id: string; username: string; role: string; tenant_id?: string }>('/auth/me');
}

export async function listTenants() {
  return platformFetch<{ tenants: Tenant[] }>('/tenants');
}

export async function createTenant(data: { name: string; slug: string; admin_password: string }) {
  return platformFetch<{ tenant: Tenant }>('/tenants', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function listVPCs() {
  const res = await platformFetch<{ vpcs: VPC[] | null }>('/vpcs');
  return { vpcs: res.vpcs ?? [] };
}

export async function fetchVPCCIDRPlan() {
  return platformFetch<VPCCIDRPlan>('/vpcs/cidr-plan');
}

export async function fetchNetworkCIDRPlan(vpcId: string, prefix = 24) {
  return platformFetch<SubnetCIDRPlan>(`/networks/cidr-plan?vpc_id=${encodeURIComponent(vpcId)}&prefix=${prefix}`);
}

export async function createVPC(data: { name: string; cidr?: string }) {
  return platformFetch<{ vpc: VPC }>('/vpcs', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateVPC(id: string, data: { name: string }) {
  return platformFetch<{ vpc: VPC }>(`/vpcs/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  });
}

export async function deleteVPC(id: string) {
  return platformFetch<{ success: boolean }>(`/vpcs/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export async function listVMs() {
  return platformFetch<{ vms: PlatformVM[] }>('/vms');
}

export async function getVM(name: string) {
  return platformFetch<{ vm: PlatformVM; velas_url?: string }>(`/vms/${encodeURIComponent(name)}`);
}

export async function fetchVMLogs(name: string, tail = 200) {
  const token = localStorage.getItem('jwt_token') || '';
  const tenantId = localStorage.getItem('tenant_id');
  const headers: Record<string, string> = { Authorization: `Bearer ${token}` };
  if (tenantId) headers['X-Tenant-ID'] = tenantId;
  const res = await fetch(`${API_BASE}/vms/${encodeURIComponent(name)}/logs?tail=${tail}`, { headers });
  if (res.status === 401) {
    dispatchUnauthorized();
    throw new Error('Session expired');
  }
  const text = await res.text();
  if (!res.ok) {
    try {
      const body = JSON.parse(text);
      throw new Error(body.error || text);
    } catch {
      throw new Error(text || 'Erro ao carregar logs');
    }
  }
  return text;
}

export async function updateVM(name: string, data: {
  display_name?: string;
  cpu?: number;
  memory_mi?: number;
  service_offering_id?: string;
}) {
  return platformFetch<{ vm: PlatformVM }>(`/vms/${encodeURIComponent(name)}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  });
}

export async function deployVM(data: {
  name: string;
  image?: string;
  template_id?: string;
  service_offering_id?: string;
  cpu?: number;
  memory_mi?: number;
  dedicated_cpu?: boolean;
  network_ids?: string[];
  public_ip?: boolean;
  security_group_ids?: string[];
  ssh_key_id?: string;
  data_volume_id?: string;
  expose_ssh?: boolean;
  display_name?: string;
}) {
  return platformFetch<{ vm: PlatformVM }>('/vms', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function startVM(name: string) {
  return platformFetch('/vms/start', { method: 'POST', body: JSON.stringify({ name }) });
}

export async function stopVM(name: string) {
  return platformFetch('/vms/stop', { method: 'POST', body: JSON.stringify({ name }) });
}

export async function deleteVM(name: string) {
  return platformFetch('/vms/delete', { method: 'POST', body: JSON.stringify({ name }) });
}

export async function listVolumes() {
  return platformFetch<{ volumes: Volume[] }>('/volumes');
}

export async function createVolume(data: { name: string; size_gi: number }) {
  return platformFetch<{ volume: Volume }>('/volumes', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function deleteVolume(id: string) {
  return platformFetch(`/volumes/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export async function listVMVolumes(vmName: string) {
  return platformFetch<{ volumes: Volume[] }>(`/vms/${encodeURIComponent(vmName)}/volumes`);
}

export async function attachVolumeToVM(vmName: string, volumeId: string) {
  return platformFetch<{ volume: Volume }>(`/vms/${encodeURIComponent(vmName)}/volumes`, {
    method: 'POST',
    body: JSON.stringify({ volume_id: volumeId }),
  });
}

export async function detachVolumeFromVM(vmName: string, volumeId: string) {
  return platformFetch<{ volume: Volume }>(
    `/vms/${encodeURIComponent(vmName)}/volumes/${encodeURIComponent(volumeId)}`,
    { method: 'DELETE' },
  );
}

export async function listNetworks() {
  const res = await platformFetch<{ networks: Network[] | null }>('/networks');
  return { networks: res.networks ?? [] };
}

export async function createNetwork(data: { name: string; cidr?: string; vpc_id: string; prefix?: number }) {
  return platformFetch<{ network: Network }>('/networks', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateNetwork(id: string, data: { name: string }) {
  return platformFetch<{ network: Network }>(`/networks/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  });
}

export async function deleteNetwork(id: string) {
  return platformFetch<{ success: boolean }>(`/networks/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export async function listSecurityGroups() {
  const res = await platformFetch<{ security_groups: SecurityGroup[] | null }>('/security-groups');
  return { security_groups: res.security_groups ?? [] };
}

export async function createSecurityGroup(data: {
  name: string;
  description?: string;
  vpc_id?: string;
  rules?: SecurityGroup['rules'];
}) {
  return platformFetch<{ security_group: SecurityGroup }>('/security-groups', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateSecurityGroup(id: string, data: {
  name?: string;
  description?: string;
  rules?: SecurityGroup['rules'];
}) {
  return platformFetch<{ security_group: SecurityGroup }>(`/security-groups/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  });
}

export async function deleteSecurityGroup(id: string) {
  return platformFetch<{ success: boolean }>(`/security-groups/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export async function listSnapshots() {
  return platformFetch<{ snapshots: Snapshot[] }>('/snapshots');
}

export async function createSnapshot(data: { volume_id: string; name: string }) {
  return platformFetch<{ snapshot: Snapshot }>('/snapshots', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function listVMSnapshots() {
  return platformFetch<{ vm_snapshots: VMSnapshot[] }>('/vm-snapshots');
}

export async function createVMSnapshot(data: { vm_name: string; name: string }) {
  return platformFetch<{ vm_snapshot: VMSnapshot }>('/vm-snapshots', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function deleteVMSnapshot(data: { name: string }) {
  return platformFetch('/vm-snapshots/delete', { method: 'POST', body: JSON.stringify(data) });
}

export async function restoreVMSnapshot(data: { name: string; vm_name: string }) {
  return platformFetch('/vm-snapshots/restore', { method: 'POST', body: JSON.stringify(data) });
}

export async function listSSHKeys() {
  const res = await platformFetch<{ ssh_keys: SSHKeyPair[] | null }>('/ssh-keys');
  return { ssh_keys: res.ssh_keys ?? [] };
}

export async function createSSHKey(data: { name: string }) {
  return platformFetch<{ key: SSHKeyPair; private_key_pem: string }>('/ssh-keys', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function registerSSHKey(data: { name: string; public_key: string }) {
  return platformFetch<{ key: SSHKeyPair }>('/ssh-keys/register', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function deleteSSHKey(id: string) {
  return platformFetch<{ success: boolean }>(`/ssh-keys/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export async function getVMSSH(name: string) {
  return platformFetch<VMSSHInfo>(`/vms/${encodeURIComponent(name)}/ssh`);
}

export async function exposeVMSSH(name: string, nodePort?: number) {
  return platformFetch<{ node_port: number }>(`/vms/${encodeURIComponent(name)}/ssh`, {
    method: 'POST',
    body: JSON.stringify(nodePort ? { node_port: nodePort } : {}),
  });
}

export interface ServiceOffering {
  id: string;
  name: string;
  display_name: string;
  cpu: number;
  memory_mi: number;
  dedicated_cpu?: boolean;
  state: string;
  storage_tags?: string;
}

export interface VMTemplate {
  id: string;
  tenant_id?: string;
  name: string;
  display_name: string;
  description?: string;
  image: string;
  source_type?: string;
  os_type?: string;
  cloud_init_user_data?: string;
  iso_volume_id?: string;
  iso_size_gi?: number;
  boot_disk_size_gi?: number;
  storage_class?: string;
  import_state?: string;
  hypervisor: string;
  state: string;
  created_at?: string;
}

export async function listServiceOfferings() {
  const res = await platformFetch<{ service_offerings: ServiceOffering[] | null }>('/service-offerings');
  return {
    service_offerings: (res.service_offerings ?? []).filter((o) => o.state === 'Active'),
  };
}

export async function listAllServiceOfferings() {
  const res = await platformFetch<{ service_offerings: ServiceOffering[] | null }>('/service-offerings?include_inactive=true');
  return { service_offerings: res.service_offerings ?? [] };
}

export async function createServiceOffering(data: {
  name: string;
  display_name?: string;
  cpu: number;
  memory_mi: number;
  dedicated_cpu?: boolean;
}) {
  return platformFetch<{ service_offering: ServiceOffering }>('/service-offerings', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateServiceOffering(id: string, data: {
  display_name?: string;
  cpu?: number;
  memory_mi?: number;
  state?: string;
  dedicated_cpu?: boolean;
}) {
  return platformFetch<{ service_offering: ServiceOffering }>(`/service-offerings/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  });
}

export async function deleteServiceOffering(id: string) {
  return platformFetch<{ success: boolean }>(`/service-offerings/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export async function listVMTemplates() {
  const res = await platformFetch<{ vm_templates: VMTemplate[] | null }>('/vm-templates');
  return { vm_templates: res.vm_templates ?? [] };
}

export async function createVMTemplate(data: {
  name: string;
  display_name?: string;
  description?: string;
  image?: string;
  source_type?: string;
  os_type?: string;
  cloud_init_user_data?: string;
  iso_volume_id?: string;
  iso_size_gi?: number;
  boot_disk_size_gi?: number;
  storage_class?: string;
}) {
  return platformFetch<{ vm_template: VMTemplate }>('/vm-templates', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

/** UI allows rename only; other fields are create-time. */
export async function updateVMTemplate(id: string, data: {
  display_name?: string;
}) {
  return platformFetch<{ vm_template: VMTemplate }>(`/vm-templates/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  });
}

export async function deleteVMTemplate(id: string) {
  return platformFetch<{ success: boolean }>(`/vm-templates/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export interface IAMUser {
  id: string;
  username: string;
  role: string;
  role_id?: string;
  tenant_id?: string;
  email?: string;
  state?: string;
}

export interface IAMRole {
  id: string;
  tenant_id?: string;
  name: string;
  description?: string;
  is_system?: boolean;
  permissions?: string[];
}

export interface IAMAPIKey {
  id: string;
  user_id: string;
  tenant_id?: string;
  name: string;
  prefix: string;
  scopes?: string[];
  expires_at?: string;
  last_used_at?: string;
  revoked_at?: string;
  created_at?: string;
}

export async function listIAMUsers() {
  const res = await platformFetch<{ users: IAMUser[] | null }>('/users');
  return { users: res.users ?? [] };
}

export async function createIAMUser(data: { username: string; password: string; email?: string; role_name?: string; role_id?: string }) {
  return platformFetch<{ user: IAMUser }>('/users', { method: 'POST', body: JSON.stringify(data) });
}

export async function deleteIAMUser(id: string) {
  return platformFetch<void>(`/users/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export async function listIAMRoles() {
  const res = await platformFetch<{ roles: IAMRole[] | null }>('/roles');
  return { roles: res.roles ?? [] };
}

export async function createIAMRole(data: { name: string; description?: string; permissions: string[] }) {
  return platformFetch<{ role: IAMRole }>('/roles', { method: 'POST', body: JSON.stringify(data) });
}

export async function deleteIAMRole(id: string) {
  return platformFetch<void>(`/roles/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export async function listIAMAPIKeys() {
  const res = await platformFetch<{ api_keys: IAMAPIKey[] | null }>('/api-keys');
  return { api_keys: res.api_keys ?? [] };
}

export async function createIAMAPIKey(data: { name: string; expires_in_days?: number; scopes?: string[] }) {
  return platformFetch<{ api_key: IAMAPIKey; secret: string }>('/api-keys', { method: 'POST', body: JSON.stringify(data) });
}

export async function revokeIAMAPIKey(id: string) {
  return platformFetch<void>(`/api-keys/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export interface DashboardSummary {
  vms: { total: number; running?: number; error?: number };
  volumes: { total: number };
  vpcs: { total: number };
  networks: { total: number };
  security_groups: { total: number };
  health: 'ok' | 'warning' | 'critical';
  recent_activity: Array<{
    type: string;
    name: string;
    display_name?: string;
    state: string;
    updated_at?: string;
    path: string;
  }>;
}

export interface SearchHit {
  type: string;
  id: string;
  name: string;
  subtitle?: string;
  path: string;
}

export interface NotificationItem {
  id: string;
  level: 'info' | 'warning' | 'error';
  title: string;
  message: string;
  path?: string;
  created_at?: string;
}

export async function getDashboardSummary() {
  return platformFetch<DashboardSummary>('/dashboard/summary');
}

export async function globalSearch(q: string) {
  return platformFetch<{ results: SearchHit[] }>(`/search?q=${encodeURIComponent(q)}`);
}

export async function listNotifications() {
  return platformFetch<{ notifications: NotificationItem[] }>('/notifications');
}
