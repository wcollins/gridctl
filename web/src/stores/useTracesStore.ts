import { create } from 'zustand';
import { fetchTraces, fetchTraceDetail } from '../lib/api';
import type { TraceSummary, TraceDetail } from '../lib/api';

export type { TraceSummary, TraceDetail };

interface TracesFilters {
  server: string;
  errorsOnly: boolean;
  minDuration: number | null;
  search: string;
}

interface TracesState {
  traces: TraceSummary[];
  total: number;
  isLoading: boolean;
  error: string | null;
  filters: TracesFilters;

  // Selected trace detail
  selectedTraceId: string | null;
  traceDetail: TraceDetail | null;
  isLoadingDetail: boolean;
  detailError: string | null;

  // Actions
  setFilters: (filters: Partial<TracesFilters>) => void;
  selectTrace: (traceId: string | null) => void;
  loadTraces: () => Promise<void>;
  loadTraceDetail: (traceId: string) => Promise<void>;
}

export const useTracesStore = create<TracesState>()((set, get) => ({
  traces: [],
  total: 0,
  isLoading: false,
  error: null,
  filters: {
    server: '',
    errorsOnly: false,
    minDuration: null,
    search: '',
  },
  selectedTraceId: null,
  traceDetail: null,
  isLoadingDetail: false,
  detailError: null,

  setFilters: (filters) =>
    set((s) => ({ filters: { ...s.filters, ...filters } })),

  selectTrace: (traceId) => {
    set({ selectedTraceId: traceId, traceDetail: null, detailError: null });
    if (traceId) {
      get().loadTraceDetail(traceId);
    }
  },

  loadTraces: async () => {
    const { filters } = get();
    set({ isLoading: true, error: null });
    try {
      const result = await fetchTraces({
        server: filters.server || undefined,
        errors: filters.errorsOnly || undefined,
        minDuration: filters.minDuration ?? undefined,
        limit: 100,
      });
      set({ traces: result.traces, total: result.total, isLoading: false });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : 'Failed to fetch traces',
        isLoading: false,
      });
    }
  },

  loadTraceDetail: async (traceId) => {
    set({ isLoadingDetail: true, detailError: null });
    try {
      const detail = await fetchTraceDetail(traceId);
      set({ traceDetail: detail, isLoadingDetail: false });
    } catch (err) {
      set({
        detailError: err instanceof Error ? err.message : 'Failed to fetch trace detail',
        isLoadingDetail: false,
      });
    }
  },
}));
