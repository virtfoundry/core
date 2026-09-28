import type { PlatformVM } from './platform-api';

export function fmtMem(mi?: number) {
  if (!mi || mi <= 0) return null;
  if (mi >= 1024) return `${(mi / 1024).toFixed(mi % 1024 === 0 ? 0 : 1)} GB`;
  return `${mi} MiB`;
}

/** Prefer catalog sizing; never pretend "0 vCPU / 0" is a real offering. */
export function formatVmOffering(vm: PlatformVM): string {
  const cpu = vm.cpu > 0 ? `${vm.cpu} vCPU` : null;
  const mem = fmtMem(vm.memory_mi);
  if (cpu && mem) return `${cpu} / ${mem}`;
  if (cpu) return cpu;
  if (mem) return mem;
  if (vm.error_message) return 'sizing n/d';
  return '—';
}

export function isVmError(state?: string) {
  return (state || '').toLowerCase() === 'error';
}

export function isVmRunning(state?: string) {
  return (state || '').toLowerCase() === 'running';
}

export function isVmStopped(state?: string) {
  return (state || '').toLowerCase() === 'stopped';
}

export type DeployPhase = 'creating' | 'scheduling' | 'networking' | 'running' | 'error' | 'unknown';

/** Map Instance state (+ IP) into a coarse deploy progress phase for the UI. */
export function deployPhaseFromVm(vm?: PlatformVM | null): DeployPhase {
  if (!vm) return 'creating';
  const s = (vm.state || '').toLowerCase();
  if (s === 'error') return 'error';
  if (s === 'running') return 'running';
  if (s === 'stopped') return 'running';
  if (s === 'starting' || s === 'creating') {
    if (vm.ip) return 'networking';
    return 'scheduling';
  }
  if (s === 'stopping') return 'scheduling';
  return 'unknown';
}

export const DEPLOY_PHASE_ORDER: DeployPhase[] = [
  'creating',
  'scheduling',
  'networking',
  'running',
];

export function roleBadgeLabel(role?: string) {
  switch ((role || '').toLowerCase()) {
    case 'root':
      return 'root';
    case 'tenant_admin':
      return 'tenant-admin';
    case 'user':
      return 'user';
    default:
      return role || '—';
  }
}
