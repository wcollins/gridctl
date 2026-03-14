import { useState, useCallback, useMemo } from 'react';
import {
  Bot,
  Container,
  Cpu,
  ChevronDown,
  ChevronRight,
  Plus,
  X,
  Zap,
  AlertCircle,
  KeyRound,
  Server,
  Check,
  GitBranch,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { cn } from '../../../lib/cn';
import type { AgentFormData } from '../../../lib/yaml-builder';
import { SecretsPopover } from '../SecretsPopover';
import { useStackStore } from '../../../stores/useStackStore';

// --- Agent type definitions ---

interface AgentTypeOption {
  type: 'container' | 'headless';
  icon: LucideIcon;
  label: string;
  description: string;
}

const AGENT_TYPES: AgentTypeOption[] = [
  {
    type: 'container',
    icon: Container,
    label: 'Container',
    description: 'Docker image or source-built agent',
  },
  {
    type: 'headless',
    icon: Cpu,
    label: 'Headless Runtime',
    description: 'Lightweight agent with runtime and prompt',
  },
];

const RUNTIME_OPTIONS = [
  { value: 'node', label: 'Node.js' },
  { value: 'python', label: 'Python' },
  { value: 'deno', label: 'Deno' },
  { value: 'bun', label: 'Bun' },
];

// --- Kebab-case validation ---

function toKebabCase(value: string): string {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9-]/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '');
}

// --- Accordion section ---

function Section({
  title,
  icon: Icon,
  expanded,
  onToggle,
  children,
  badge,
}: {
  title: string;
  icon: LucideIcon;
  expanded: boolean;
  onToggle: () => void;
  children: React.ReactNode;
  badge?: string;
}) {
  return (
    <div className="border border-border/20 rounded-xl overflow-hidden transition-colors hover:border-border/30">
      <button
        type="button"
        onClick={onToggle}
        className="w-full flex items-center gap-2.5 px-4 py-3 text-left hover:bg-white/[0.02] transition-colors"
      >
        {expanded ? (
          <ChevronDown size={14} className="text-text-muted flex-shrink-0" />
        ) : (
          <ChevronRight size={14} className="text-text-muted flex-shrink-0" />
        )}
        <Icon size={14} className="text-tertiary flex-shrink-0" />
        <span className="text-xs font-medium text-text-primary">{title}</span>
        {badge && (
          <span className="ml-auto text-[10px] text-text-muted bg-surface-highlight px-1.5 py-0.5 rounded-full">
            {badge}
          </span>
        )}
      </button>
      {expanded && (
        <div className="px-4 pb-4 pt-1 space-y-3 animate-fade-in-up" style={{ animationDuration: '200ms' }}>
          {children}
        </div>
      )}
    </div>
  );
}

// --- Form field components ---

const inputClass = 'w-full bg-background/60 border border-border/40 rounded-lg px-3 py-2 text-xs focus:outline-none focus:border-primary/50 text-text-primary placeholder:text-text-muted/50 transition-colors';
const labelClass = 'block text-xs text-text-secondary mb-1.5';

function FieldError({ error }: { error?: string }) {
  if (!error) return null;
  return (
    <p className="text-[10px] text-status-error mt-1">
      <AlertCircle size={10} className="inline mr-1 -mt-0.5" />
      {error}
    </p>
  );
}

// --- Key-Value Editor ---

function KeyValueEditor({
  label,
  value,
  onChange,
  placeholder,
  showSecrets = false,
}: {
  label: string;
  value: Record<string, string>;
  onChange: (val: Record<string, string>) => void;
  placeholder?: { key: string; value: string };
  showSecrets?: boolean;
}) {
  const entries = Object.entries(value);

  const addEntry = () => onChange({ ...value, '': '' });

  const updateKey = (_oldKey: string, newKey: string, idx: number) => {
    const newVal: Record<string, string> = {};
    Object.entries(value).forEach(([k, v], i) => {
      newVal[i === idx ? newKey : k] = v;
    });
    onChange(newVal);
  };

  const updateValue = (key: string, newValue: string) => {
    onChange({ ...value, [key]: newValue });
  };

  const removeEntry = (key: string) => {
    const next = { ...value };
    delete next[key];
    onChange(next);
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-1.5">
        <label className="text-xs text-text-secondary">{label}</label>
        <button
          type="button"
          onClick={addEntry}
          className="flex items-center gap-1 text-[10px] text-secondary hover:text-secondary-light transition-colors"
        >
          <Plus size={10} />
          Add
        </button>
      </div>
      {entries.length === 0 && (
        <p className="text-[10px] text-text-muted/60 italic py-2">No entries</p>
      )}
      <div className="space-y-1.5">
        {entries.map(([key, val], i) => (
          <div key={i} className="flex items-center gap-1.5">
            <input
              type="text"
              value={key}
              onChange={(e) => updateKey(key, e.target.value, i)}
              placeholder={placeholder?.key ?? 'KEY'}
              className={cn(inputClass, 'w-[40%] font-mono')}
            />
            <span className="text-text-muted text-xs">=</span>
            <div className="flex-1 flex items-center gap-0.5">
              <input
                type="text"
                value={val}
                onChange={(e) => updateValue(key, e.target.value)}
                placeholder={placeholder?.value ?? 'value'}
                className={cn(inputClass, 'flex-1', val.startsWith('${vault:') && 'text-tertiary font-medium')}
              />
              {showSecrets && (
                <SecretsPopover onSelect={(ref) => updateValue(key, ref)} />
              )}
            </div>
            <button
              type="button"
              onClick={() => removeEntry(key)}
              className="p-1 text-text-muted hover:text-status-error transition-colors flex-shrink-0"
            >
              <X size={12} />
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}

// --- Command Array Builder ---

function CommandArrayBuilder({
  value,
  onChange,
}: {
  value: string[];
  onChange: (val: string[]) => void;
}) {
  const addItem = () => onChange([...value, '']);
  const updateItem = (idx: number, val: string) => {
    const next = [...value];
    next[idx] = val;
    onChange(next);
  };
  const removeItem = (idx: number) => {
    onChange(value.filter((_, i) => i !== idx));
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-1.5">
        <label className={labelClass}>Command</label>
        <button
          type="button"
          onClick={addItem}
          className="flex items-center gap-1 text-[10px] text-secondary hover:text-secondary-light transition-colors"
        >
          <Plus size={10} />
          Add argument
        </button>
      </div>
      {value.length === 0 && (
        <p className="text-[10px] text-text-muted/60 italic py-2">No command specified</p>
      )}
      <div className="space-y-1.5">
        {value.map((item, i) => (
          <div key={i} className="flex items-center gap-1.5">
            <span className="text-[10px] text-text-muted w-4 text-right flex-shrink-0">{i === 0 ? '$' : ''}</span>
            <input
              type="text"
              value={item}
              onChange={(e) => updateItem(i, e.target.value)}
              placeholder={i === 0 ? 'command' : `arg ${i}`}
              className={cn(inputClass, 'flex-1 font-mono')}
            />
            <button
              type="button"
              onClick={() => removeItem(i)}
              className="p-1 text-text-muted hover:text-status-error transition-colors flex-shrink-0"
            >
              <X size={12} />
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}

// --- Main AgentForm ---

interface AgentFormProps {
  data: AgentFormData;
  onChange: (data: Partial<AgentFormData>) => void;
  errors?: Record<string, string>;
}

export function AgentForm({ data, onChange, errors }: AgentFormProps) {
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    new Set(['identity', 'type', 'config']),
  );

  const toggleSection = useCallback((section: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev);
      if (next.has(section)) next.delete(section);
      else next.add(section);
      return next;
    });
  }, []);

  const isHeadless = data.agentType === 'headless';
  const typeOption = AGENT_TYPES.find((t) => t.type === data.agentType)!;

  // Get available MCP servers from stack store for "uses" selection
  const mcpServers = useStackStore((s) => s.mcpServers) ?? [];
  const availableServers = useMemo(
    () => mcpServers.map((s) => s.name).filter(Boolean),
    [mcpServers],
  );

  const envCount = data.env ? Object.keys(data.env).length : 0;
  const usesCount = data.uses?.length ?? 0;
  const a2aSkillCount = data.a2a?.skills?.length ?? 0;

  return (
    <div className="space-y-3">
      {/* Section 1: Identity */}
      <Section
        title="Identity"
        icon={Bot}
        expanded={expandedSections.has('identity')}
        onToggle={() => toggleSection('identity')}
      >
        <div>
          <label className={labelClass}>
            Name <span className="text-status-error">*</span>
          </label>
          <input
            type="text"
            value={data.name}
            onChange={(e) => onChange({ name: toKebabCase(e.target.value) })}
            placeholder="my-agent"
            className={cn(inputClass, 'font-mono', errors?.name && 'border-status-error/50')}
          />
          <FieldError error={errors?.name} />
          <p className="text-[10px] text-text-muted mt-1">Kebab-case identifier for this agent</p>
        </div>
      </Section>

      {/* Section 2: Agent Type */}
      <Section
        title="Agent Type"
        icon={Cpu}
        expanded={expandedSections.has('type')}
        onToggle={() => toggleSection('type')}
        badge={typeOption.label}
      >
        <div className="grid grid-cols-2 gap-2">
          {AGENT_TYPES.map((opt) => {
            const Icon = opt.icon;
            const isSelected = data.agentType === opt.type;
            return (
              <button
                key={opt.type}
                type="button"
                onClick={() => onChange({ agentType: opt.type })}
                className={cn(
                  'group flex flex-col items-center text-center p-4 rounded-xl border transition-all duration-200',
                  'bg-white/[0.02] hover:bg-white/[0.05]',
                  isSelected
                    ? 'border-tertiary/50 bg-tertiary/[0.06] shadow-[0_0_16px_rgba(139,92,246,0.08)]'
                    : 'border-white/[0.06] hover:border-white/[0.1]',
                )}
              >
                <Icon
                  size={18}
                  className={cn(
                    'mb-2 transition-colors',
                    isSelected ? 'text-tertiary' : 'text-text-muted group-hover:text-text-secondary',
                  )}
                />
                <span className={cn('text-[11px] font-medium', isSelected ? 'text-tertiary' : 'text-text-primary')}>
                  {opt.label}
                </span>
                <span className="text-[9px] text-text-muted leading-tight mt-0.5">
                  {opt.description}
                </span>
              </button>
            );
          })}
        </div>
      </Section>

      {/* Section 3: Type-Specific Configuration */}
      <Section
        title="Configuration"
        icon={typeOption.icon}
        expanded={expandedSections.has('config')}
        onToggle={() => toggleSection('config')}
      >
        <div className="space-y-3 transition-all duration-200">
          {/* Container fields */}
          {!isHeadless && (
            <>
              <div>
                <label className={labelClass}>Image</label>
                <input
                  type="text"
                  value={data.image ?? ''}
                  onChange={(e) => onChange({ image: e.target.value })}
                  placeholder="my-agent:latest"
                  className={cn(inputClass, 'font-mono', errors?.image && 'border-status-error/50')}
                />
                <FieldError error={errors?.image} />
              </div>

              {/* Source config (alternative to image) */}
              <div className="space-y-3 p-3 rounded-xl bg-white/[0.02] border border-white/[0.04]">
                <div className="flex items-center gap-1.5 mb-1">
                  <GitBranch size={11} className="text-text-muted" />
                  <span className="text-[10px] text-text-muted uppercase tracking-wider font-medium">
                    Or build from source
                  </span>
                </div>
                <div>
                  <label className={labelClass}>Source Type</label>
                  <div className="flex gap-2">
                    {['git', 'local'].map((t) => (
                      <button
                        key={t}
                        type="button"
                        onClick={() =>
                          onChange({
                            source: {
                              ...data.source,
                              type: t,
                              url: t === 'git' ? (data.source?.url ?? '') : undefined,
                              path: t === 'local' ? (data.source?.path ?? '') : undefined,
                            } as AgentFormData['source'],
                          })
                        }
                        className={cn(
                          'px-3 py-1.5 rounded-lg text-xs font-medium transition-all duration-200 border',
                          (data.source?.type ?? 'git') === t
                            ? 'bg-tertiary/10 border-tertiary/30 text-tertiary'
                            : 'bg-white/[0.02] border-white/[0.06] text-text-muted hover:text-text-secondary',
                        )}
                      >
                        {t === 'git' ? 'Git' : 'Local'}
                      </button>
                    ))}
                    {data.source && (
                      <button
                        type="button"
                        onClick={() => onChange({ source: undefined })}
                        className="px-2 py-1 text-[10px] text-text-muted hover:text-text-secondary"
                      >
                        Clear
                      </button>
                    )}
                  </div>
                </div>
                {data.source && (data.source.type ?? 'git') === 'git' && (
                  <div className="grid grid-cols-2 gap-2">
                    <div className="col-span-2">
                      <label className={labelClass}>Repository URL</label>
                      <input
                        type="url"
                        value={data.source?.url ?? ''}
                        onChange={(e) =>
                          onChange({ source: { ...data.source, type: 'git', url: e.target.value } as AgentFormData['source'] })
                        }
                        placeholder="https://github.com/org/repo.git"
                        className={cn(inputClass, errors?.['source.url'] && 'border-status-error/50')}
                      />
                      <FieldError error={errors?.['source.url']} />
                    </div>
                    <div>
                      <label className={labelClass}>Ref</label>
                      <input
                        type="text"
                        value={data.source?.ref ?? ''}
                        onChange={(e) =>
                          onChange({ source: { ...data.source, type: 'git', ref: e.target.value } as AgentFormData['source'] })
                        }
                        placeholder="main"
                        className={cn(inputClass, 'font-mono')}
                      />
                    </div>
                    <div>
                      <label className={labelClass}>Path</label>
                      <input
                        type="text"
                        value={data.source?.path ?? ''}
                        onChange={(e) =>
                          onChange({ source: { ...data.source, type: 'git', path: e.target.value } as AgentFormData['source'] })
                        }
                        placeholder="."
                        className={cn(inputClass, 'font-mono')}
                      />
                    </div>
                  </div>
                )}
                {data.source && data.source.type === 'local' && (
                  <div>
                    <label className={labelClass}>Local Path</label>
                    <input
                      type="text"
                      value={data.source?.path ?? ''}
                      onChange={(e) =>
                        onChange({ source: { ...data.source, type: 'local', path: e.target.value } as AgentFormData['source'] })
                      }
                      placeholder="./path/to/agent"
                      className={cn(inputClass, 'font-mono', errors?.['source.path'] && 'border-status-error/50')}
                    />
                    <FieldError error={errors?.['source.path']} />
                  </div>
                )}
              </div>

              {/* Command */}
              <CommandArrayBuilder
                value={data.command ?? []}
                onChange={(command) => onChange({ command })}
              />
            </>
          )}

          {/* Headless fields */}
          {isHeadless && (
            <>
              <div>
                <label className={labelClass}>
                  Runtime <span className="text-status-error">*</span>
                </label>
                <select
                  value={data.runtime ?? ''}
                  onChange={(e) => onChange({ runtime: e.target.value || undefined })}
                  className={cn(inputClass, errors?.runtime && 'border-status-error/50')}
                >
                  <option value="">Select runtime</option>
                  {RUNTIME_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>{opt.label}</option>
                  ))}
                </select>
                <FieldError error={errors?.runtime} />
              </div>
              <div>
                <label className={labelClass}>
                  Prompt <span className="text-status-error">*</span>
                </label>
                <textarea
                  value={data.prompt ?? ''}
                  onChange={(e) => onChange({ prompt: e.target.value })}
                  placeholder="Agent system prompt..."
                  rows={4}
                  className={cn(inputClass, 'resize-none', errors?.prompt && 'border-status-error/50')}
                />
                <FieldError error={errors?.prompt} />
                <p className="text-[10px] text-text-muted mt-1">System prompt defining agent behavior</p>
              </div>
            </>
          )}
        </div>
      </Section>

      {/* Section 4: Tool Access (MCP Servers) */}
      <Section
        title="Tool Access"
        icon={Server}
        expanded={expandedSections.has('uses')}
        onToggle={() => toggleSection('uses')}
        badge={usesCount > 0 ? `${usesCount}` : undefined}
      >
        {availableServers.length === 0 ? (
          <p className="text-[10px] text-text-muted/60 italic py-2">
            No MCP servers available. Deploy servers first or add them in a Stack.
          </p>
        ) : (
          <div className="space-y-1">
            {availableServers.map((serverName) => {
              const useEntry = data.uses?.find((u) => u.server === serverName);
              const isUsed = !!useEntry;
              return (
                <div key={serverName}>
                  <button
                    type="button"
                    onClick={() => {
                      const currentUses = data.uses ?? [];
                      if (isUsed) {
                        onChange({ uses: currentUses.filter((u) => u.server !== serverName) });
                      } else {
                        onChange({ uses: [...currentUses, { server: serverName }] });
                      }
                    }}
                    className={cn(
                      'w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs transition-all duration-200 border',
                      isUsed
                        ? 'bg-primary/[0.06] border-primary/20 text-primary'
                        : 'bg-white/[0.02] border-white/[0.04] text-text-muted hover:text-text-secondary hover:border-white/[0.08]',
                    )}
                  >
                    <div className={cn(
                      'w-3.5 h-3.5 rounded border flex items-center justify-center transition-colors',
                      isUsed ? 'bg-primary/20 border-primary/40' : 'border-border/40',
                    )}>
                      {isUsed && <Check size={9} className="text-primary" />}
                    </div>
                    <Server size={11} className="flex-shrink-0" />
                    <span className="font-mono truncate">{serverName}</span>
                  </button>
                </div>
              );
            })}
          </div>
        )}
      </Section>

      {/* Section 5: Environment & Secrets */}
      <Section
        title="Environment & Secrets"
        icon={KeyRound}
        expanded={expandedSections.has('env')}
        onToggle={() => toggleSection('env')}
        badge={envCount > 0 ? `${envCount}` : undefined}
      >
        <KeyValueEditor
          label="Environment Variables"
          value={data.env ?? {}}
          onChange={(env) => onChange({ env })}
          placeholder={{ key: 'ENV_VAR', value: 'value' }}
          showSecrets
        />
      </Section>

      {/* Section 6: A2A Protocol */}
      <Section
        title="A2A Protocol"
        icon={Zap}
        expanded={expandedSections.has('a2a')}
        onToggle={() => toggleSection('a2a')}
        badge={data.a2a?.enabled ? (a2aSkillCount > 0 ? `${a2aSkillCount} skills` : 'enabled') : undefined}
      >
        <div className="space-y-3">
          {/* A2A Toggle */}
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={() =>
                onChange({ a2a: { enabled: !(data.a2a?.enabled ?? false), skills: data.a2a?.skills } })
              }
              className={cn(
                'relative w-8 h-4 rounded-full transition-colors duration-200',
                data.a2a?.enabled ? 'bg-secondary/40' : 'bg-border/40',
              )}
            >
              <div
                className={cn(
                  'absolute top-0.5 w-3 h-3 rounded-full transition-all duration-200',
                  data.a2a?.enabled ? 'left-4.5 bg-secondary' : 'left-0.5 bg-text-muted',
                )}
              />
            </button>
            <div>
              <span className="text-xs text-text-secondary">Enable A2A</span>
              <p className="text-[10px] text-text-muted">Agent-to-Agent protocol for inter-agent communication</p>
            </div>
          </div>

          {data.a2a?.enabled && (
            <div className="space-y-2 p-3 rounded-xl bg-secondary/[0.03] border border-secondary/10">
              <div className="flex items-center justify-between mb-1">
                <label className="text-[10px] text-secondary/80 uppercase tracking-wider font-medium">A2A Skills</label>
                <button
                  type="button"
                  onClick={() => {
                    const skills = [...(data.a2a?.skills ?? []), { id: '', name: '', description: '' }];
                    onChange({ a2a: { enabled: true, skills } });
                  }}
                  className="flex items-center gap-1 text-[10px] text-secondary hover:text-secondary/80 transition-colors"
                >
                  <Plus size={10} />
                  Add skill
                </button>
              </div>
              {(!data.a2a?.skills || data.a2a.skills.length === 0) && (
                <p className="text-[10px] text-text-muted/60 italic py-1">No A2A skills configured</p>
              )}
              {data.a2a?.skills?.map((skill, i) => (
                <div key={i} className="flex items-start gap-1.5">
                  <div className="flex-1 space-y-1">
                    <input
                      type="text"
                      value={skill.id}
                      onChange={(e) => {
                        const skills = [...(data.a2a?.skills ?? [])];
                        skills[i] = { ...skills[i], id: e.target.value };
                        onChange({ a2a: { enabled: true, skills } });
                      }}
                      placeholder="skill-id"
                      className={cn(inputClass, 'font-mono')}
                    />
                    <input
                      type="text"
                      value={skill.name}
                      onChange={(e) => {
                        const skills = [...(data.a2a?.skills ?? [])];
                        skills[i] = { ...skills[i], name: e.target.value };
                        onChange({ a2a: { enabled: true, skills } });
                      }}
                      placeholder="Skill Name"
                      className={inputClass}
                    />
                    <input
                      type="text"
                      value={skill.description ?? ''}
                      onChange={(e) => {
                        const skills = [...(data.a2a?.skills ?? [])];
                        skills[i] = { ...skills[i], description: e.target.value };
                        onChange({ a2a: { enabled: true, skills } });
                      }}
                      placeholder="Description (optional)"
                      className={inputClass}
                    />
                  </div>
                  <button
                    type="button"
                    onClick={() => {
                      const skills = (data.a2a?.skills ?? []).filter((_, j) => j !== i);
                      onChange({ a2a: { enabled: true, skills } });
                    }}
                    className="p-1 mt-1 text-text-muted hover:text-status-error transition-colors flex-shrink-0"
                  >
                    <X size={12} />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </Section>
    </div>
  );
}
