import { useEffect, useRef, useState, useCallback, useMemo } from 'react';
import {
  ChevronDown,
  ChevronUp,
  Copy,
  Trash2,
  Pause,
  Play,
  Terminal,
  Search,
  Filter,
  ChevronRight,
  Radio,
  Check,
} from 'lucide-react';
import { cn } from '../../lib/cn';
import { IconButton } from '../ui/IconButton';
import { PopoutButton } from '../ui/PopoutButton';
import { useUIStore } from '../../stores/useUIStore';
import { useSelectedNodeData } from '../../stores/useStackStore';
import { useWindowManager } from '../../hooks/useWindowManager';
import { fetchAgentLogs, fetchGatewayLogs, type LogEntry } from '../../lib/api';
import { POLLING } from '../../lib/constants';
import type { NodeData } from '../../types';

// Log level configuration
type LogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';

const LOG_LEVELS: LogLevel[] = ['ERROR', 'WARN', 'INFO', 'DEBUG'];

const LEVEL_STYLES: Record<LogLevel, { text: string; bg: string; border: string; dot: string }> = {
  ERROR: {
    text: 'text-status-error',
    bg: 'bg-status-error/10',
    border: 'border-status-error/30',
    dot: 'bg-status-error',
  },
  WARN: {
    text: 'text-status-pending',
    bg: 'bg-status-pending/10',
    border: 'border-status-pending/30',
    dot: 'bg-status-pending',
  },
  INFO: {
    text: 'text-primary',
    bg: 'bg-primary/10',
    border: 'border-primary/30',
    dot: 'bg-primary',
  },
  DEBUG: {
    text: 'text-text-muted',
    bg: 'bg-surface-highlight',
    border: 'border-border/30',
    dot: 'bg-text-muted',
  },
};

// Parse log entry from string or JSON
interface ParsedLog {
  level: LogLevel;
  timestamp: string;
  message: string;
  component?: string;
  traceId?: string;
  attrs?: Record<string, unknown>;
  raw: string;
}

function parseLogEntry(input: string | LogEntry): ParsedLog {
  // If it's already a structured entry
  if (typeof input === 'object') {
    return {
      level: (input.level?.toUpperCase() as LogLevel) || 'INFO',
      timestamp: input.ts || '',
      message: input.msg || '',
      component: input.component,
      traceId: input.trace_id,
      attrs: input.attrs,
      raw: JSON.stringify(input, null, 2),
    };
  }

  // Try to parse as JSON
  try {
    const parsed = JSON.parse(input);
    return {
      level: (parsed.level?.toUpperCase() as LogLevel) || 'INFO',
      timestamp: parsed.ts || parsed.time || parsed.timestamp || '',
      message: parsed.msg || parsed.message || '',
      component: parsed.component || parsed.logger,
      traceId: parsed.trace_id || parsed.traceId,
      attrs: parsed,
      raw: input,
    };
  } catch {
    // Fall back to text parsing
    const level: LogLevel = input.includes('ERROR')
      ? 'ERROR'
      : input.includes('WARN')
        ? 'WARN'
        : input.includes('INFO')
          ? 'INFO'
          : 'DEBUG';

    return {
      level,
      timestamp: '',
      message: input,
      raw: input,
    };
  }
}

function formatTimestamp(ts: string): string {
  if (!ts) return '';
  try {
    const date = new Date(ts);
    return date.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    }) + '.' + String(date.getMilliseconds()).padStart(3, '0');
  } catch {
    return ts.slice(11, 23); // Fallback: extract time portion
  }
}

// Level filter dropdown component
function LevelFilter({
  enabledLevels,
  onToggle,
}: {
  enabledLevels: Set<LogLevel>;
  onToggle: (level: LogLevel) => void;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close on outside click
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const activeCount = enabledLevels.size;

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          'flex items-center gap-1.5 px-2 py-1 rounded-md text-xs',
          'border transition-all duration-200',
          activeCount < 4
            ? 'bg-primary/10 border-primary/30 text-primary'
            : 'bg-surface-elevated/60 border-border/50 text-text-muted hover:text-text-primary hover:border-text-muted/30'
        )}
      >
        <Filter size={12} />
        <span>Level</span>
        {activeCount < 4 && (
          <span className="px-1.5 py-0.5 bg-primary/20 rounded text-[10px] font-medium">
            {activeCount}
          </span>
        )}
      </button>

      {isOpen && (
        <div className="absolute top-full left-0 mt-1 z-50 min-w-[140px] py-1 rounded-lg bg-surface-elevated border border-border/50 shadow-xl backdrop-blur-xl">
          {LOG_LEVELS.map((level) => {
            const enabled = enabledLevels.has(level);
            const styles = LEVEL_STYLES[level];
            return (
              <button
                key={level}
                onClick={() => onToggle(level)}
                className={cn(
                  'w-full flex items-center gap-2 px-3 py-1.5 text-xs transition-colors',
                  'hover:bg-surface-highlight',
                  enabled ? styles.text : 'text-text-muted'
                )}
              >
                <span
                  className={cn(
                    'w-4 h-4 rounded flex items-center justify-center border',
                    enabled ? `${styles.bg} ${styles.border}` : 'border-border/50'
                  )}
                >
                  {enabled && <Check size={10} />}
                </span>
                <span className={cn('w-2 h-2 rounded-full', styles.dot)} />
                <span className="font-medium">{level}</span>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

// Expandable log line component
function LogLine({
  log,
  isExpanded,
  onToggle,
}: {
  log: ParsedLog;
  isExpanded: boolean;
  onToggle: () => void;
}) {
  const styles = LEVEL_STYLES[log.level] || LEVEL_STYLES.DEBUG;
  const hasDetails = log.attrs || log.traceId;

  return (
    <div
      className={cn(
        'group border-l-2 transition-all duration-200',
        isExpanded ? 'bg-surface-highlight/30' : 'hover:bg-surface-highlight/20',
        styles.border.replace('border-', 'border-l-')
      )}
    >
      {/* Main log line */}
      <div
        className={cn(
          'grid gap-2 px-3 py-1 cursor-pointer',
          'grid-cols-[90px_50px_80px_1fr_20px]'
        )}
        onClick={hasDetails ? onToggle : undefined}
      >
        {/* Timestamp */}
        <span className="text-text-muted font-mono text-[11px] tabular-nums">
          {formatTimestamp(log.timestamp)}
        </span>

        {/* Level badge */}
        <span
          className={cn(
            'inline-flex items-center justify-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-semibold uppercase tracking-wide',
            styles.bg,
            styles.text
          )}
        >
          <span className={cn('w-1 h-1 rounded-full', styles.dot)} />
          {log.level.slice(0, 4)}
        </span>

        {/* Component */}
        <span className="text-secondary font-mono text-[11px] truncate" title={log.component}>
          {log.component || '—'}
        </span>

        {/* Message */}
        <span
          className={cn(
            'font-mono text-[11px] truncate',
            log.level === 'ERROR' ? 'text-status-error' : 'text-text-primary'
          )}
          title={log.message}
        >
          {log.message}
        </span>

        {/* Expand indicator */}
        <span className="flex items-center justify-center">
          {hasDetails && (
            <ChevronRight
              size={12}
              className={cn(
                'text-text-muted transition-transform duration-200',
                isExpanded && 'rotate-90'
              )}
            />
          )}
        </span>
      </div>

      {/* Expanded details */}
      {isExpanded && hasDetails && (
        <div className="px-3 pb-2 ml-[90px]">
          <div className="p-2 rounded-md bg-background/60 border border-border/30 font-mono text-[10px]">
            {log.traceId && (
              <div className="flex gap-2 mb-1">
                <span className="text-text-muted">trace_id:</span>
                <span className="text-secondary">{log.traceId}</span>
              </div>
            )}
            {log.attrs && (
              <pre className="text-text-secondary whitespace-pre-wrap break-all overflow-x-auto">
                {JSON.stringify(log.attrs, null, 2)}
              </pre>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

export function BottomPanel() {
  const bottomPanelOpen = useUIStore((s) => s.bottomPanelOpen);
  const toggleBottomPanel = useUIStore((s) => s.toggleBottomPanel);
  const logsDetached = useUIStore((s) => s.logsDetached);
  const { openDetachedWindow } = useWindowManager();

  const selectedData = useSelectedNodeData() as NodeData | undefined;

  const [logs, setLogs] = useState<ParsedLog[]>([]);
  const [isPaused, setIsPaused] = useState(false);
  const [autoScroll, setAutoScroll] = useState(true);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Filter state
  const [searchQuery, setSearchQuery] = useState('');
  const [enabledLevels, setEnabledLevels] = useState<Set<LogLevel>>(
    new Set(['ERROR', 'WARN', 'INFO', 'DEBUG'])
  );
  const [expandedIndex, setExpandedIndex] = useState<number | null>(null);

  const containerRef = useRef<HTMLDivElement>(null);
  const intervalRef = useRef<number | null>(null);

  // Determine log source: gateway or agent
  const isGateway = selectedData?.type === 'gateway';
  const agentName: string | null =
    selectedData && selectedData.type !== 'gateway' ? selectedData.name : null;
  const hasSource = isGateway || agentName !== null;

  const fetchLogs = useCallback(async () => {
    try {
      if (isGateway) {
        const entries = await fetchGatewayLogs(500);
        setLogs((entries ?? []).map(parseLogEntry));
      } else if (agentName) {
        const lines = await fetchAgentLogs(agentName, 500);
        setLogs((lines ?? []).map(parseLogEntry));
      }
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch logs');
    } finally {
      setIsLoading(false);
    }
  }, [isGateway, agentName]);

  // Reset logs when source changes
  useEffect(() => {
    setLogs([]);
    setError(null);
    setExpandedIndex(null);
    setSearchQuery('');
    if (hasSource) {
      setIsLoading(true);
      fetchLogs();
    } else {
      setIsLoading(false);
    }
  }, [hasSource, isGateway, agentName, fetchLogs]);

  // Polling for logs
  useEffect(() => {
    if (!hasSource || isPaused || !bottomPanelOpen) {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
      return;
    }

    intervalRef.current = window.setInterval(fetchLogs, POLLING.LOGS);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [hasSource, isPaused, bottomPanelOpen, fetchLogs]);

  // Filter logs
  const filteredLogs = useMemo(() => {
    return (logs ?? []).filter((log) => {
      // Level filter
      if (!enabledLevels.has(log.level)) return false;

      // Search filter
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        return (
          log.message.toLowerCase().includes(query) ||
          log.component?.toLowerCase().includes(query) ||
          log.traceId?.toLowerCase().includes(query)
        );
      }

      return true;
    });
  }, [logs, enabledLevels, searchQuery]);

  // Auto-scroll to bottom
  useEffect(() => {
    if (autoScroll && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [filteredLogs, autoScroll]);

  // Detect manual scroll
  const handleScroll = () => {
    if (!containerRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
    setAutoScroll(scrollHeight - scrollTop - clientHeight < 50);
  };

  const handleClearLogs = () => {
    setLogs([]);
    setExpandedIndex(null);
  };

  const handleCopyLogs = async () => {
    const text = filteredLogs.map((log) => log.raw).join('\n');
    try {
      await navigator.clipboard.writeText(text);
    } catch {
      const textArea = document.createElement('textarea');
      textArea.value = text;
      document.body.appendChild(textArea);
      textArea.select();
      document.execCommand('copy');
      document.body.removeChild(textArea);
    }
  };

  const toggleLevel = (level: LogLevel) => {
    setEnabledLevels((prev) => {
      const next = new Set(prev);
      if (next.has(level)) {
        next.delete(level);
      } else {
        next.add(level);
      }
      return next;
    });
  };

  const sourceName = isGateway ? 'Gateway' : agentName;

  return (
    <div
      className={cn(
        'h-full w-full',
        'bg-surface/90 backdrop-blur-xl border-t border-border/50',
        'flex flex-col relative',
        'transition-all duration-300 ease-out'
      )}
    >
      {/* Top accent line */}
      <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-primary/20 to-transparent" />

      {/* Header - Always visible */}
      <div
        className="h-10 flex-shrink-0 flex items-center justify-between px-4 cursor-pointer hover:bg-surface-highlight/30 transition-colors"
        onClick={toggleBottomPanel}
      >
        <div className="flex items-center gap-3">
          <button
            onClick={(e) => {
              e.stopPropagation();
              toggleBottomPanel();
            }}
            className="p-1 rounded-md hover:bg-surface-highlight transition-colors"
          >
            {bottomPanelOpen ? (
              <ChevronDown size={14} className="text-text-muted" />
            ) : (
              <ChevronUp size={14} className="text-text-muted" />
            )}
          </button>
          <div className="flex items-center gap-2">
            <div
              className={cn(
                'p-1 rounded-md',
                isGateway ? 'bg-primary/10' : 'bg-tertiary/10'
              )}
            >
              {isGateway ? (
                <Radio size={12} className="text-primary" />
              ) : (
                <Terminal size={12} className="text-tertiary" />
              )}
            </div>
            <span className="text-xs font-medium text-text-primary">
              {sourceName ? `Logs: ${sourceName}` : 'Logs'}
            </span>
            {isGateway && (
              <span className="text-[10px] px-1.5 py-0.5 bg-primary/10 text-primary rounded font-medium border border-primary/20">
                Structured
              </span>
            )}
          </div>
          {isPaused && (
            <span className="text-[10px] px-2 py-0.5 bg-status-pending/15 text-status-pending rounded-full font-medium border border-status-pending/20">
              Paused
            </span>
          )}
          {!autoScroll && !isPaused && (
            <span className="text-[10px] px-2 py-0.5 bg-surface-highlight text-text-muted rounded-full font-medium border border-border/30">
              Scrolled
            </span>
          )}
        </div>

        <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
          {hasSource && (
            <>
              <IconButton
                icon={isPaused ? Play : Pause}
                onClick={() => setIsPaused(!isPaused)}
                tooltip={isPaused ? 'Resume' : 'Pause'}
                size="sm"
                variant="ghost"
                className={isPaused ? 'text-status-running hover:text-status-running' : ''}
              />
              <IconButton icon={Copy} onClick={handleCopyLogs} tooltip="Copy Logs" size="sm" variant="ghost" />
              <IconButton
                icon={Trash2}
                onClick={handleClearLogs}
                tooltip="Clear Logs"
                size="sm"
                variant="ghost"
                className="hover:text-status-error"
              />
              <div className="w-px h-4 bg-border/50 mx-0.5" />
            </>
          )}
          <PopoutButton
            onClick={() => openDetachedWindow('logs', agentName ? `agent=${encodeURIComponent(agentName)}` : undefined)}
            tooltip="Open in new window"
            disabled={logsDetached}
          />
        </div>
      </div>

      {/* Content - Only visible when panel is open */}
      {bottomPanelOpen && (
        <>
          {/* Filter bar */}
          {hasSource && (
            <div className="flex items-center gap-2 px-4 py-2 border-b border-border/30 bg-surface-elevated/30">
              {/* Search input */}
              <div className="relative flex-1 max-w-xs">
                <Search
                  size={12}
                  className="absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted"
                />
                <input
                  type="text"
                  placeholder="Filter logs..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className={cn(
                    'w-full pl-7 pr-3 py-1.5 text-xs font-mono',
                    'bg-background/60 border border-border/50 rounded-md',
                    'text-text-primary placeholder:text-text-muted',
                    'focus:outline-none focus:border-primary/50 focus:ring-1 focus:ring-primary/20',
                    'transition-all duration-200'
                  )}
                />
              </div>

              {/* Level filter */}
              <LevelFilter enabledLevels={enabledLevels} onToggle={toggleLevel} />

              {/* Log count */}
              <span className="text-[10px] text-text-muted font-mono ml-auto">
                {filteredLogs.length} / {(logs ?? []).length} entries
              </span>
            </div>
          )}

          {/* Log content */}
          <div
            ref={containerRef}
            onScroll={handleScroll}
            className="flex-1 overflow-auto bg-background/80 scrollbar-dark min-h-0"
          >
            {/* Empty state */}
            {!hasSource && (
              <div className="h-full flex flex-col items-center justify-center text-text-muted gap-2">
                <Terminal size={24} className="text-text-muted/50" />
                <span className="text-xs">Select a node to view logs</span>
              </div>
            )}

            {/* Loading state */}
            {hasSource && isLoading && (
              <div className="flex items-center gap-2 p-4 text-text-muted text-xs">
                <div className="w-3 h-3 border-2 border-primary border-t-transparent rounded-full animate-spin" />
                Loading logs...
              </div>
            )}

            {/* Error state */}
            {hasSource && error && (
              <div className="flex items-center gap-2 p-4 text-status-error text-xs">
                <span className="w-2 h-2 rounded-full bg-status-error" />
                Error: {error}
              </div>
            )}

            {/* No logs state */}
            {hasSource && !isLoading && !error && (filteredLogs?.length ?? 0) === 0 && (
              <div className="flex flex-col items-center justify-center h-full text-text-muted text-xs gap-2">
                <Terminal size={20} className="text-text-muted/30" />
                <span>No logs available</span>
                {searchQuery && (
                  <button
                    onClick={() => setSearchQuery('')}
                    className="text-primary hover:underline"
                  >
                    Clear filter
                  </button>
                )}
              </div>
            )}

            {/* Log entries */}
            {hasSource && !isLoading && !error && (filteredLogs?.length ?? 0) > 0 && (
              <div className="divide-y divide-border/20">
                {(filteredLogs ?? []).map((log, i) => (
                  <LogLine
                    key={i}
                    log={log}
                    isExpanded={expandedIndex === i}
                    onToggle={() => setExpandedIndex(expandedIndex === i ? null : i)}
                  />
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
