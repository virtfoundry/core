/** Dev-only: `VITE_DEMO=1 npm run dev` serves fixtures for /api/v1 so the UI renders without a backend. */
const ago = (min: number) => new Date(Date.now() - min * 60_000).toISOString();
const ns = 'virtfoundry-tenant-default';

const vm = (name: string, state: string, ip: string, cpu: number, mem: number, tpl: string, off: string, min: number, extra = {}) => ({
  id: `vm-${name}`, name, display_name: name, namespace: ns, state, power_state: state === 'stopped' ? 'Halted' : 'Running',
  cpu, memory_mi: mem, template: tpl, image: tpl, ip, service_offering_id: off, tenant_id: 'default', zone: 'homelab', host_name: 'worker-01',
  template_ref: tpl, ssh_key_refs: min > 1000 ? ['matheus-mac', 'ci-deploy'] : ['matheus-mac'],
  nics: ip ? [{ name: 'default', ip, mac: '52:54:00:ab:12:34', type: 'bridge', network_id: ip.startsWith('10.0.30') ? 'vks-lab' : ip.startsWith('10.0.2') ? 'vpc-dev-a' : 'vpc-prod-a' }] : [], created_at: ago(min), updated_at: ago(min / 2), ...extra,
});

const vms = [
  vm('web-prod-01', 'running', '10.0.1.14', 4, 8192, 'ubuntu-2204', 'medium', 17280),
  vm('web-prod-02', 'running', '10.0.1.15', 4, 8192, 'ubuntu-2204', 'medium', 17280),
  vm('db-primary', 'running', '10.0.1.21', 8, 32768, 'debian-12', 'large', 44640),
  vm('ci-runner-03', 'starting', '', 2, 4096, 'ubuntu-2404', 'small', 1),
  vm('lab-vks-worker-0', 'running', '10.0.30.12', 4, 16384, 'node-1-36-5', 'medium', 4320),
  vm('backup-old', 'stopped', '10.0.1.40', 2, 4096, 'rocky-9', 'small', 126720),
  ...Array.from({ length: 8 }, (_, i) => vm(`batch-worker-${i + 1}`, i % 3 === 0 ? 'stopped' : 'running', `10.0.2.${50 + i}`, 2, 4096, 'ubuntu-2404', 'small', 200 + i * 90)),
  vm('gpu-sandbox', 'error', '', 16, 65536, 'ubuntu-2204', 'gpu-xl', 60, { error_message: 'insufficient nvidia.com/gpu on nodes' }),
];

const vpcs = [
  { id: 'vpc-prod', name: 'vpc-prod', cidr: '10.0.0.0/16', namespace: ns, state: 'Ready' },
  { id: 'vpc-dev', name: 'vpc-dev', cidr: '10.1.0.0/16', namespace: ns, state: 'Ready' },
  { id: 'vpc-lab', name: 'vpc-lab', cidr: '10.2.0.0/16', namespace: ns, state: 'Ready' },
];
const networks = [
  { id: 'vpc-prod-a', name: 'vpc-prod-a', cidr: '10.0.1.0/24', vpc_id: 'vpc-prod', network_type: 'isolated', gateway: '10.0.1.1', state: 'Ready', created_at: ago(30000), nad_name: 'vpc-prod-a', tags: ['prod', 'web'] },
  { id: 'vpc-dev-a', name: 'vpc-dev-a', cidr: '10.0.2.0/24', vpc_id: 'vpc-dev', network_type: 'isolated', gateway: '10.0.2.1', state: 'Ready', tags: ['dev'] },
  { id: 'vks-lab', name: 'vks-lab', cidr: '10.0.30.0/24', vpc_id: 'vpc-lab', network_type: 'isolated', gateway: '10.0.30.1', state: 'Ready' },
];
const volumes = [
  { id: 'v1', name: 'pgdata-primary', size_gi: 500, state: 'Bound', namespace: ns, pvc_name: 'pgdata-primary', vm_id: 'vm-db-primary', created_at: ago(44640), tags: ['prod', 'database'] },
  { id: 'v2', name: 'uploads-prod', size_gi: 200, state: 'Bound', namespace: ns, pvc_name: 'uploads-prod', vm_id: 'vm-web-prod-01', created_at: ago(20000), tags: ['prod'] },
  { id: 'v3', name: 'archive-2025', size_gi: 1024, state: 'Bound', namespace: ns, pvc_name: 'archive-2025', created_at: ago(9000) },
];
const rule = (direction: string, port: number | undefined, cidr: string) => ({ direction, protocol: port ? 'tcp' : 'all', port_from: port, port_to: port, cidr });
const sgs = [
  { id: 'sg-web', name: 'web-public', description: 'HTTP/HTTPS', vpc_id: 'vpc-prod', created_at: ago(50000), rules: [rule('ingress', 80, '0.0.0.0/0'), rule('ingress', 443, '0.0.0.0/0'), rule('egress', undefined, '0.0.0.0/0')] },
  { id: 'sg-db', name: 'db-internal', description: 'PostgreSQL from VPC', vpc_id: 'vpc-prod', rules: [rule('ingress', 5432, '10.0.0.0/16'), rule('egress', undefined, '0.0.0.0/0')] },
];
const templates = ['ubuntu-2204', 'ubuntu-2404', 'debian-12', 'rocky-9', 'node-1-36-5'].map((n, i) => ({
  id: `t${i}`, name: n, display_name: n.replace(/-/g, ' '), image: `quay.io/containerdisks/${n}:latest`, source_type: 'container', os_type: 'linux',
  boot_disk_size_gi: 20, hypervisor: 'kubevirt', state: 'Active', created_at: ago(100000),
}));
const offerings = [['small', 1, 2048], ['medium', 4, 8192], ['large', 8, 32768], ['gpu-xl', 16, 65536]].map(([n, c, m]) => ({
  id: n, name: n, display_name: n, cpu: c, memory_mi: m, state: 'Active',
}));
const sshKeys = [
  { id: 'k1', tenant_id: 'default', name: 'matheus-mac', public_key: 'ssh-ed25519 AAAAC3Nza…demo', fingerprint: 'SHA256:9f2a…c41e', created_at: ago(57600), tags: ['personal'] },
  { id: 'k2', tenant_id: 'default', name: 'ci-deploy', public_key: 'ssh-ed25519 AAAAC3Nza…demo', fingerprint: 'SHA256:1be7…a09d', created_at: ago(31680), tags: ['ci', 'prod'] },
];
const clusters = [
  { name: 'lab-vks', tenant_id: 'default', namespace: ns, kubernetes_version: 'v1.36.5', control_plane: { service_type: 'LoadBalancer', address: '10.0.50.101', port: 443 }, workers: { count: 1, template_ref: { name: 'node-1-36-5' }, offering_ref: { name: 'medium' }, network_ref: { name: 'vks-lab' } }, phase: 'Ready', control_plane_endpoint: 'https://10.0.50.101:443', ready_workers: 1, created_at: ago(4320), conditions: [{ type: 'Ready', status: 'True' }] },
  { name: 'e2e-disposable', tenant_id: 'default', namespace: ns, kubernetes_version: 'v1.36.5', workers: { count: 1, template_ref: { name: 'node-1-36-5' }, offering_ref: { name: 'medium' }, network_ref: { name: 'vks-lab' } }, phase: 'Provisioning', ready_workers: 0, created_at: ago(5) },
];

const ROUTES: Record<string, unknown> = {
  '/auth/me': { id: 'root', username: 'root', role: 'root', tenant_id: 'default' },
  '/tenants': { tenants: ['default', 'smoke', 'lab'].map((s) => ({ id: s, name: s, slug: s, namespace: `virtfoundry-tenant-${s}`, state: 'Active', created_at: ago(129600) })) },
  '/vms': { vms }, '/vpcs': { vpcs }, '/networks': { networks }, '/volumes': { volumes }, '/security-groups': { security_groups: sgs },
  '/snapshots': { snapshots: [{ id: 's1', name: 'pgdata-0930', volume_id: 'v1', state: 'Ready', created_at: ago(23040), tags: ['prod', 'backup'] }] },
  '/vm-snapshots': { vm_snapshots: [{ id: 'vs1', name: 'db-primary-0930', vm_id: 'vm-db-primary', vm_name: 'db-primary', phase: 'Succeeded', namespace: ns, created_at: ago(23040), tags: ['before-upgrade'] }] },
  '/ssh-keys': { ssh_keys: sshKeys }, '/service-offerings': { service_offerings: offerings }, '/vm-templates': { vm_templates: templates },
  '/users': { users: [{ id: 'u1', username: 'root', role: 'root', email: 'root@virtfoundry.homelab', state: 'Active' }, { id: 'u2', username: 'matheus', role: 'tenant_admin', tenant_id: 'default', email: 'matheus@homelab', state: 'Active' }] },
  '/roles': { roles: [{ id: 'r1', name: 'admin', is_system: true, permissions: ['*'] }, { id: 'r2', name: 'viewer', is_system: true, permissions: ['read'] }] },
  '/api-keys': { api_keys: [{ id: 'a1', user_id: 'u1', name: 'terraform', prefix: 'vf_9a2c', scopes: ['read', 'write'], last_used_at: ago(120) }] },
  '/vks/clusters': { clusters },
  '/notifications': { notifications: [{ id: 'n1', level: 'error', title: 'gpu-sandbox', message: 'insufficient nvidia.com/gpu', path: '/vms/gpu-sandbox', created_at: ago(60) }] },
  '/dashboard/summary': {
    vms: { total: vms.length, running: vms.filter((v) => v.state === 'running').length, error: 1 }, volumes: { total: volumes.length }, vpcs: { total: vpcs.length },
    networks: { total: networks.length }, security_groups: { total: sgs.length }, health: 'critical',
    recent_activity: vms.slice(0, 5).map((v) => ({ type: 'vm', name: v.name, display_name: v.name, state: v.state, updated_at: v.updated_at, path: `/vms/${v.name}` })),
    hosts: {
      nodes: 3, nodes_ready: 2, cpu_capacity_millicores: 48000, cpu_allocatable_millicores: 46000, memory_capacity_bytes: 137438953472, memory_allocatable_bytes: 128849018880,
      kubelet_versions: ['v1.36.5'], os_images: ['Talos (v1.12)'], os_architectures: ['amd64'], collected_at: ago(1),
      usage: { cpu_usage_millicores: 21000, memory_usage_bytes: 83751862272, window_seconds: 60, collected_at: ago(1) },
    },
    storage: { total_bytes: 2199023255552, bound_bytes: 1374389534720, available_bytes: 824633720832, count: 14 },
    addons: { checked_at: ago(1), addons: ['kubevirt', 'cdi', 'multus', 'metrics-server'].map((name) => ({ name, status: 'ok' })) },
  },
  '/events-ticket': { ticket: 'demo' },
};

function resolve(path: string, method: string): unknown {
  if (method !== 'GET') {
    if (path === '/auth/login') return { token: 'demo-token', user: ROUTES['/auth/me'] };
    return { success: true };
  }
  const clean = path.split('?')[0];
  if (clean in ROUTES) return ROUTES[clean];
  const vmName = clean.match(/^\/vms\/([^/]+)$/)?.[1];
  if (vmName) return { vm: vms.find((v) => v.name === decodeURIComponent(vmName)) ?? vms[0] };
  const vksName = clean.match(/^\/vks\/clusters\/([^/]+)$/)?.[1];
  if (vksName) return { cluster: clusters.find((c) => c.name === vksName) ?? clusters[0] };
  if (/^\/vks\/clusters\/[^/]+\/summary$/.test(clean)) return { summary: { pod_totals: { running: 14, pending: 0, failed: 0, other: 0 }, namespaces: [{ name: 'kube-system', pod_count: 9 }, { name: 'default', pod_count: 5 }], guest_nodes: [{ name: 'lab-vks-worker-0', ready: true }] } };
  if (/^\/vms\/[^/]+\/volumes$/.test(clean)) return { volumes: volumes.slice(0, 1) };
  if (/^\/vms\/[^/]+\/ssh$/.test(clean)) return { exposed: false };
  if (clean === '/search') return { results: [] };
  return {};
}

export function installDemo() {
  const real = window.fetch.bind(window);
  localStorage.setItem('jwt_token', 'demo-token');
  localStorage.setItem('tenant_id', 'default');
  window.fetch = async (input, init) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.pathname : input.url;
    if (!url.startsWith('/api/v1')) return real(input, init);
    const body = resolve(url.slice('/api/v1'.length), (init?.method ?? 'GET').toUpperCase());
    return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } });
  };
}
