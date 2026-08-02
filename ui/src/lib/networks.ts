import type { Network } from './platform-api';

/** Platform-wide public/shared network (matches backend SharedNetworkID). */
export const PUBLIC_NETWORK_ID = '00000000-0000-4000-8000-000000000001';

export function isPublicNetwork(net: Network): boolean {
  if (net.network_type === 'shared') return true;
  if (net.id === PUBLIC_NETWORK_ID) return true;
  if (net.name === 'public' && !net.vpc_id) return true;
  return false;
}

export function isIsolatedNetwork(net: Network): boolean {
  return !isPublicNetwork(net);
}

export function partitionNetworks(networks: Network[]) {
  const publicNetworks = networks.filter(isPublicNetwork);
  const isolatedNetworks = networks.filter(isIsolatedNetwork);
  return { publicNetworks, isolatedNetworks };
}
