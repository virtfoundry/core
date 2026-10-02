/** Map VKSCluster.phase → StatusBadge status (GKE-like lifecycle). */
export function vksPhaseBadgeStatus(phase?: string): string {
  switch ((phase || '').toLowerCase()) {
    case 'ready':
      return 'running';
    case 'failed':
      return 'error';
    case 'deleting':
      return 'stopping';
    case 'controlplaneready':
      return 'starting';
    case 'provisioning':
    case 'pending':
      return 'creating';
    default:
      return phase || 'inactive';
  }
}

export function vksCanDownloadKubeconfig(phase?: string): boolean {
  return phase === 'Ready' || phase === 'ControlPlaneReady';
}

export function vksNodesLabel(ready?: number, desired?: number): string {
  const r = ready ?? 0;
  const d = desired ?? 0;
  return `${r} / ${d}`;
}

export function isVKSWorkerVM(clusterName: string, vmName: string): boolean {
  return vmName.startsWith(`${clusterName}-worker-`);
}
