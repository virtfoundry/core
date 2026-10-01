import type { QueryClient } from '@tanstack/react-query';
import { queryKeys } from './query-keys';
import { getVM } from './platform-api';
import type { PlatformVM } from './platform-api';
import {
  applyVmEventToCache,
  mergeVmRow,
  payloadLacksSizing,
  type VMsCache,
} from './vm-event-cache';

type VMDetailCache = { vm: PlatformVM; velas_url?: string };

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

function applyVmListPatch(
  queryClient: QueryClient,
  type: string,
  payload: Record<string, unknown> | undefined,
) {
  queryClient.setQueryData<VMsCache>(queryKeys.vms, (old) =>
    applyVmEventToCache(old, type, payload),
  );
}

/** Merge-patch detail cache when present — avoid refetch on every state tick.
 * Payload may include power_state (#176); callers use effectiveVmState for UX. */
function applyVmDetailPatch(
  queryClient: QueryClient,
  name: string,
  payload: Record<string, unknown> | undefined,
) {
  if (!payload) return;
  queryClient.setQueryData<VMDetailCache>(queryKeys.vm(name), (old) => {
    if (!old?.vm) return old;
    return { ...old, vm: mergeVmRow(old.vm, payload) };
  });
}

/** Fill thin vm.created rows once via GET /vms/{name} when sizing is missing. */
function fetchVmIfCreatedThin(
  queryClient: QueryClient,
  name: string,
  payload: Record<string, unknown> | undefined,
  hadRow: boolean,
) {
  if (hadRow || !payloadLacksSizing(payload)) return;
  void queryClient
    .fetchQuery({
      queryKey: queryKeys.vm(name),
      queryFn: () => getVM(name),
    })
    .then((res) => {
      if (!res?.vm) return;
      const full = res.vm as unknown as Record<string, unknown>;
      applyVmListPatch(queryClient, 'vm.updated', full);
    })
    .catch(() => {
      // Leave thin merge-patch row; next WS event or manual refresh can recover.
    });
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
      const prev = queryClient.getQueryData<VMsCache>(queryKeys.vms);
      const hadRow = name ? (prev?.vms ?? []).some((vm) => vm.name === name) : false;

      // Happy path: merge-patch list cache — do not invalidateQueries(vms).
      applyVmListPatch(queryClient, type, payload);

      if (type === 'vm.deleted') {
        invalidate(queryClient, queryKeys.vmSnapshots);
        invalidate(queryClient, queryKeys.volumes);
        invalidateDashboardShell(queryClient);
        if (name) {
          invalidate(queryClient, queryKeys.vm(name));
          void queryClient.invalidateQueries({ queryKey: ['platform-vm-volumes', name] });
        }
        return;
      }

      if (type === 'vm.created') {
        if (name) {
          if (!hadRow && payloadLacksSizing(payload)) {
            fetchVmIfCreatedThin(queryClient, name, payload, hadRow);
          } else {
            invalidate(queryClient, queryKeys.vm(name));
          }
        }
        invalidate(queryClient, queryKeys.vmSnapshots);
        invalidateDashboardShell(queryClient);
        return;
      }

      // vm.updated (and unknown vm.*): merge state/power_state/id/name —
      // Phase 1 skips volumes invalidate (cannot distinguish attach/detach).
      // Pure state updates must not touch volumes or dashboard.
      if (name) {
        applyVmDetailPatch(queryClient, name, payload);
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
