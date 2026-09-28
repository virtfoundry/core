/** Local-only prefs for UI preview (no backend). Cleared with localStorage. */

const FAV_KEY = 'vf_preview_fav_vms';
const PIN_KEY = 'vf_preview_pin_vms';
const TAGS_KEY = 'vf_preview_vm_tags';
const ONBOARD_KEY = 'vf_preview_onboarding';
const CLOUDINIT_KEY = 'vf_preview_cloudinit';
const SPLIT_KEY = 'vf_preview_split_view';
const RECENT_KEY = 'vf_preview_recent';

function readSet(key: string): Set<string> {
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return new Set();
    const arr = JSON.parse(raw) as string[];
    return new Set(Array.isArray(arr) ? arr : []);
  } catch {
    return new Set();
  }
}

function writeSet(key: string, set: Set<string>) {
  localStorage.setItem(key, JSON.stringify([...set]));
}

function readMap(key: string): Record<string, string[]> {
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return {};
    const obj = JSON.parse(raw) as Record<string, string[]>;
    return obj && typeof obj === 'object' ? obj : {};
  } catch {
    return {};
  }
}

function writeMap(key: string, map: Record<string, string[]>) {
  localStorage.setItem(key, JSON.stringify(map));
}

export function getFavoriteVMs(): Set<string> {
  return readSet(FAV_KEY);
}

export function toggleFavoriteVM(name: string): Set<string> {
  const set = readSet(FAV_KEY);
  if (set.has(name)) set.delete(name);
  else set.add(name);
  writeSet(FAV_KEY, set);
  return set;
}

export function getPinnedVMs(): Set<string> {
  return readSet(PIN_KEY);
}

export function togglePinnedVM(name: string): Set<string> {
  const set = readSet(PIN_KEY);
  if (set.has(name)) set.delete(name);
  else set.add(name);
  writeSet(PIN_KEY, set);
  return set;
}

export function getVmTags(name: string): string[] {
  return readMap(TAGS_KEY)[name] || [];
}

export function getAllVmTags(): Record<string, string[]> {
  return readMap(TAGS_KEY);
}

export function setVmTags(name: string, tags: string[]) {
  const map = readMap(TAGS_KEY);
  const cleaned = [...new Set(tags.map((t) => t.trim().toLowerCase()).filter(Boolean))];
  if (cleaned.length === 0) delete map[name];
  else map[name] = cleaned;
  writeMap(TAGS_KEY, map);
}

export type OnboardingState = {
  dismissed: boolean;
  seenTemplate: boolean;
  seenOffering: boolean;
  seenSsh: boolean;
  seenDeploy: boolean;
};

export function getOnboarding(): OnboardingState {
  try {
    const raw = localStorage.getItem(ONBOARD_KEY);
    if (!raw) {
      return { dismissed: false, seenTemplate: false, seenOffering: false, seenSsh: false, seenDeploy: false };
    }
    return { dismissed: false, seenTemplate: false, seenOffering: false, seenSsh: false, seenDeploy: false, ...JSON.parse(raw) };
  } catch {
    return { dismissed: false, seenTemplate: false, seenOffering: false, seenSsh: false, seenDeploy: false };
  }
}

export function saveOnboarding(next: Partial<OnboardingState>) {
  const cur = getOnboarding();
  const merged = { ...cur, ...next };
  localStorage.setItem(ONBOARD_KEY, JSON.stringify(merged));
  return merged;
}

export function getCloudInitDraft(vmName: string): string {
  try {
    const raw = localStorage.getItem(CLOUDINIT_KEY);
    if (!raw) return '';
    const map = JSON.parse(raw) as Record<string, string>;
    return map[vmName] || '';
  } catch {
    return '';
  }
}

export function setCloudInitDraft(vmName: string, yaml: string) {
  let map: Record<string, string> = {};
  try {
    const raw = localStorage.getItem(CLOUDINIT_KEY);
    if (raw) map = JSON.parse(raw) as Record<string, string>;
  } catch {
    map = {};
  }
  map[vmName] = yaml;
  localStorage.setItem(CLOUDINIT_KEY, JSON.stringify(map));
}

export function getSplitView(): boolean {
  return localStorage.getItem(SPLIT_KEY) === '1';
}

export function setSplitView(on: boolean) {
  localStorage.setItem(SPLIT_KEY, on ? '1' : '0');
}

export type RecentAction = {
  id: string;
  label: string;
  path: string;
  at: number;
};

export function pushRecentAction(entry: Omit<RecentAction, 'id' | 'at'>) {
  let list: RecentAction[] = [];
  try {
    const raw = localStorage.getItem(RECENT_KEY);
    if (raw) list = JSON.parse(raw) as RecentAction[];
  } catch {
    list = [];
  }
  const next: RecentAction = {
    id: `${entry.path}-${Date.now()}`,
    label: entry.label,
    path: entry.path,
    at: Date.now(),
  };
  list = [next, ...list.filter((x) => x.path !== entry.path)].slice(0, 5);
  localStorage.setItem(RECENT_KEY, JSON.stringify(list));
  return list;
}

export function getRecentActions(): RecentAction[] {
  try {
    const raw = localStorage.getItem(RECENT_KEY);
    if (!raw) return [];
    const list = JSON.parse(raw) as RecentAction[];
    return Array.isArray(list) ? list.slice(0, 5) : [];
  } catch {
    return [];
  }
}
