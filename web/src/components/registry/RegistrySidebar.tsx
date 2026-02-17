import { useState, useCallback } from 'react';
import {
  Library,
  Wrench,
  BarChart3,
  ChevronDown,
  ChevronRight,
  Plus,
  Pencil,
  Trash2,
  Power,
  PowerOff,
  X,
  FileCode,
} from 'lucide-react';
import { cn } from '../../lib/cn';
import { useRegistryStore } from '../../stores/useRegistryStore';
import { useStackStore } from '../../stores/useStackStore';
import { useUIStore } from '../../stores/useUIStore';
import { useWindowManager } from '../../hooks/useWindowManager';
import { PopoutButton } from '../ui/PopoutButton';
import { SkillEditor } from './SkillEditor';
import { showToast } from '../ui/Toast';
import {
  fetchRegistryStatus,
  fetchRegistrySkills,
  deleteRegistrySkill,
  activateRegistrySkill,
  disableRegistrySkill,
} from '../../lib/api';
import type { AgentSkill, ItemState, RegistryStatus } from '../../types';

type Tab = 'skills' | 'status';

const tabConfig: { key: Tab; label: string; icon: typeof Wrench }[] = [
  { key: 'skills', label: 'Skills', icon: Wrench },
  { key: 'status', label: 'Status', icon: BarChart3 },
];

export function RegistrySidebar() {
  const [activeTab, setActiveTab] = useState<Tab>('skills');
  const skills = useRegistryStore((s) => s.skills);
  const status = useRegistryStore((s) => s.status);
  const setSidebarOpen = useUIStore((s) => s.setSidebarOpen);
  const sidebarDetached = useUIStore((s) => s.sidebarDetached);
  const editorDetached = useUIStore((s) => s.editorDetached);
  const selectNode = useStackStore((s) => s.selectNode);
  const { openDetachedWindow } = useWindowManager();

  // Modal state
  const [showSkillEditor, setShowSkillEditor] = useState(false);
  const [editingSkill, setEditingSkill] = useState<AgentSkill | undefined>();

  // Delete confirmation state
  const [confirmDelete, setConfirmDelete] = useState<{ name: string } | null>(null);

  const handleClose = () => {
    setSidebarOpen(false);
    selectNode(null);
  };

  const handlePopout = () => {
    openDetachedWindow('sidebar', `node=${encodeURIComponent('Registry')}`);
  };

  const refreshRegistry = useCallback(async () => {
    try {
      const [regStatus, regSkills] = await Promise.all([
        fetchRegistryStatus(),
        fetchRegistrySkills(),
      ]);
      useRegistryStore.getState().setStatus(regStatus);
      useRegistryStore.getState().setSkills(regSkills);
    } catch {
      // Next polling cycle will pick up changes
    }
  }, []);

  const handleEditSkill = useCallback((skill: AgentSkill) => {
    setEditingSkill(skill);
    setShowSkillEditor(true);
  }, []);

  const handleDeleteConfirm = useCallback(async () => {
    if (!confirmDelete) return;
    try {
      await deleteRegistrySkill(confirmDelete.name);
      showToast('success', 'Skill deleted');
      refreshRegistry();
    } catch (err) {
      showToast('error', err instanceof Error ? err.message : 'Delete failed');
    } finally {
      setConfirmDelete(null);
    }
  }, [confirmDelete, refreshRegistry]);

  const handleToggleSkillState = useCallback(async (skill: AgentSkill) => {
    try {
      if (skill.state === 'active') {
        await disableRegistrySkill(skill.name);
        showToast('success', `Skill "${skill.name}" disabled`);
      } else {
        await activateRegistrySkill(skill.name);
        showToast('success', `Skill "${skill.name}" activated`);
      }
      refreshRegistry();
    } catch (err) {
      showToast('error', err instanceof Error ? err.message : 'State change failed');
    }
  }, [refreshRegistry]);

  return (
    <div className="h-full w-full flex flex-col overflow-hidden">
      {/* Accent line */}
      <div className="absolute top-0 left-0 bottom-0 w-px bg-gradient-to-b from-primary/40 via-primary/20 to-transparent" />

      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-border/50 bg-surface-elevated/30">
        <div className="flex items-center gap-3 min-w-0">
          <div className="p-2 rounded-xl flex-shrink-0 border bg-primary/10 border-primary/20">
            <Library size={16} className="text-primary" />
          </div>
          <div className="min-w-0">
            <h2 className="font-semibold text-text-primary truncate tracking-tight">Registry</h2>
            <div className="flex items-center gap-1.5">
              <p className="text-[10px] text-text-muted uppercase tracking-wider">Agent Skills</p>
            </div>
          </div>
        </div>
        <div className="flex items-center gap-1">
          <PopoutButton
            onClick={handlePopout}
            tooltip="Open in new window"
            disabled={sidebarDetached}
          />
          <button onClick={handleClose} className="p-1.5 rounded-lg hover:bg-surface-highlight transition-colors group">
            <X size={16} className="text-text-muted group-hover:text-text-primary transition-colors" />
          </button>
        </div>
      </div>

      {/* Tab bar */}
      <div className="flex border-b border-border/30">
        {tabConfig.map(({ key, label, icon: TabIcon }) => (
          <button
            key={key}
            onClick={() => setActiveTab(key)}
            className={cn(
              'flex-1 flex items-center justify-center gap-1.5 px-3 py-2.5 text-xs font-medium transition-colors',
              activeTab === key
                ? 'text-primary border-b-2 border-primary'
                : 'text-text-muted hover:text-text-secondary'
            )}
          >
            <TabIcon size={12} />
            {label}
          </button>
        ))}
      </div>

      {/* Item count + New button */}
      {activeTab === 'skills' && (
        <div className="flex items-center justify-between px-4 py-2 border-b border-border/20">
          <span className="text-[10px] text-text-muted">
            {(skills ?? []).length} items
          </span>
          <button
            onClick={() => setShowSkillEditor(true)}
            className="flex items-center gap-1 text-[10px] text-primary hover:text-primary/80 transition-colors"
          >
            <Plus size={10} /> New
          </button>
        </div>
      )}

      {/* Tab content */}
      <div className="flex-1 overflow-y-auto scrollbar-dark">
        {activeTab === 'skills' && (
          <SkillsTab
            skills={skills ?? []}
            onEdit={handleEditSkill}
            onDelete={(name) => setConfirmDelete({ name })}
            onToggleState={handleToggleSkillState}
          />
        )}
        {activeTab === 'status' && <StatusTab status={status} />}
      </div>

      {/* Delete confirmation */}
      {confirmDelete && (
        <div className="absolute inset-0 bg-background/80 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="glass-panel-elevated rounded-xl p-5 max-w-xs mx-4 space-y-3">
            <p className="text-sm text-text-primary">
              Delete <span className="font-mono text-primary">{confirmDelete.name}</span>?
            </p>
            <p className="text-xs text-text-muted">This action cannot be undone.</p>
            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => setConfirmDelete(null)}
                className="px-3 py-1.5 text-xs text-text-secondary hover:text-text-primary bg-surface-elevated rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleDeleteConfirm}
                className="px-3 py-1.5 text-xs font-medium rounded-lg bg-status-error text-white hover:bg-status-error/90 transition-colors"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Modals */}
      <SkillEditor
        isOpen={showSkillEditor}
        onClose={() => {
          setShowSkillEditor(false);
          setEditingSkill(undefined);
        }}
        onSaved={refreshRegistry}
        skill={editingSkill}
        popoutDisabled={editorDetached}
        onPopout={() => {
          const params = editingSkill
            ? `type=skill&name=${encodeURIComponent(editingSkill.name)}`
            : 'type=skill';
          openDetachedWindow('editor', params);
          setShowSkillEditor(false);
          setEditingSkill(undefined);
        }}
      />
    </div>
  );
}

// --- Skills Tab ---

function SkillsTab({
  skills,
  onEdit,
  onDelete,
  onToggleState,
}: {
  skills: AgentSkill[];
  onEdit: (skill: AgentSkill) => void;
  onDelete: (name: string) => void;
  onToggleState: (skill: AgentSkill) => void;
}) {
  if ((skills ?? []).length === 0) {
    return (
      <div className="p-6 text-center">
        <Wrench size={24} className="text-text-muted/30 mx-auto mb-2" />
        <p className="text-text-muted text-xs">No skills registered</p>
      </div>
    );
  }

  return (
    <div className="p-2 space-y-1">
      {(skills ?? []).map((skill) => (
        <SkillItem
          key={skill.name}
          skill={skill}
          onEdit={onEdit}
          onDelete={onDelete}
          onToggleState={onToggleState}
        />
      ))}
    </div>
  );
}

function SkillItem({
  skill,
  onEdit,
  onDelete,
  onToggleState,
}: {
  skill: AgentSkill;
  onEdit: (skill: AgentSkill) => void;
  onDelete: (name: string) => void;
  onToggleState: (skill: AgentSkill) => void;
}) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="rounded-lg bg-surface-elevated/50 border border-border-subtle overflow-hidden">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-2 p-3 hover:bg-surface-highlight/50 transition-colors"
      >
        <div className="p-0.5 text-text-muted">
          {expanded ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
        </div>
        <Wrench size={12} className="text-primary/60 flex-shrink-0" />
        <span className="text-xs font-medium text-text-primary flex-1 text-left truncate">
          {skill.name}
        </span>
        <StateBadge state={skill.state} />
        {skill.fileCount > 0 && (
          <span className="flex items-center gap-0.5 text-[10px] text-text-muted font-mono">
            <FileCode size={9} />
            {skill.fileCount}
          </span>
        )}
      </button>

      {expanded && (
        <div className="px-3 pb-3 border-t border-border-subtle">
          {skill.description && (
            <p className="text-[11px] text-text-secondary mt-2 mb-2 leading-relaxed">
              {skill.description}
            </p>
          )}

          {/* Body preview */}
          {skill.body && (
            <pre className="text-[10px] text-text-muted font-mono bg-background/60 p-2 rounded overflow-x-auto max-h-32 scrollbar-dark leading-relaxed whitespace-pre-wrap break-words">
              {skill.body}
            </pre>
          )}

          {/* Allowed tools */}
          {skill.allowedTools && (
            <div className="mt-2">
              <span className="text-[10px] text-text-muted uppercase tracking-wider block mb-1">Allowed Tools</span>
              <span className="text-[10px] font-mono text-primary">{skill.allowedTools}</span>
            </div>
          )}

          {/* Metadata */}
          {skill.metadata && Object.keys(skill.metadata).length > 0 && (
            <div className="mt-2 space-y-1">
              <span className="text-[10px] text-text-muted uppercase tracking-wider">Metadata</span>
              {Object.entries(skill.metadata).map(([key, value]) => (
                <div key={key} className="flex items-center gap-2 text-[10px]">
                  <span className="font-mono text-primary">{key}</span>
                  <span className="text-text-muted">=</span>
                  <span className="text-text-secondary truncate">{value}</span>
                </div>
              ))}
            </div>
          )}

          {/* Actions */}
          <div className="flex items-center gap-2 mt-3 pt-2 border-t border-border-subtle/50">
            <button
              onClick={(e) => {
                e.stopPropagation();
                onToggleState(skill);
              }}
              className={cn(
                'flex items-center gap-1 text-[10px] transition-colors',
                skill.state === 'active'
                  ? 'text-text-muted hover:text-status-pending'
                  : 'text-text-muted hover:text-status-running',
              )}
            >
              {skill.state === 'active' ? <PowerOff size={10} /> : <Power size={10} />}
              {skill.state === 'active' ? 'Disable' : 'Activate'}
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation();
                onEdit(skill);
              }}
              className="flex items-center gap-1 text-[10px] text-text-muted hover:text-primary transition-colors"
            >
              <Pencil size={10} /> Edit
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation();
                onDelete(skill.name);
              }}
              className="flex items-center gap-1 text-[10px] text-text-muted hover:text-status-error transition-colors"
            >
              <Trash2 size={10} /> Delete
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

// --- Status Tab ---

function StatusTab({ status }: { status: RegistryStatus | null }) {
  if (!status) {
    return (
      <div className="p-6 text-center">
        <BarChart3 size={24} className="text-text-muted/30 mx-auto mb-2" />
        <p className="text-text-muted text-xs">Loading registry status...</p>
      </div>
    );
  }

  return (
    <div className="p-4 space-y-3">
      <div className="glass-panel p-3 rounded-lg">
        <div className="flex items-center gap-2 mb-3">
          <Wrench size={12} className="text-primary/60" />
          <span className="text-[10px] text-text-muted uppercase tracking-wider font-medium">Skills</span>
        </div>
        <div className="space-y-2">
          <div className="flex justify-between items-center">
            <span className="text-xs text-text-secondary">Total</span>
            <span className="text-xs font-mono text-text-primary font-bold tabular-nums">{status.totalSkills}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-xs text-text-secondary">Active</span>
            <span className="text-xs font-mono text-status-running font-bold tabular-nums">{status.activeSkills}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

// --- Shared Components ---

function StateBadge({ state }: { state: ItemState }) {
  const styles: Record<ItemState, string> = {
    active: 'bg-status-running/10 text-status-running',
    draft: 'bg-status-pending/10 text-status-pending',
    disabled: 'bg-surface-highlight text-text-muted',
  };

  return (
    <span className={cn('text-[9px] px-1.5 py-0.5 rounded font-mono', styles[state] ?? styles.draft)}>
      {state}
    </span>
  );
}
