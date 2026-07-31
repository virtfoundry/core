/** Central query keys — keep in sync with useRealtimeEvents invalidation. */
export const queryKeys = {
  tenants: ['tenants'] as const,
  vms: ['platform-vms'] as const,
  vm: (name: string) => ['platform-vm', name] as const,
  vmSnapshots: ['platform-vm-snapshots'] as const,
  volumes: ['platform-volumes'] as const,
  snapshots: ['platform-snapshots'] as const,
  vpcs: ['platform-vpcs'] as const,
  networks: ['platform-networks'] as const,
  securityGroups: ['platform-sgs'] as const,
  sshKeys: ['platform-ssh-keys'] as const,
  offerings: ['platform-offerings'] as const,
  templates: ['platform-templates'] as const,
};

export function isPlatformQueryKey(key: unknown): boolean {
  if (!Array.isArray(key) || typeof key[0] !== 'string') return false;
  const k = key[0];
  return k.startsWith('platform-') || k === 'tenants';
}
