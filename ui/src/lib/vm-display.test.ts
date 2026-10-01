import { describe, expect, it } from 'vitest';
import { effectiveVmState, isVmRunning } from './vm-display';

describe('effectiveVmState', () => {
  it('keeps observed state when power_state matches or is absent', () => {
    expect(effectiveVmState({ state: 'Running' })).toBe('Running');
    expect(effectiveVmState({ state: 'Running', power_state: 'Running' })).toBe('Running');
    expect(effectiveVmState({ state: 'Stopped', power_state: 'Halted' })).toBe('Stopped');
  });

  it('maps operator lag Halted desired + Running observed to Stopping', () => {
    expect(effectiveVmState({ state: 'Running', power_state: 'Halted' })).toBe('Stopping');
  });

  it('maps Running desired + Stopped observed to Starting', () => {
    expect(effectiveVmState({ state: 'Stopped', power_state: 'Running' })).toBe('Starting');
  });

  it('preserves explicit Starting/Stopping over power_state', () => {
    expect(effectiveVmState({ state: 'Stopping', power_state: 'Halted' })).toBe('Stopping');
    expect(effectiveVmState({ state: 'Starting', power_state: 'Running' })).toBe('Starting');
  });

  it('keeps Start/Stop buttons from flipping on hub lag', () => {
    const lag = effectiveVmState({ state: 'Running', power_state: 'Halted' });
    expect(isVmRunning(lag)).toBe(false);
  });
});
