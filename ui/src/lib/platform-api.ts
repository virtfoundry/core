const API_BASE = import.meta.env.VITE_PLATFORM_API_URL || '/api/v1';

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
  const res = await fetch(`/api/v1/vms/${encodeURIComponent(name)}/logs?tail=${tail}`, { headers });
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

export async function updateVM(name: string, data: { display_name?: string; cpu?: number; memory_mi?: number }) {
  return platformFetch<{ vm: PlatformVM }>(`/vms/${encodeURIComponent(name)}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  });
}

export async function deployVM(data: {
  name: string;
  image?: string;
  cpu?: number;
  memory_mi?: number;
  network_ids?: string[];
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

export const VM_IMAGES = [
  { id: 'cirros', label: 'Cirros (demo)', image: 'quay.io/kubevirt/cirros-container-disk-demo' },
  { id: 'fedora', label: 'Fedora 39', image: 'quay.io/kubevirt/fedora-container-disk-demo' },
];

export const VM_SIZES = [
  { id: 'small', cpu: 1, memory_mi: 1024, label: 'Small (1 vCPU, 1 GB)' },
  { id: 'medium', cpu: 2, memory_mi: 4096, label: 'Medium (2 vCPU, 4 GB)' },
  { id: 'large', cpu: 4, memory_mi: 8192, label: 'Large (4 vCPU, 8 GB)' },
  { id: 'windows-large', cpu: 4, memory_mi: 16384, label: 'Windows Large (4 vCPU, 16 GB)', os: 'windows' as const },
];

export const VM_WINDOWS_IMAGES = [
  { id: 'windows-server-2022', label: 'Windows Server 2022 Eval', image: 'windows-server-2022-eval', os: 'windows' as const },
];
