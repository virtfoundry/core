import type { QueryClient } from '@tanstack/react-query';
import { queryKeys } from './query-keys';

export interface PlatformEvent {
  type: string;
  payload?: Record<string, unknown>;
}

function invalidate(queryClient: QueryClient, key: readonly unknown[]) {
  void queryClient.invalidateQueries({ queryKey: key });
}

function invalidateDashboardShell(queryClient: QueryClient) {
  invalidate(queryClient, queryKeys.dashboardSummary);
  invalidate(queryClient, queryKeys.notifications);
}

function resourcePrefix(type: string): string {
  const dot = type.indexOf('.');
  return dot === -1 ? type : type.slice(0, dot);
}

/** Invalidate only the queries affected by a realtime event — no global refetch storm. */
export function invalidateForPlatformEvent(queryClient: QueryClient, event: PlatformEvent) {
  const { type, payload } = event;
  if (!type) return;

  // Ignore hub probes / heartbeats
  if (type.startsWith('subscription.') || type === 'ping' || type === 'pong') {
    return;
  }

  const name = typeof payload?.name === 'string' ? payload.name : undefined;
  const prefix = resourcePrefix(type);

  switch (prefix) {
    case 'vm': {
      invalidate(queryClient, queryKeys.vms);
      invalidate(queryClient, queryKeys.vmSnapshots);
      // Attach/detach and destroy change volume ownership shown in lists
      invalidate(queryClient, queryKeys.volumes);
      invalidateDashboardShell(queryClient);
      if (name) {
        invalidate(queryClient, queryKeys.vm(name));
        void queryClient.invalidateQueries({ queryKey: ['platform-vm-volumes', name] });
      }
      return;
    }
    case 'volume': {
      invalidate(queryClient, queryKeys.volumes);
      invalidate(queryClient, queryKeys.snapshots);
      invalidateDashboardShell(queryClient);
      return;
    }
    case 'snapshot': {
      invalidate(queryClient, queryKeys.snapshots);
      invalidate(queryClient, queryKeys.vmSnapshots);
      invalidate(queryClient, queryKeys.volumes);
      invalidateDashboardShell(queryClient);
      return;
    }
    case 'vpc': {
      invalidate(queryClient, queryKeys.vpcs);
      // Default subnet is provisioned with the VPC
      invalidate(queryClient, queryKeys.networks);
      invalidateDashboardShell(queryClient);
      return;
    }
    case 'network': {
      invalidate(queryClient, queryKeys.networks);
      invalidateDashboardShell(queryClient);
      return;
    }
    case 'security_group':
    case 'sg': {
      invalidate(queryClient, queryKeys.securityGroups);
      invalidateDashboardShell(queryClient);
      return;
    }
    case 'ssh_key':
    case 'sshkey': {
      invalidate(queryClient, queryKeys.sshKeys);
      invalidateDashboardShell(queryClient);
      return;
    }
    case 'offering':
    case 'service_offering': {
      invalidate(queryClient, queryKeys.offerings);
      invalidate(queryClient, queryKeys.allOfferings);
      invalidateDashboardShell(queryClient);
      return;
    }
    case 'template':
    case 'vm_template': {
      invalidate(queryClient, queryKeys.templates);
      invalidateDashboardShell(queryClient);
      return;
    }
    case 'iam':
    case 'user':
    case 'role':
    case 'api_key': {
      invalidate(queryClient, queryKeys.iamUsers);
      invalidate(queryClient, queryKeys.iamRoles);
      invalidate(queryClient, queryKeys.iamKeys);
      invalidate(queryClient, queryKeys.notifications);
      return;
    }
    case 'tenant': {
      invalidate(queryClient, queryKeys.tenants);
      invalidateDashboardShell(queryClient);
      return;
    }
    default: {
      // Unknown type — refresh lightweight aggregates only (no inventory storm)
      invalidateDashboardShell(queryClient);
    }
  }
}

/**
 * Slow safety net when WebSocket is down.
 * Covers inventory lists that have no aggressive page-level poll of their own.
 */
export function invalidateConnectivityFallback(queryClient: QueryClient) {
  invalidateDashboardShell(queryClient);
  invalidate(queryClient, queryKeys.vms);
  invalidate(queryClient, queryKeys.volumes);
  invalidate(queryClient, queryKeys.vpcs);
  invalidate(queryClient, queryKeys.networks);
  invalidate(queryClient, queryKeys.securityGroups);
  invalidate(queryClient, queryKeys.sshKeys);
  invalidate(queryClient, queryKeys.templates);
  invalidate(queryClient, queryKeys.offerings);
  invalidate(queryClient, queryKeys.snapshots);
  invalidate(queryClient, queryKeys.vmSnapshots);
}
