import { describe, expect, it } from 'vitest';
import { parseTemplateK8sVersion, resolveVKSVersionOptions } from './vks-catalog';

describe('parseTemplateK8sVersion', () => {
  it('parses ubuntu-node-1-36-5', () => {
    expect(parseTemplateK8sVersion('ubuntu-node-1-36-5')).toBe('v1.36.5');
  });
  it('returns null for unrelated templates', () => {
    expect(parseTemplateK8sVersion('ubuntu-2204')).toBeNull();
  });
});

describe('resolveVKSVersionOptions', () => {
  it('intersects published pins with templates', () => {
    const opts = resolveVKSVersionOptions([
      { name: 'ubuntu-node-1-36-5' },
      { name: 'ubuntu-2204' },
    ]);
    expect(opts).toEqual([{ kubernetes_version: 'v1.36.5', template: 'ubuntu-node-1-36-5' }]);
  });
});
