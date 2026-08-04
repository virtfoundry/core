import type { QueryClient } from '@tanstack/react-query';
import { queryKeys } from './query-keys';

export interface PlatformEvent {
  type: string;
  payload?: Record<string, unknown>;
}

/** Invalidate only the queries affected by a realtime event — no global refetch storm. */
export function invalidateForPlatformEvent(queryClient: QueryClient, event: PlatformEvent) {
  const { type, payload } = event;
  const name = typeof payload?.name === 'string' ? payload.name : undefined;

  if (type.startsWith('vm.')) {
    void queryClient.invalidateQueries({ queryKey: queryKeys.vms });
    void queryClient.invalidateQueries({ queryKey: queryKeys.dashboardSummary });
    void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
    void queryClient.invalidateQueries({ queryKey: queryKeys.vmSnapshots });
    if (name) {
      void queryClient.invalidateQueries({ queryKey: queryKeys.vm(name) });
    }
    return;
  }

  // Fallback for unknown event types — refresh lightweight aggregates only
  void queryClient.invalidateQueries({ queryKey: queryKeys.dashboardSummary });
  void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
}

/** Slow safety net when WebSocket is down — dashboard/notifications only. */
export function invalidateConnectivityFallback(queryClient: QueryClient) {
  void queryClient.invalidateQueries({ queryKey: queryKeys.dashboardSummary });
  void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
}
