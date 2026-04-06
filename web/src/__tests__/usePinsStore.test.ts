import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { usePinsStore, useDriftedServers } from '../stores/usePinsStore';

describe('useDriftedServers', () => {
  beforeEach(() => {
    usePinsStore.setState({ pins: null });
  });

  it('returns a stable reference when pins is null', () => {
    const { result, rerender } = renderHook(() => useDriftedServers());
    const first = result.current;
    rerender();
    expect(result.current).toBe(first);
  });

  it('returns an empty array when pins is null', () => {
    const { result } = renderHook(() => useDriftedServers());
    expect(result.current).toHaveLength(0);
  });

  it('returns a stable reference when pins has no drifted servers', () => {
    act(() => {
      usePinsStore.setState({
        pins: {
          'my-server': {
            status: 'pinned',
            tool_count: 3,
            server_hash: 'abc',
            pinned_at: '2026-01-01T00:00:00Z',
            last_verified_at: '2026-01-01T00:00:00Z',
            tools: {},
          },
        },
      });
    });
    const { result, rerender } = renderHook(() => useDriftedServers());
    const first = result.current;
    rerender();
    expect(result.current).toBe(first);
  });

  it('returns a stable reference when pins has drifted servers', () => {
    act(() => {
      usePinsStore.setState({
        pins: {
          'server-a': {
            status: 'drift',
            tool_count: 5,
            server_hash: 'def',
            pinned_at: '2026-01-01T00:00:00Z',
            last_verified_at: '2026-01-01T00:00:00Z',
            tools: {},
          },
        },
      });
    });
    const { result, rerender } = renderHook(() => useDriftedServers());
    const first = result.current;
    rerender();
    expect(result.current).toBe(first);
  });

  it('returns drifted servers with correct shape', () => {
    act(() => {
      usePinsStore.setState({
        pins: {
          'server-a': {
            status: 'drift',
            tool_count: 5,
            server_hash: 'def',
            pinned_at: '2026-01-01T00:00:00Z',
            last_verified_at: '2026-01-02T00:00:00Z',
            tools: {},
          },
          'server-b': {
            status: 'pinned',
            tool_count: 2,
            server_hash: 'ghi',
            pinned_at: '2026-01-01T00:00:00Z',
            last_verified_at: '2026-01-01T00:00:00Z',
            tools: {},
          },
        },
      });
    });
    const { result } = renderHook(() => useDriftedServers());
    expect(result.current).toHaveLength(1);
    expect(result.current[0].name).toBe('server-a');
    expect(result.current[0].status).toBe('drift');
    expect(result.current[0].tool_count).toBe(5);
  });

  it('updates when pins changes from null to data', () => {
    const { result } = renderHook(() => useDriftedServers());
    expect(result.current).toHaveLength(0);

    act(() => {
      usePinsStore.setState({
        pins: {
          'server-a': {
            status: 'drift',
            tool_count: 3,
            server_hash: 'abc',
            pinned_at: '2026-01-01T00:00:00Z',
            last_verified_at: '2026-01-01T00:00:00Z',
            tools: {},
          },
        },
      });
    });

    expect(result.current).toHaveLength(1);
    expect(result.current[0].name).toBe('server-a');
  });
});
