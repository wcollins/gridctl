import { useMemo } from 'react';
import { create } from 'zustand';
import { subscribeWithSelector } from 'zustand/middleware';
import type { ServerPins } from '../lib/api';

interface PinsState {
  pins: Record<string, ServerPins> | null;
  setPins: (pins: Record<string, ServerPins>) => void;
}

export const usePinsStore = create<PinsState>()(
  subscribeWithSelector((set) => ({
    pins: null,
    setPins: (pins) => set({ pins }),
  }))
);

// Stable empty reference — avoids new array allocation when pins is null,
// satisfying useSyncExternalStore's requirement for referential stability.
const EMPTY_DRIFTED: Array<{ name: string } & ServerPins> = [];

export const useDriftedServers = () => {
  const pins = usePinsStore((s) => s.pins);
  return useMemo(() => {
    if (!pins) return EMPTY_DRIFTED;
    return Object.entries(pins)
      .filter(([, sp]) => sp.status === 'drift')
      .map(([name, sp]) => ({ name, ...sp }));
  }, [pins]);
};
