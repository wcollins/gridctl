import { useMemo } from 'react';
import Fuse from 'fuse.js';
import type { AgentSkill } from '../types';

const FUSE_OPTIONS: Fuse.IFuseOptions<AgentSkill> = {
  keys: ['name', 'description'],
  threshold: 0.4,
};

export function useFuzzySearch(skills: AgentSkill[], query: string): AgentSkill[] {
  const fuse = useMemo(() => new Fuse(skills, FUSE_OPTIONS), [skills]);

  return useMemo(() => {
    if (!query.trim()) return skills;
    return fuse.search(query).map((r) => r.item);
  }, [fuse, query, skills]);
}
