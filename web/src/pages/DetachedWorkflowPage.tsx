import { Component, type ReactNode } from 'react';
import { useSearchParams } from 'react-router-dom';
import { AlertCircle, GitBranch } from 'lucide-react';
import { ReactFlowProvider } from '@xyflow/react';
import { WorkflowGraph } from '../components/workflow/WorkflowGraph';
import { WorkflowInspector } from '../components/workflow/WorkflowInspector';
import { WorkflowRunner } from '../components/workflow/WorkflowRunner';
import { useWorkflowStore } from '../stores/useWorkflowStore';
import { useDetachedWindowSync } from '../hooks/useBroadcastChannel';
import { useEffect } from 'react';

// Error boundary
interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

class DetachedErrorBoundary extends Component<{ children: ReactNode }, ErrorBoundaryState> {
  constructor(props: { children: ReactNode }) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error };
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="h-screen w-screen bg-background flex items-center justify-center">
          <div className="text-center p-8 max-w-md">
            <div className="p-4 rounded-xl bg-status-error/10 border border-status-error/20 inline-block mb-4">
              <AlertCircle size={32} className="text-status-error" />
            </div>
            <h1 className="text-lg text-status-error mb-2">Something went wrong</h1>
            <pre className="text-xs text-text-muted bg-surface p-4 rounded-lg overflow-auto max-h-32 mb-4">
              {this.state.error?.message}
            </pre>
            <button
              onClick={() => window.location.reload()}
              className="px-4 py-2 bg-primary text-background rounded-lg font-medium hover:bg-primary/90 transition-colors"
            >
              Reload Window
            </button>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}

function DetachedWorkflowContent() {
  const [searchParams] = useSearchParams();
  const skillName = searchParams.get('skill');

  const loading = useWorkflowStore((s) => s.loading);
  const error = useWorkflowStore((s) => s.error);
  const selectedStepId = useWorkflowStore((s) => s.selectedStepId);
  const loadWorkflow = useWorkflowStore((s) => s.loadWorkflow);

  // Register with main window
  useDetachedWindowSync('workflow');

  // Load workflow data
  useEffect(() => {
    if (skillName) {
      loadWorkflow(skillName);
    }
  }, [skillName, loadWorkflow]);

  if (!skillName) {
    return (
      <div className="h-screen w-screen bg-background flex items-center justify-center">
        <div className="text-center text-text-muted">
          <AlertCircle size={32} className="mx-auto mb-3 opacity-50" />
          <p className="text-sm">No skill specified</p>
        </div>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="h-screen w-screen bg-background flex items-center justify-center">
        <div className="text-center space-y-4">
          <div className="w-10 h-10 mx-auto border-2 border-primary border-t-transparent rounded-full animate-spin" />
          <p className="text-sm text-text-muted">Loading workflow...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="h-screen w-screen bg-background flex items-center justify-center">
        <div className="text-center p-8 max-w-md">
          <AlertCircle size={32} className="mx-auto mb-3 text-status-error" />
          <p className="text-sm text-status-error mb-4">{error}</p>
          <button
            onClick={() => loadWorkflow(skillName)}
            className="px-4 py-2 bg-primary text-background rounded-lg font-medium hover:bg-primary/90 transition-colors"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="h-screen w-screen bg-background overflow-hidden flex flex-col">
      {/* Background grain */}
      <div
        className="fixed inset-0 pointer-events-none z-0 opacity-[0.015]"
        style={{
          backgroundImage: `url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noise'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noise)'/%3E%3C/svg%3E")`,
        }}
      />

      {/* Main content */}
      <div className="flex-1 flex flex-col min-h-0 relative z-10">
        <div className="flex-1 flex min-h-0">
          {/* Graph */}
          <div className="flex-1 min-w-0 min-h-0">
            <ReactFlowProvider>
              <WorkflowGraph />
            </ReactFlowProvider>
          </div>

          {/* Inspector */}
          {selectedStepId && <WorkflowInspector />}
        </div>

        {/* Runner */}
        <div className="max-h-[40%] min-h-[120px] overflow-hidden flex flex-col">
          <WorkflowRunner />
        </div>
      </div>

      {/* Status bar */}
      <div className="h-6 bg-surface/90 backdrop-blur-xl border-t border-border/50 flex items-center justify-between px-4 text-[10px] text-text-muted z-20 flex-shrink-0">
        <span className="flex items-center gap-1.5">
          <GitBranch size={10} className="text-primary/60" />
          Workflow: {skillName}
        </span>
        <span className="flex items-center gap-1">
          <span className="w-1.5 h-1.5 rounded-full bg-status-running animate-pulse" />
          Detached Workflow
        </span>
      </div>
    </div>
  );
}

export function DetachedWorkflowPage() {
  return (
    <DetachedErrorBoundary>
      <DetachedWorkflowContent />
    </DetachedErrorBoundary>
  );
}
