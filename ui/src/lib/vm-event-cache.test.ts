import { describe, expect, it } from 'vitest';
import {
  applyVmEventToCache,
  mergeVmRow,
  payloadLacksSizing,
  type VMsCache,
} from './vm-event-cache';
import type { PlatformVM } from './platform-api';

const runningRow: PlatformVM = {
  id: 'vm-1',
  name: 'web-01',
  namespace: 'tenant-a',
  state: 'Running',
  ip: '10.0.0.5',
  cpu: 2,
  memory_mi: 4096,
};

describe('mergeVmRow', () => {
  it('keeps ip/cpu/memory_mi when event only patches state', () => {
    const merged = mergeVmRow(runningRow, { name: 'web-01', state: 'Stopped' });
    expect(merged).toEqual({
      ...runningRow,
      state: 'Stopped',
    });
  });

  it('does not overwrite with undefined keys from sparse payload', () => {
    const merged = mergeVmRow(runningRow, {
      name: 'web-01',
      state: 'Stopped',
      ip: undefined,
    });
    expect(merged.ip).toBe('10.0.0.5');
    expect(merged.state).toBe('Stopped');
  });
});

describe('applyVmEventToCache', () => {
  it('upserts by name with merge-patch on vm.updated', () => {
    const cache: VMsCache = { vms: [runningRow] };
    const next = applyVmEventToCache(cache, 'vm.updated', {
      name: 'web-01',
      state: 'Stopped',
    });
    expect(next.vms).toHaveLength(1);
    expect(next.vms[0].state).toBe('Stopped');
    expect(next.vms[0].ip).toBe('10.0.0.5');
    expect(next.vms[0].cpu).toBe(2);
    expect(next.vms[0].memory_mi).toBe(4096);
  });

  it('appends on vm.created when name is new', () => {
    const cache: VMsCache = { vms: [runningRow] };
    const next = applyVmEventToCache(cache, 'vm.created', {
      name: 'db-01',
      state: 'Pending',
      cpu: 4,
      memory_mi: 8192,
    });
    expect(next.vms).toHaveLength(2);
    expect(next.vms[1].name).toBe('db-01');
    expect(next.vms[1].cpu).toBe(4);
  });

  it('filters out by name on vm.deleted', () => {
    const cache: VMsCache = {
      vms: [runningRow, { ...runningRow, id: 'vm-2', name: 'db-01' }],
    };
    const next = applyVmEventToCache(cache, 'vm.deleted', { name: 'web-01' });
    expect(next.vms.map((v) => v.name)).toEqual(['db-01']);
  });

  it('returns empty list cache when deleting the only row', () => {
    const next = applyVmEventToCache({ vms: [runningRow] }, 'vm.deleted', {
      name: 'web-01',
    });
    expect(next.vms).toEqual([]);
  });
});

describe('payloadLacksSizing', () => {
  it('is true when cpu and memory_mi are absent', () => {
    expect(payloadLacksSizing({ name: 'web-01', state: 'Running' })).toBe(true);
  });

  it('is false when either sizing field is present', () => {
    expect(payloadLacksSizing({ name: 'web-01', cpu: 2 })).toBe(false);
    expect(payloadLacksSizing({ name: 'web-01', memory_mi: 1024 })).toBe(false);
  });
});
