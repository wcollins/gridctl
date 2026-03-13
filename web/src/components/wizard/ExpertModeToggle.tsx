import { FormInput, Code2 } from 'lucide-react';
import { cn } from '../../lib/cn';

interface ExpertModeToggleProps {
  expertMode: boolean;
  onToggle: (enabled: boolean) => void;
}

export function ExpertModeToggle({ expertMode, onToggle }: ExpertModeToggleProps) {
  return (
    <div className="inline-flex items-center bg-surface-elevated/60 border border-border/40 rounded-lg p-0.5">
      <button
        onClick={() => onToggle(false)}
        className={cn(
          'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-all duration-200',
          !expertMode
            ? 'bg-primary/15 text-primary shadow-sm'
            : 'text-text-muted hover:text-text-secondary',
        )}
      >
        <FormInput size={12} />
        Form
      </button>
      <button
        onClick={() => onToggle(true)}
        className={cn(
          'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-all duration-200',
          expertMode
            ? 'bg-primary/15 text-primary shadow-sm'
            : 'text-text-muted hover:text-text-secondary',
        )}
      >
        <Code2 size={12} />
        YAML
      </button>
    </div>
  );
}
