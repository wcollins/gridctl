import { useMemo } from 'react';
import { useTracesStore } from '../stores/useTracesStore';
import { useUIStore } from '../stores/useUIStore';

/**
 * Returns a latency heat intensity (0-1) for a given server name based on its
 * average duration relative to the slowest server in recent traces. Returns 0
 * when the latency heat overlay is disabled or the server has no trace data.
 */
export function useLatencyHeat(serverName: string): number {
  const showLatencyHeat = useUIStore((s) => s.showLatencyHeat);
  const traces = useTracesStore((s) => s.traces);

  return useMemo(() => {
    if (!showLatencyHeat || !serverName) return 0;

    // Compute average duration per server from recent traces
    const serverStats = new Map<string, { total: number; count: number }>();
    for (const trace of traces) {
      const s = trace.server;
      if (!s) continue;
      const existing = serverStats.get(s) ?? { total: 0, count: 0 };
      serverStats.set(s, { total: existing.total + trace.duration, count: existing.count + 1 });
    }

    const stats = serverStats.get(serverName);
    if (!stats || stats.count === 0) return 0;

    const avgDuration = stats.total / stats.count;
    if (avgDuration === 0) return 0;

    // Find max avg across all servers
    let maxAvg = 0;
    for (const [, v] of serverStats) {
      const avg = v.total / v.count;
      if (avg > maxAvg) maxAvg = avg;
    }

    if (maxAvg === 0) return 0;
    return avgDuration / maxAvg;
  }, [showLatencyHeat, traces, serverName]);
}
