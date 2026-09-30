import type { PlatformVM } from './platform-api';

export type VMsCache = { vms: PlatformVM[] };

/** True when event payload has neither cpu nor memory_mi (thin hub event). */
export function payloadLacksSizing(payload: Record<string, unknown> | undefined): boolean {
  return payload?.cpu === undefined && payload?.memory_mi === undefined;
}

/**
 * Merge-patch a VM row: only overwrite keys that are defined in the payload.
 * Never replace a full row with a thin hub object (name/state only).
 */
export function mergeVmRow(
  existing: PlatformVM | undefined,
  payload: Record<string, unknown>,
): PlatformVM {
  const name = typeof payload.name === 'string' ? payload.name : existing?.name;
  if (!name) {
    throw new Error('vm event payload missing name');
  }

  const base: PlatformVM = existing ?? {
    id: typeof payload.id === 'string' ? payload.id : '',
    name,
    namespace: typeof payload.namespace === 'string' ? payload.namespace : '',
    state: typeof payload.state === 'string' ? payload.state : '',
    cpu: typeof payload.cpu === 'number' ? payload.cpu : 0,
    memory_mi: typeof payload.memory_mi === 'number' ? payload.memory_mi : 0,
  };

  const next: PlatformVM = { ...base };
  for (const [key, value] of Object.entries(payload)) {
    if (value !== undefined) {
      (next as unknown as Record<string, unknown>)[key] = value;
    }
  }
  return next;
}

/**
 * Apply a vm.* event to the React Query list cache shape `{ vms: PlatformVM[] }`.
 * Upserts by `name`; `vm.deleted` removes the row.
 */
export function applyVmEventToCache(
  cache: VMsCache | undefined,
  eventType: string,
  payload: Record<string, unknown> | undefined,
): VMsCache {
  const vms = cache?.vms ?? [];
  const name = typeof payload?.name === 'string' ? payload.name : undefined;
  if (!name || !payload) {
    return cache ?? { vms };
  }

  if (eventType === 'vm.deleted') {
    return { vms: vms.filter((vm) => vm.name !== name) };
  }

  const idx = vms.findIndex((vm) => vm.name === name);
  const existing = idx >= 0 ? vms[idx] : undefined;
  const merged = mergeVmRow(existing, payload);

  if (idx >= 0) {
    const next = vms.slice();
    next[idx] = merged;
    return { vms: next };
  }
  return { vms: [...vms, merged] };
}
