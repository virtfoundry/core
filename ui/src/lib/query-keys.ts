/** Central query keys — keep in sync with useRealtimeEvents invalidation. */
export const queryKeys = {
  tenants: ['tenants'] as const,
  dashboardSummary: ['platform-dashboard-summary'] as const,
  search: (q: string) => ['platform-search', q] as const,
  notifications: ['platform-notifications'] as const,
  vms: ['platform-vms'] as const,
  vm: (name: string) => ['platform-vm', name] as const,
  vmSnapshots: ['platform-vm-snapshots'] as const,
  volumes: ['platform-volumes'] as const,
  snapshots: ['platform-snapshots'] as const,
  vpcs: ['platform-vpcs'] as const,
  networks: ['platform-networks'] as const,
  securityGroups: ['platform-sgs'] as const,
  loadBalancers: ['platform-lbs'] as const,
  loadBalancer: (id: string) => ['platform-lbs', id] as const,
  targetGroups: ['platform-tgs'] as const,
  targetGroup: (id: string) => ['platform-tgs', id] as const,
  sshKeys: ['platform-ssh-keys'] as const,
  offerings: ['platform-offerings'] as const,
  allOfferings: ['platform-offerings', 'all'] as const,
  templates: ['platform-templates'] as const,
  iamUsers: ['platform-iam-users'] as const,
  iamRoles: ['platform-iam-roles'] as const,
  iamKeys: ['platform-iam-keys'] as const,
};

export function isPlatformQueryKey(key: unknown): boolean {
  if (!Array.isArray(key) || typeof key[0] !== 'string') return false;
  const k = key[0];
  return k.startsWith('platform-') || k === 'tenants';
}
