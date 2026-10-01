import { useEffect, useRef, useCallback, useSyncExternalStore } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import {
  invalidateConnectivityFallback,
  invalidateForPlatformEvent,
  type PlatformEvent,
} from '../lib/realtime-invalidation';
import { createEventsTicket } from '../lib/platform-api';

const WS_BASE = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}`;

/** Safety poll only when WebSocket is disconnected (not a global refetch). */
const WS_DOWN_FALLBACK_MS = 45_000;

/** Aggressive transitional poll when WS is down (VMs starting/stopping). */
export const POLL_WS_DOWN_MS = 3_000;
/** Slow safety poll while transitional and WS is healthy (events should win). */
export const POLL_WS_HEALTHY_MS = 15_000;
/** Template ISO import / dashboard warning when WS healthy. */
export const POLL_WS_HEALTHY_SLOW_MS = 30_000;

type Listener = () => void;

let wsHealthy = false;
const listeners = new Set<Listener>();

function setWsHealthy(next: boolean) {
  if (wsHealthy === next) return;
  wsHealthy = next;
  listeners.forEach((l) => l());
}

function subscribeWsHealthy(listener: Listener) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function getWsHealthy() {
  return wsHealthy;
}

/** True while /ws/events is connected — list pages back off polling. */
export function useRealtimeConnected(): boolean {
  return useSyncExternalStore(subscribeWsHealthy, getWsHealthy, () => false);
}

/**
 * refetchInterval helper: aggressive when WS down, slow (or off) when healthy.
 * Pass `false` as healthyMs to fully disable poll while connected.
 */
export function realtimePollInterval(
  connected: boolean,
  needsPoll: boolean,
  opts?: { downMs?: number; healthyMs?: number | false },
): number | false {
  if (!needsPoll) return false;
  if (connected) {
    return opts?.healthyMs === undefined ? POLL_WS_HEALTHY_MS : opts.healthyMs;
  }
  return opts?.downMs ?? POLL_WS_DOWN_MS;
}

/**
 * /ws/events uses a short-lived ticket (minted via Authorization header) so the
 * session JWT never appears in the WebSocket URL (core#133).
 */
async function eventsWsUrl(): Promise<string | null> {
  try {
    const { ticket } = await createEventsTicket();
    const params = new URLSearchParams();
    params.set('ticket', ticket);
    const tenantId = localStorage.getItem('tenant_id');
    if (tenantId) params.set('tenant_id', tenantId);
    return `${WS_BASE}/ws/events?${params.toString()}`;
  } catch {
    return null;
  }
}

export function useRealtimeEvents() {
  const queryClient = useQueryClient();
  const wsRef = useRef<WebSocket | null>(null);
  const connectedRef = useRef(false);

  const handleEvent = useCallback(
    (event: PlatformEvent) => {
      if (!event.type) return;
      invalidateForPlatformEvent(queryClient, event);
    },
    [queryClient],
  );

  useEffect(() => {
    let cancelled = false;
    let reconnectTimer: ReturnType<typeof setTimeout>;
    let fallbackTimer: ReturnType<typeof setInterval>;

    async function connect() {
      const url = await eventsWsUrl();
      if (cancelled) return;
      if (!url) {
        reconnectTimer = setTimeout(connect, 3000);
        return;
      }
      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        connectedRef.current = true;
        setWsHealthy(true);
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as PlatformEvent;
          handleEvent(data);
        } catch {
          // ignore malformed messages
        }
      };

      ws.onclose = () => {
        connectedRef.current = false;
        setWsHealthy(false);
        if (!cancelled) {
          reconnectTimer = setTimeout(connect, 3000);
        }
      };

      ws.onerror = () => {
        ws.close();
      };
    }

    void connect();

    fallbackTimer = setInterval(() => {
      if (!connectedRef.current) {
        invalidateConnectivityFallback(queryClient);
      }
    }, WS_DOWN_FALLBACK_MS);

    return () => {
      cancelled = true;
      clearTimeout(reconnectTimer);
      clearInterval(fallbackTimer);
      connectedRef.current = false;
      setWsHealthy(false);
      wsRef.current?.close();
    };
  }, [handleEvent, queryClient]);
}

/**
 * Transitional observed states (optimistic or phase). For Start/Stop lag under
 * operatorReconcile, pass `effectiveVmState(vm)` so Running+Halted (desired)
 * from the Instance informer counts as Stopping without list refetch.
 */
export function isVMTransitional(state?: string) {
  const s = state?.toLowerCase();
  return s === 'starting' || s === 'stopping' || s === 'creating';
}
