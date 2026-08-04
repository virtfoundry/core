import { useEffect, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import {
  invalidateConnectivityFallback,
  invalidateForPlatformEvent,
  type PlatformEvent,
} from '../lib/realtime-invalidation';

const WS_BASE = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}`;

/** Safety poll only when WebSocket is disconnected (not a global refetch). */
const WS_DOWN_FALLBACK_MS = 45_000;

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
    let reconnectTimer: ReturnType<typeof setTimeout>;
    let fallbackTimer: ReturnType<typeof setInterval>;

    function connect() {
      const ws = new WebSocket(`${WS_BASE}/ws/events`);
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
        reconnectTimer = setTimeout(connect, 3000);
      };

      ws.onerror = () => {
        ws.close();
      };
    }

    connect();

    fallbackTimer = setInterval(() => {
      if (!connectedRef.current) {
        invalidateConnectivityFallback(queryClient);
      }
    }, WS_DOWN_FALLBACK_MS);

    return () => {
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
