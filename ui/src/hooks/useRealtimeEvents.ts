import { useEffect, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { isPlatformQueryKey } from '../lib/query-keys';

const WS_BASE = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}`;

/** Poll interval fallback when WebSocket is disconnected or misses events. */
const FALLBACK_POLL_MS = 12_000;

export function useRealtimeEvents() {
  const queryClient = useQueryClient();
  const wsRef = useRef<WebSocket | null>(null);
  const connectedRef = useRef(false);

  const refreshPlatform = useCallback(() => {
    queryClient.refetchQueries({
      predicate: (q) => isPlatformQueryKey(q.queryKey),
      type: 'active',
    });
  }, [queryClient]);

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
          const data = JSON.parse(event.data);
          if (data.type) {
            refreshPlatform();
          }
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

    // Fallback polling keeps UI fresh even if WS drops events
    fallbackTimer = setInterval(() => {
      refreshPlatform();
    }, FALLBACK_POLL_MS);

    return () => {
      clearTimeout(reconnectTimer);
      clearInterval(fallbackTimer);
      wsRef.current?.close();
    };
  }, [refreshPlatform]);
}

export function isVMTransitional(state?: string) {
  const s = state?.toLowerCase();
  return s === 'starting' || s === 'stopping' || s === 'creating';
}
