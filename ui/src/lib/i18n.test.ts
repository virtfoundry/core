import { describe, expect, it } from 'vitest';
import { dict } from './i18n';

describe('i18n dictionaries', () => {
  const pt = Object.keys(dict.pt).sort();
  const en = Object.keys(dict.en).sort();

  it('have the same keys in pt and en', () => {
    expect(pt.filter((k) => !(k in dict.en))).toEqual([]);
    expect(en.filter((k) => !(k in dict.pt))).toEqual([]);
  });

  it('translate every status badge label', () => {
    const statuses = ['running', 'stopped', 'starting', 'stopping', 'creating', 'error', 'enabled', 'disabled', 'active', 'inactive'];
    for (const s of statuses) {
      expect(dict.pt[`status.${s}` as keyof typeof dict.pt], `pt status.${s}`).toBeTruthy();
      expect(dict.en[`status.${s}` as keyof typeof dict.en], `en status.${s}`).toBeTruthy();
    }
    expect(dict.pt['status.running']).not.toBe(dict.en['status.running']);
  });
});
