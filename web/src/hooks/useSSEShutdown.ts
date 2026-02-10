import { useEffect, useRef } from 'react';

/**
 * Lightweight SSE connection that only listens for shutdown events.
 * Does NOT replace polling — just provides early shutdown notification.
 */
export function useSSEShutdown(onShutdown: () => void) {
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    const es = new EventSource('/sse');
    eventSourceRef.current = es;

    es.addEventListener('close', () => {
      onShutdown();
    });

    es.onerror = () => {
      // SSE connection failed — this is expected when gateway is down
      // Don't take action here; polling handles disconnection
      es.close();
    };

    return () => {
      es.close();
    };
  }, [onShutdown]);
}
