import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import { AgentForm } from '../components/wizard/steps/AgentForm';
import type { AgentFormData } from '../lib/yaml-builder';

// Mock useStackStore to provide available MCP servers
vi.mock('../stores/useStackStore', () => ({
  useStackStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({
      mcpServers: [
        { name: 'filesystem-server' },
        { name: 'postgres-server' },
      ],
    }),
}));

function defaultData(overrides?: Partial<AgentFormData>): AgentFormData {
  return { name: '', agentType: 'container', ...overrides };
}

const noop = (_data: Partial<AgentFormData>) => {};

describe('AgentForm', () => {
  let onChange: typeof noop;

  beforeEach(() => {
    onChange = vi.fn<typeof noop>();
  });

  it('renders all 6 accordion sections', () => {
    render(<AgentForm data={defaultData()} onChange={onChange} />);
    expect(screen.getByText('Identity')).toBeInTheDocument();
    expect(screen.getByText('Agent Type')).toBeInTheDocument();
    expect(screen.getByText('Configuration')).toBeInTheDocument();
    expect(screen.getByText('Tool Access')).toBeInTheDocument();
    expect(screen.getByText('Environment & Secrets')).toBeInTheDocument();
    expect(screen.getByText('A2A Protocol')).toBeInTheDocument();
  });

  it('renders 2 agent type options', () => {
    render(<AgentForm data={defaultData()} onChange={onChange} />);
    // Container appears in both the badge and radio card
    expect(screen.getAllByText('Container').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('Headless Runtime')).toBeInTheDocument();
  });

  it('enforces kebab-case on name input', () => {
    render(<AgentForm data={defaultData()} onChange={onChange} />);
    const nameInput = screen.getByPlaceholderText('my-agent');
    fireEvent.change(nameInput, { target: { value: 'My Agent Name' } });
    expect(onChange).toHaveBeenCalledWith({ name: 'my-agent-name' });
  });

  it('shows image field for container type', () => {
    render(<AgentForm data={defaultData({ agentType: 'container' })} onChange={onChange} />);
    expect(screen.getByPlaceholderText('my-agent:latest')).toBeInTheDocument();
  });

  it('shows runtime and prompt fields for headless type', () => {
    render(<AgentForm data={defaultData({ agentType: 'headless' })} onChange={onChange} />);
    expect(screen.getByText('Select runtime')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Agent system prompt...')).toBeInTheDocument();
  });

  it('hides image field for headless type', () => {
    render(<AgentForm data={defaultData({ agentType: 'headless' })} onChange={onChange} />);
    expect(screen.queryByPlaceholderText('my-agent:latest')).not.toBeInTheDocument();
  });

  it('hides runtime/prompt fields for container type', () => {
    render(<AgentForm data={defaultData({ agentType: 'container' })} onChange={onChange} />);
    expect(screen.queryByPlaceholderText('Agent system prompt...')).not.toBeInTheDocument();
  });

  it('calls onChange when switching agent type', () => {
    render(<AgentForm data={defaultData()} onChange={onChange} />);
    fireEvent.click(screen.getByText('Headless Runtime'));
    expect(onChange).toHaveBeenCalledWith({ agentType: 'headless' });
  });

  it('shows available MCP servers in tool access section', () => {
    render(<AgentForm data={defaultData()} onChange={onChange} />);
    // Expand Tool Access section
    fireEvent.click(screen.getByText('Tool Access'));
    expect(screen.getByText('filesystem-server')).toBeInTheDocument();
    expect(screen.getByText('postgres-server')).toBeInTheDocument();
  });

  it('toggles MCP server selection', () => {
    render(<AgentForm data={defaultData()} onChange={onChange} />);
    fireEvent.click(screen.getByText('Tool Access'));
    fireEvent.click(screen.getByText('filesystem-server'));
    expect(onChange).toHaveBeenCalledWith({
      uses: [{ server: 'filesystem-server' }],
    });
  });

  it('shows env badge count when env vars are set', () => {
    render(
      <AgentForm
        data={defaultData({ env: { API_KEY: 'test', DB_HOST: 'localhost' } })}
        onChange={onChange}
      />,
    );
    expect(screen.getByText('2')).toBeInTheDocument();
  });

  it('shows A2A toggle', () => {
    render(<AgentForm data={defaultData()} onChange={onChange} />);
    fireEvent.click(screen.getByText('A2A Protocol'));
    expect(screen.getByText('Enable A2A')).toBeInTheDocument();
  });

  it('shows A2A skills section when enabled', () => {
    render(
      <AgentForm
        data={defaultData({ a2a: { enabled: true, skills: [] } })}
        onChange={onChange}
      />,
    );
    fireEvent.click(screen.getByText('A2A Protocol'));
    expect(screen.getByText('A2A Skills')).toBeInTheDocument();
    expect(screen.getByText('Add skill')).toBeInTheDocument();
  });

  it('displays field errors when provided', () => {
    render(
      <AgentForm
        data={defaultData()}
        onChange={onChange}
        errors={{ name: 'Name is required' }}
      />,
    );
    expect(screen.getByText('Name is required')).toBeInTheDocument();
  });

  it('shows source configuration for container type', () => {
    render(<AgentForm data={defaultData({ agentType: 'container' })} onChange={onChange} />);
    expect(screen.getByText('Or build from source')).toBeInTheDocument();
    expect(screen.getByText('Git')).toBeInTheDocument();
    expect(screen.getByText('Local')).toBeInTheDocument();
  });

  it('shows uses badge count when servers are selected', () => {
    render(
      <AgentForm
        data={defaultData({ uses: [{ server: 'filesystem-server' }] })}
        onChange={onChange}
      />,
    );
    expect(screen.getByText('1')).toBeInTheDocument();
  });
});
