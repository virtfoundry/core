import { useEffect, useRef, useCallback } from 'react';
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
      wsRef.current?.close();
    };
  }, [handleEvent, queryClient]);
}

export function isVMTransitional(state?: string) {
  const s = state?.toLowerCase();
  return s === 'starting' || s === 'stopping' || s === 'creating';
}
